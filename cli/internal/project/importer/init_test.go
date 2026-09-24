package importer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubInitProvider records the filter it is called with so the test can assert
// init asks for managed resources too.
type stubInitProvider struct {
	importable  *resources.RemoteResources
	entities    []writer.FormattableEntity
	gotFilters  []resources.ImportableFilter
	gotResolver resolver.ReferenceResolver
}

func (p *stubInitProvider) LoadImportable(_ context.Context, _ namer.Namer, filter ...resources.ImportableFilter) (*resources.RemoteResources, error) {
	p.gotFilters = filter
	return p.importable, nil
}

func (p *stubInitProvider) FormatForExport(
	_ *resources.RemoteResources,
	_ namer.Namer,
	r resolver.ReferenceResolver,
) ([]writer.FormattableEntity, []importmanifest.ImportEntry, error) {
	p.gotResolver = r
	return p.entities, nil, nil
}

func TestWorkspaceInit(t *testing.T) {
	dir := t.TempDir()
	entities, _ := exportFixture()
	p := &stubInitProvider{importable: importableCollection(), entities: entities}

	require.NoError(t, WorkspaceInit(context.Background(), dir, p))

	// Managed resources are in scope — that is the whole point of init.
	assert.Equal(t, []resources.ImportableFilter{{IncludeManaged: true}}, p.gotFilters)

	// Specs land at the project root, not under an imported/ wrapper.
	_, err := os.Stat(filepath.Join(dir, "sources", "my-src.yaml"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(dir, ImportedDir))
	assert.True(t, os.IsNotExist(err), "init must not create an imported/ directory")

	// No manifest: nothing is being adopted into an existing project.
	_, err = os.Stat(filepath.Join(dir, importmanifest.FileName))
	assert.True(t, os.IsNotExist(err))
}

func TestWorkspaceInit_ResolvesEverythingThroughImportable(t *testing.T) {
	entities, _ := exportFixture()

	referenced := resources.NewRemoteResources()
	referenced.Set("source", map[string]*resources.RemoteResource{
		"rid-1": {ID: "rid-1", ExternalID: "my-src", Reference: "#source:my-src"},
	})
	p := &stubInitProvider{importable: referenced, entities: entities}

	require.NoError(t, WorkspaceInit(context.Background(), t.TempDir(), p))

	// A clone has no local project, so references must resolve off the
	// importable collection alone — never off the (empty) graph.
	ref, err := p.gotResolver.ResolveToReference("source", "rid-1")
	require.NoError(t, err)
	assert.Equal(t, "#source:my-src", ref)
}

func TestWorkspaceInit_NoResources(t *testing.T) {
	dir := t.TempDir()
	p := &stubInitProvider{importable: resources.NewRemoteResources()}

	require.NoError(t, WorkspaceInit(context.Background(), dir, p))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestWorkspaceInit_RefusesDirectoryWithSpecs(t *testing.T) {
	for _, name := range []string{"existing.yaml", "existing.yml", "EXISTING.YAML"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("version: rudder/v1\n"), 0644))

			err := WorkspaceInit(context.Background(), dir, &stubInitProvider{})
			require.ErrorIs(t, err, ErrProjectNotEmpty)
		})
	}
}

func TestWorkspaceInit_RefusesSpecsInSubdirectories(t *testing.T) {
	// A project keeps its specs in per-provider subdirectories — the layout init
	// itself writes — so a top-level-only check would miss re-running init over
	// its own output.
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "data-catalog"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data-catalog", "events.yaml"),
		[]byte("version: rudder/v1\n"), 0644))

	err := WorkspaceInit(context.Background(), dir, &stubInitProvider{})
	require.ErrorIs(t, err, ErrProjectNotEmpty)
	assert.Contains(t, err.Error(), filepath.Join("data-catalog", "events.yaml"))
}

func TestWorkspaceInit_AllowsDirectoryWithoutSpecs(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.vars.yaml"), 0644))

	entities, _ := exportFixture()
	err := WorkspaceInit(context.Background(), dir, &stubInitProvider{
		importable: importableCollection(),
		entities:   entities,
	})
	require.NoError(t, err)
}

func TestWorkspaceInit_MissingDirectoryIsCreated(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new-project")
	entities, _ := exportFixture()

	err := WorkspaceInit(context.Background(), dir, &stubInitProvider{
		importable: importableCollection(),
		entities:   entities,
	})
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "sources", "my-src.yaml"))
	assert.NoError(t, err)
}
