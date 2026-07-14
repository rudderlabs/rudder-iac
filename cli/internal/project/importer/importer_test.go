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
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	})
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
	})
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, ImportedDir, importmanifest.FileName))
	assert.NoError(t, err, "import-manifest.yaml must be written when importMerge is on")
}
