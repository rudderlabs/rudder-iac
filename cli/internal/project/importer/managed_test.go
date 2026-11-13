package importer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// stubManagedProvider records the collection ManagedExport asks it to format.
type stubManagedProvider struct {
	stubImportProvider
	formatted *resources.RemoteResources
}

func (p *stubManagedProvider) LoadManaged(context.Context) (*resources.RemoteResources, []string, error) {
	return p.importable, nil, nil
}

func (p *stubManagedProvider) FormatForExport(
	collection *resources.RemoteResources,
	idNamer namer.Namer,
	r resolver.ReferenceResolver,
) ([]writer.FormattableEntity, []importmanifest.ImportEntry, error) {
	p.formatted = collection
	return p.stubImportProvider.FormatForExport(collection, idNamer, r)
}

// twoManaged holds the managed sources my-src and other-src.
func twoManaged() *resources.RemoteResources {
	c := resources.NewRemoteResources()
	c.Set("source", map[string]*resources.RemoteResource{
		"rid-1": {ID: "rid-1", ExternalID: "my-src"},
		"rid-2": {ID: "rid-2", ExternalID: "other-src"},
	})
	return c
}

func projectWith(t *testing.T, ids ...string) *stubProject {
	graph := resources.NewGraph()
	for _, id := range ids {
		graph.AddResource(resources.NewResource(id, "source", resources.ResourceData{}, nil))
	}
	return &stubProject{location: t.TempDir(), graph: graph}
}

func TestManagedExport(t *testing.T) {
	t.Run("writes specs under managed/", func(t *testing.T) {
		dir := t.TempDir()
		entities, entries := exportFixture()

		err := ManagedExport(context.Background(), &stubProject{location: dir, graph: resources.NewGraph()},
			&stubManagedProvider{stubImportProvider: stubImportProvider{importable: importableCollection(), entities: entities, entries: entries}},
			ManagedExportOptions{})
		require.NoError(t, err)

		_, err = os.Stat(filepath.Join(dir, ManagedDir, "sources", "my-src.yaml"))
		assert.NoError(t, err)
		_, err = os.Stat(filepath.Join(dir, ImportedDir))
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("drops resources without an externalId", func(t *testing.T) {
		managed := importableCollection()
		managed.Set("destination", map[string]*resources.RemoteResource{"rid-2": {ID: "rid-2"}})
		p := &stubManagedProvider{stubImportProvider: stubImportProvider{importable: managed}}

		require.NoError(t, ManagedExport(context.Background(), projectWith(t), p, ManagedExportOptions{}))
		assert.Equal(t, 1, p.formatted.Len())
		_, ok := p.formatted.GetByID("source", "rid-1")
		assert.True(t, ok)
	})

	t.Run("refuses a project that has specs", func(t *testing.T) {
		err := ManagedExport(context.Background(), projectWith(t, "my-src"),
			&stubManagedProvider{stubImportProvider: stubImportProvider{importable: twoManaged()}}, ManagedExportOptions{})
		assert.ErrorIs(t, err, ErrProjectNotEmpty)
	})

	t.Run("skip-existing writes only what the project lacks", func(t *testing.T) {
		p := &stubManagedProvider{stubImportProvider: stubImportProvider{importable: twoManaged()}}

		require.NoError(t, ManagedExport(context.Background(), projectWith(t, "my-src"), p, ManagedExportOptions{SkipExisting: true}))
		require.Equal(t, 1, p.formatted.Len())
		_, ok := p.formatted.GetByID("source", "rid-2")
		assert.True(t, ok, "other-src is not in the project, so it is written")
	})

	t.Run("skip-existing with everything local writes nothing", func(t *testing.T) {
		project := projectWith(t, "my-src", "other-src")
		p := &stubManagedProvider{stubImportProvider: stubImportProvider{importable: twoManaged()}}

		require.NoError(t, ManagedExport(context.Background(), project, p, ManagedExportOptions{SkipExisting: true}))
		assert.Nil(t, p.formatted, "nothing is formatted")
		_, err := os.Stat(filepath.Join(project.location, ManagedDir))
		assert.True(t, os.IsNotExist(err))
	})
}
