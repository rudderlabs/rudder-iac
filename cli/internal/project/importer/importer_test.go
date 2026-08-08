package importer

import (
	"context"
	"fmt"
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
			name:  "merge allows pending deletions",
			diff:  &differ.Diff{RemovedResources: []string{"category:legacy"}},
			merge: true,
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
	remote      *resources.RemoteResources
	remoteState *state.State
	importable  *resources.RemoteResources
	entities    []writer.FormattableEntity
	entries     []importmanifest.ImportEntry
	format      func(*resources.RemoteResources, namer.Namer, resolver.ReferenceResolver) ([]writer.FormattableEntity, []importmanifest.ImportEntry, error)
}

func (p *stubImportProvider) LoadResourcesFromRemote(context.Context) (*resources.RemoteResources, error) {
	if p.remote != nil {
		return p.remote, nil
	}
	return resources.NewRemoteResources(), nil
}

func (p *stubImportProvider) MapRemoteToState(*resources.RemoteResources) (*state.State, error) {
	if p.remoteState != nil {
		return p.remoteState, nil
	}
	return state.EmptyState(), nil
}

func (p *stubImportProvider) LoadImportable(context.Context, namer.Namer) (*resources.RemoteResources, error) {
	return p.importable, nil
}

func (p *stubImportProvider) FormatForExport(
	collection *resources.RemoteResources,
	idNamer namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, []importmanifest.ImportEntry, error) {
	if p.format != nil {
		return p.format(collection, idNamer, inputResolver)
	}
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

func TestInitNamer_ReservesTargetAndSourceGraphIDs(t *testing.T) {
	targetGraph := resources.NewGraph()
	targetGraph.AddResource(resources.NewResource("existing", "category", resources.ResourceData{}, nil))

	sourceGraph := resources.NewGraph()
	sourceGraph.AddResource(resources.NewResource("existing", "category", resources.ResourceData{}, nil))
	sourceGraph.AddResource(resources.NewResource("pending-delete", "category", resources.ResourceData{}, nil))

	idNamer, err := initNamer(targetGraph, sourceGraph)
	require.NoError(t, err)

	existingName, err := idNamer.Name(namer.ScopeName{Name: "Existing", Scope: "category"})
	require.NoError(t, err)
	pendingDeleteName, err := idNamer.Name(namer.ScopeName{Name: "Pending Delete", Scope: "category"})
	require.NoError(t, err)
	freshName, err := idNamer.Name(namer.ScopeName{Name: "Fresh", Scope: "category"})
	require.NoError(t, err)

	assert.Equal(t, "existing-1", existingName)
	assert.Equal(t, "pending-delete-1", pendingDeleteName)
	assert.Equal(t, "fresh", freshName)
}

func TestWorkspaceImport_MergePendingDeleteReferenceConflictWritesNoFiles(t *testing.T) {
	dir := t.TempDir()
	remote := pendingDeleteRemoteCollection()
	importable := resources.NewRemoteResources()
	importable.Set("source", map[string]*resources.RemoteResource{
		"src_remote": {ID: "src_remote", ExternalID: "checkout"},
	})

	err := WorkspaceImport(context.Background(), &stubProject{
		location: dir,
		graph:    resources.NewGraph(),
	}, &stubImportProvider{
		remote:      remote,
		remoteState: pendingDeleteRemoteState(),
		importable:  importable,
		format: func(collection *resources.RemoteResources, _ namer.Namer, inputResolver resolver.ReferenceResolver) ([]writer.FormattableEntity, []importmanifest.ImportEntry, error) {
			importing, ok := collection.GetByID("source", "src_remote")
			require.True(t, ok)

			_, err := inputResolver.ResolveToReference("category", "cat_deleted")
			if err != nil {
				return nil, nil, fmt.Errorf("cannot import %s: %w", resources.URN(importing.ExternalID, "source"), err)
			}
			return nil, nil, nil
		},
	}, ImportOptions{Merge: true})

	require.ErrorIs(t, err, resolver.ErrPendingDeleteConflict)
	assert.Contains(t, err.Error(), "source:checkout")
	assert.Contains(t, err.Error(), "category:legacy-category")
	assert.Contains(t, err.Error(), "cat_deleted")
	assertNoImportedFiles(t, dir)
}

func TestWorkspaceImport_MergeAllowsUnrelatedPendingDeletions(t *testing.T) {
	dir := t.TempDir()
	entities, entries := exportFixture()

	err := WorkspaceImport(context.Background(), &stubProject{
		location: dir,
		graph:    resources.NewGraph(),
	}, &stubImportProvider{
		remote:      pendingDeleteRemoteCollection(),
		remoteState: pendingDeleteRemoteState(),
		importable:  importableCollection(),
		entities:    entities,
		entries:     entries,
	}, ImportOptions{Merge: true})

	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(dir, ImportedDir, "sources", "my-src.yaml"))
	assert.NoError(t, err)
}

func pendingDeleteRemoteCollection() *resources.RemoteResources {
	remote := resources.NewRemoteResources()
	remote.Set("category", map[string]*resources.RemoteResource{
		"cat_deleted": {
			ID:         "cat_deleted",
			ExternalID: "legacy-category",
		},
	})
	return remote
}

func pendingDeleteRemoteState() *state.State {
	remoteState := state.EmptyState()
	remoteState.AddResource(&state.ResourceState{
		ID:     "legacy-category",
		Type:   "category",
		Input:  map[string]any{},
		Output: map[string]any{},
	})
	return remoteState
}

func assertNoImportedFiles(t *testing.T, dir string) {
	t.Helper()

	entries, err := os.ReadDir(filepath.Join(dir, ImportedDir))
	if os.IsNotExist(err) {
		return
	}
	require.NoError(t, err)
	assert.Empty(t, entries)
}
