package importer

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestCheckSyncStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		diff    *differ.Diff
		merge   bool
		wantErr bool
	}{
		{
			name:  "no merge, synced project",
			diff:  &differ.Diff{},
			merge: false,
		},
		{
			name:    "no merge, pending changes block import",
			diff:    &differ.Diff{NewResources: []string{"category:checkout"}},
			merge:   false,
			wantErr: true,
		},
		{
			name:  "merge allows pending additions",
			diff:  &differ.Diff{NewResources: []string{"category:checkout"}},
			merge: true,
		},
		{
			name:    "merge still blocks pending deletions",
			diff:    &differ.Diff{RemovedResources: []string{"category:legacy"}},
			merge:   true,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := checkSyncStatus(tc.diff, tc.merge)

			if !tc.wantErr {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, ErrProjectNotSynced)
		})
	}
}

type stubMatcherProvider struct {
	matchers []importmatcher.Matcher
}

func (s stubMatcherProvider) ResourceMatchers() []importmatcher.Matcher {
	return s.matchers
}

func TestMarkMatchedWith(t *testing.T) {
	t.Parallel()

	local := resources.NewResource("checkout", "category", resources.ResourceData{"name": "Checkout"}, []string{})
	localGraph := resources.NewGraph()
	localGraph.AddResource(local)

	// Any category remote matches the single local category.
	matcher := importmatcher.Matcher{
		ResourceType: "category",
		Match: func(_ importmatcher.Scope, _ *resources.RemoteResource) *resources.Resource {
			return local
		},
	}
	stub := stubMatcherProvider{matchers: []importmatcher.Matcher{matcher}}

	importableWith := func(rs ...*resources.RemoteResource) *resources.RemoteResources {
		collection := resources.NewRemoteResources()
		m := make(map[string]*resources.RemoteResource, len(rs))
		for _, r := range rs {
			m[r.ID] = r
		}
		collection.Set("category", m)
		return collection
	}

	t.Run("single match succeeds", func(t *testing.T) {
		t.Parallel()

		importable := importableWith(&resources.RemoteResource{ID: "cat_a", Reference: "#category:checkout-1"})

		err := markMatchedWith(stub, resources.NewGraph(), localGraph, importable)

		assert.NoError(t, err)
	})

	t.Run("multiple remotes matching one local fail fast", func(t *testing.T) {
		t.Parallel()

		importable := importableWith(
			&resources.RemoteResource{ID: "cat_a", Reference: "#category:checkout-1"},
			&resources.RemoteResource{ID: "cat_b", Reference: "#category:checkout-2"},
		)

		err := markMatchedWith(stub, resources.NewGraph(), localGraph, importable)

		require.ErrorIs(t, err, ErrAmbiguousMatch)
		assert.ErrorContains(t, err, `local resource "category:checkout" is matched by multiple remotes "cat_a" and "cat_b"`)
	})

	t.Run("no matcher provider is a no-op", func(t *testing.T) {
		t.Parallel()

		importable := importableWith(&resources.RemoteResource{ID: "cat_a", Reference: "#category:checkout-1"})

		err := markMatchedWith(stubMatcherProvider{}, resources.NewGraph(), localGraph, importable)

		assert.NoError(t, err)
	})
}

func enableImportMerge(t *testing.T) {
	t.Helper()
	prevExp, prevFlag := viper.Get("experimental"), viper.Get("flags.importMerge")
	viper.Set("experimental", true)
	viper.Set("flags.importMerge", true)
	t.Cleanup(func() {
		viper.Set("experimental", prevExp)
		viper.Set("flags.importMerge", prevFlag)
	})
}

type stubProject struct {
	location string
	graph    *resources.Graph
}

func (p *stubProject) ResourceGraph() (*resources.Graph, error) {
	return p.graph, nil
}

func (p *stubProject) Location() string {
	return p.location
}

type stubImportProvider struct {
	importable *resources.RemoteResources
	entities   []writer.FormattableEntity
	entries    []importmanifest.ImportEntry
}

func (p *stubImportProvider) LoadResourcesFromRemote(context.Context) (*resources.RemoteResources, error) {
	return resources.NewRemoteResources(), nil
}

func (p *stubImportProvider) MapRemoteToState(*resources.RemoteResources) (*state.State, error) {
	return state.EmptyState(), nil
}

func (p *stubImportProvider) LoadImportable(context.Context, namer.Namer) (*resources.RemoteResources, error) {
	return p.importable, nil
}

func (p *stubImportProvider) FormatForExport(
	*resources.RemoteResources,
	namer.Namer,
	resolver.ReferenceResolver,
) ([]writer.FormattableEntity, []importmanifest.ImportEntry, error) {
	return p.entities, p.entries, nil
}

func (p *stubImportProvider) ResourceMatchers() []importmatcher.Matcher {
	return nil
}

