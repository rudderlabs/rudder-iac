package importer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

type stubManagedProvider struct {
	stubImportProvider
}

func (p *stubManagedProvider) LoadManaged(context.Context) (*resources.RemoteResources, []string, error) {
	return p.importable, nil, nil
}

func TestManagedExport(t *testing.T) {
	t.Run("writes specs under managed/", func(t *testing.T) {
		dir := t.TempDir()
		entities, entries := exportFixture()

		err := ManagedExport(context.Background(), &stubProject{location: dir, graph: resources.NewGraph()},
			&stubManagedProvider{stubImportProvider{importable: importableCollection(), entities: entities, entries: entries}})
		require.NoError(t, err)

		_, err = os.Stat(filepath.Join(dir, ManagedDir, "sources", "my-src.yaml"))
		assert.NoError(t, err)
		_, err = os.Stat(filepath.Join(dir, ImportedDir))
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("refuses a project that has specs", func(t *testing.T) {
		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource("my-src", "source", resources.ResourceData{}, nil))

		err := ManagedExport(context.Background(), &stubProject{location: t.TempDir(), graph: graph},
			&stubManagedProvider{stubImportProvider{importable: importableCollection()}})
		assert.ErrorIs(t, err, ErrProjectNotEmpty)
	})
}
