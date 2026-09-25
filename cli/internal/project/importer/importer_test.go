package importer

import (
	"context"
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

	_, err := WorkspaceImport(context.Background(), &stubProject{
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

	_, err := WorkspaceImport(context.Background(), &stubProject{
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

func TestImportSummary(t *testing.T) {
	const applyHint = "\nThe imported resources are not managed by the CLI yet. Run `rudder-cli apply` to start managing them.\n"

	tests := []struct {
		name       string
		importable map[string]map[string]*resources.RemoteResource
		expected   string
	}{
		{
			name: "plain import omits the merged section",
			importable: map[string]map[string]*resources.RemoteResource{
				"source":              {"rid-1": {ID: "rid-1", ExternalID: "my-src"}},
				"event-stream-source": {"rid-2": {ID: "rid-2", ExternalID: "web"}},
			},
			expected: `Imported 2 resources into imported/:
  event-stream-source  1
  source               1
` + applyHint,
		},
		{
			name: "merge lists merged resources by local URN",
			importable: map[string]map[string]*resources.RemoteResource{
				"source": {"rid-1": {ID: "rid-1", ExternalID: "my-src"}},
				"event-stream-source": {
					"rid-2": {ID: "rid-2", ExternalID: "web"},
					"rid-3": {ID: "rid-3", ExternalID: "ios"},
					"rid-4": {ID: "rid-4", ExternalID: "android", MatchedWith: resources.NewResource("android", "event-stream-source", nil, nil)},
				},
				"tracking-plan": {"tp-1": {ID: "tp-1", ExternalID: "checkout", MatchedWith: resources.NewResource("checkout", "tracking-plan", nil, nil)}},
			},
			expected: `Imported 3 resources into imported/:
  event-stream-source  2
  source               1
Merged 2 remote resources into existing local resources:
  event-stream-source:android  <- remote rid-4
  tracking-plan:checkout       <- remote tp-1
` + applyHint,
		},
		{
			name: "everything merged still prints the imported total",
			importable: map[string]map[string]*resources.RemoteResource{
				"tracking-plan": {"tp-1": {ID: "tp-1", ExternalID: "checkout", MatchedWith: resources.NewResource("checkout", "tracking-plan", nil, nil)}},
			},
			expected: `Imported 0 resources into imported/:
Merged 1 remote resources into existing local resources:
  tracking-plan:checkout  <- remote tp-1
` + applyHint,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			importable := resources.NewRemoteResources()
			for resourceType, rs := range tt.importable {
				importable.Set(resourceType, rs)
			}

			assert.Equal(t, tt.expected, importSummary(importable))
		})
	}
}