func importableCollection() *resources.RemoteResources {
	c := resources.NewRemoteResources()
	c.Set("source", map[string]*resources.RemoteResource{
		"rid-1": {ID: "rid-1", ExternalID: "my-src"},
	})
	return c
}

func exportFixture() ([]writer.FormattableEntity, []importmanifest.ImportEntry) {
	entities := []writer.FormattableEntity{{
		Content: &specs.Spec{
			Version:  specs.SpecVersionV1,
			Kind:     "source",
			Metadata: map[string]any{"name": "my-src"},
			Spec:     map[string]any{"id": "my-src"},
		},
		RelativePath: "sources/my-src.yaml",
	}}
	entries := []importmanifest.ImportEntry{{
		WorkspaceID: "ws-1",
		URN:         "source:my-src",
		RemoteID:    "rid-1",
	}}
	return entities, entries
}

func TestWorkspaceImport_SkipsManifestWhenFlagOff(t *testing.T) {
	dir := t.TempDir()
	entities, entries := exportFixture()

	err := WorkspaceImport(context.Background(), &stubProject{
		location: dir,
		graph:    resources.NewGraph(),
	}, &stubImportProvider{
		importable: importableCollection(),
		entities:   entities,
		entries:    entries,
	}, ImportOptions{})
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, ImportedDir, importmanifest.FileName))
	assert.True(t, os.IsNotExist(err), "import-manifest.yaml must not be written when importMerge is off")
}

func TestWorkspaceImport_WritesManifestWhenFlagOn(t *testing.T) {
	enableImportMerge(t)

	dir := t.TempDir()
	entities, entries := exportFixture()

	err := WorkspaceImport(context.Background(), &stubProject{
		location: dir,
		graph:    resources.NewGraph(),
	}, &stubImportProvider{
		importable: importableCollection(),
		entities:   entities,
		entries:    entries,
	}, ImportOptions{})
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, ImportedDir, importmanifest.FileName))
	assert.NoError(t, err, "import-manifest.yaml must be written when importMerge is on")
}

// A re-run merge import re-matches resources an earlier manifest already
// linked; re-emitting them would duplicate their URNs across manifest files.
func TestWorkspaceImport_OmitsManifestEntriesForLinkedResources(t *testing.T) {
	enableImportMerge(t)

	dir := t.TempDir()
	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource("linked", "source", resources.ResourceData{}, nil,
		resources.WithResourceImportMetadata("rid-linked", "ws-1")))
	graph.AddResource(resources.NewResource("other-ws", "source", resources.ResourceData{}, nil,
		resources.WithResourceImportMetadata("rid-other", "ws-2")))
	graph.AddResource(resources.NewResource("unlinked", "source", resources.ResourceData{}, nil))

	entries := []importmanifest.ImportEntry{
		{WorkspaceID: "ws-1", URN: "source:linked", RemoteID: "rid-linked"},
		{WorkspaceID: "ws-1", URN: "source:other-ws", RemoteID: "rid-other-ws"},
		{WorkspaceID: "ws-1", URN: "source:unlinked", RemoteID: "rid-unlinked"},
		{WorkspaceID: "ws-1", URN: "source:my-src", RemoteID: "rid-1"},
	}

	err := WorkspaceImport(context.Background(), &stubProject{
		location: dir,
		graph:    graph,
	}, &stubImportProvider{
		importable: importableCollection(),
		entries:    entries,
	}, ImportOptions{Merge: true})
	require.NoError(t, err)

	raw, err := os.ReadFile(filepath.Join(dir, ImportedDir, importmanifest.FileName))
	require.NoError(t, err)
	var spec specs.Spec
	require.NoError(t, yaml.Unmarshal(raw, &spec))
	assert.Equal(t, importmanifest.BuildSpec(entries[1:]), &spec)
}

func TestWorkspaceImport_NothingNewToImport(t *testing.T) {
	enableImportMerge(t)

	dir := t.TempDir()
	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource("linked", "source", resources.ResourceData{}, nil,
		resources.WithResourceImportMetadata("rid-linked", "ws-1")))

	stdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	err = WorkspaceImport(context.Background(), &stubProject{
		location: dir,
		graph:    graph,
	}, &stubImportProvider{
		importable: importableCollection(),
		entries: []importmanifest.ImportEntry{
			{WorkspaceID: "ws-1", URN: "source:linked", RemoteID: "rid-linked"},
		},
	}, ImportOptions{Merge: true})
	os.Stdout = stdout
	require.NoError(t, err)
	require.NoError(t, w.Close())

	out, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "Nothing new to import\n", string(out))

	_, err = os.Stat(filepath.Join(dir, ImportedDir))
	assert.True(t, os.IsNotExist(err), "nothing must be written when every entry is already linked")
}
