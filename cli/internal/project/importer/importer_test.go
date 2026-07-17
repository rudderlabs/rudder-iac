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
			name:  "merge allows pending deletions (conflicts caught later in precheck)",
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

func TestInitNamer(t *testing.T) {
	t.Parallel()

	graphWith := func(id, resourceType string) *resources.Graph {
		g := resources.NewGraph()
		g.AddResource(resources.NewResource(id, resourceType, resources.ResourceData{}, []string{}))
		return g
	}

	t.Run("reserves IDs from both target and source graphs", func(t *testing.T) {
		t.Parallel()

		// "checkout" is only in target; "legacy" is only in source (a pending
		// local deletion whose remote twin still holds the ID). Both must be
		// reserved so a fresh mint of either name gets suffixed.
		target := graphWith("checkout", "category")
		source := graphWith("legacy", "category")

		idNamer, err := initNamer(target, source)
		require.NoError(t, err)

		gotTarget, err := idNamer.Name(namer.ScopeName{Name: "checkout", Scope: "category"})
		require.NoError(t, err)
		assert.Equal(t, "checkout-1", gotTarget, "target ID must stay reserved")

		gotSource, err := idNamer.Name(namer.ScopeName{Name: "legacy", Scope: "category"})
		require.NoError(t, err)
		assert.Equal(t, "legacy-1", gotSource, "source-only ID must stay reserved (pending deletion)")
	})

	t.Run("deduplicates an ID present in both graphs", func(t *testing.T) {
		t.Parallel()

		// Same URN in both graphs must load once, not error as a duplicate.
		target := graphWith("checkout", "category")
		source := graphWith("checkout", "category")

		_, err := initNamer(target, source)
		require.NoError(t, err)
	})
}

func TestCheckPendingDeleteConflicts(t *testing.T) {
	t.Parallel()

	// An event importable references a category by remote ID. The lister
	// surfaces that reference for the precheck.
	eventLister := func(refs []importmatcher.Ref) importmatcher.RefLister {
		return importmatcher.RefLister{
			ResourceType: "event",
			Refs: func(*resources.RemoteResource) []importmatcher.Ref {
				return refs
			},
		}
	}

	collectionWith := func(resourceType string, rs ...*resources.RemoteResource) *resources.RemoteResources {
		c := resources.NewRemoteResources()
		m := make(map[string]*resources.RemoteResource, len(rs))
		for _, r := range rs {
			m[r.ID] = r
		}
		c.Set(resourceType, m)
		return c
	}

	importableEvent := collectionWith("event", &resources.RemoteResource{ID: "evt_1", ExternalID: "signed-up"})

	t.Run("references a managed resource absent from local graph -> conflict", func(t *testing.T) {
		t.Parallel()

		// cat_legacy is managed remotely (ExternalID set) but its category:legacy
		// URN is absent from the local graph: a pending, unapplied deletion.
		remote := collectionWith("category", &resources.RemoteResource{ID: "cat_legacy", ExternalID: "legacy"})
		listers := []importmatcher.RefLister{eventLister([]importmatcher.Ref{{EntityType: "category", RemoteID: "cat_legacy"}})}

		err := checkPendingDeleteConflicts(listers, importableEvent, remote, resources.NewGraph())

		require.ErrorIs(t, err, ErrPendingDeleteConflict)
		assert.ErrorContains(t, err, "category:legacy")
		assert.ErrorContains(t, err, "cat_legacy")
	})

	t.Run("referenced entity is itself being imported -> no conflict", func(t *testing.T) {
		t.Parallel()

		// The category is in the importable collection (imported together), so it
		// resolves fine at format time.
		importable := collectionWith("event", &resources.RemoteResource{ID: "evt_1", ExternalID: "signed-up"})
		importable.Set("category", map[string]*resources.RemoteResource{
			"cat_new": {ID: "cat_new", ExternalID: "new-category"},
		})
		listers := []importmatcher.RefLister{eventLister([]importmatcher.Ref{{EntityType: "category", RemoteID: "cat_new"}})}

		err := checkPendingDeleteConflicts(listers, importable, resources.NewRemoteResources(), resources.NewGraph())

		assert.NoError(t, err)
	})

	t.Run("references an unknown remote id -> no conflict (surfaced later)", func(t *testing.T) {
		t.Parallel()

		// Not in importable, not in managed remote: an unknown ID. The precheck
		// stays silent; formatting surfaces it as it does today.
		listers := []importmatcher.RefLister{eventLister([]importmatcher.Ref{{EntityType: "category", RemoteID: "cat_ghost"}})}

		err := checkPendingDeleteConflicts(listers, importableEvent, resources.NewRemoteResources(), resources.NewGraph())

		assert.NoError(t, err)
	})

	t.Run("references a managed resource present in local graph -> no conflict", func(t *testing.T) {
		t.Parallel()

		remote := collectionWith("category", &resources.RemoteResource{ID: "cat_ok", ExternalID: "checkout"})
		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource("checkout", "category", resources.ResourceData{"name": "Checkout"}, []string{}))
		listers := []importmatcher.RefLister{eventLister([]importmatcher.Ref{{EntityType: "category", RemoteID: "cat_ok"}})}

		err := checkPendingDeleteConflicts(listers, importableEvent, remote, graph)

		assert.NoError(t, err)
	})

	t.Run("all conflicts are reported together", func(t *testing.T) {
		t.Parallel()

		remote := collectionWith("category",
			&resources.RemoteResource{ID: "cat_a", ExternalID: "legacy-a"},
			&resources.RemoteResource{ID: "cat_b", ExternalID: "legacy-b"},
		)
		listers := []importmatcher.RefLister{eventLister([]importmatcher.Ref{
			{EntityType: "category", RemoteID: "cat_a"},
			{EntityType: "category", RemoteID: "cat_b"},
		})}

		err := checkPendingDeleteConflicts(listers, importableEvent, remote, resources.NewGraph())

		require.ErrorIs(t, err, ErrPendingDeleteConflict)
		assert.ErrorContains(t, err, "category:legacy-a")
		assert.ErrorContains(t, err, "category:legacy-b")
	})

	t.Run("no listers -> no conflict", func(t *testing.T) {
		t.Parallel()

		err := checkPendingDeleteConflicts(nil, importableEvent, resources.NewRemoteResources(), resources.NewGraph())

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

func (p *stubImportProvider) ImportableRefs() []importmatcher.RefLister {
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
