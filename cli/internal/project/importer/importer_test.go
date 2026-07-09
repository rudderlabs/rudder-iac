package importer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/backend"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockProject struct {
	graph    *resources.Graph
	graphErr error
	location string
}

func (m *mockProject) ResourceGraph() (*resources.Graph, error) {
	return m.graph, m.graphErr
}

func (m *mockProject) Location() string {
	return m.location
}

type mockImportProvider struct {
	loadRemoteVal       *resources.RemoteResources
	loadRemoteErr       error
	mapStateVal         *state.State
	mapStateErr         error
	loadImportableVal   *resources.RemoteResources
	loadImportableErr   error
	formatVal           []writer.FormattableEntity
	formatErr           error
}

func (m *mockImportProvider) LoadResourcesFromRemote(context.Context) (*resources.RemoteResources, error) {
	return m.loadRemoteVal, m.loadRemoteErr
}

func (m *mockImportProvider) MapRemoteToState(*resources.RemoteResources) (*state.State, error) {
	return m.mapStateVal, m.mapStateErr
}

func (m *mockImportProvider) LoadImportable(context.Context, namer.Namer) (*resources.RemoteResources, error) {
	return m.loadImportableVal, m.loadImportableErr
}

func (m *mockImportProvider) FormatForExport(*resources.RemoteResources, namer.Namer, resolver.ReferenceResolver) ([]writer.FormattableEntity, error) {
	return m.formatVal, m.formatErr
}

func syncedGraph() *resources.Graph {
	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource("book-a", "books", resources.ResourceData{"title": "A"}, nil))
	return graph
}

func syncedState() *state.State {
	st := state.EmptyState()
	st.AddResource(&state.ResourceState{
		ID:    "book-a",
		Type:  "books",
		Input: map[string]any{"title": "A"},
	})
	return st
}

func TestWorkspaceImport(t *testing.T) {
	t.Parallel()

	remoteErr := errors.New("remote unavailable")
	mapStateErr := errors.New("map state failed")
	graphErr := errors.New("graph unavailable")
	importableErr := errors.New("importable failed")
	formatErr := errors.New("format failed")

	tests := []struct {
		name  string
		setup func(t *testing.T) (*mockProject, *mockImportProvider, func(t *testing.T, location string))
		wantErr     string
		wantErrIs   error
	}{
		{
			name: "load remote resources error",
			setup: func(t *testing.T) (*mockProject, *mockImportProvider, func(t *testing.T, location string)) {
				return &mockProject{location: t.TempDir()}, &mockImportProvider{loadRemoteErr: remoteErr}, nil
			},
			wantErr: "loading remote resources: remote unavailable",
		},
		{
			name: "map remote to state error",
			setup: func(t *testing.T) (*mockProject, *mockImportProvider, func(t *testing.T, location string)) {
				return &mockProject{
						location: t.TempDir(),
						graph:    syncedGraph(),
					}, &mockImportProvider{
						loadRemoteVal: resources.NewRemoteResources(),
						mapStateErr:   mapStateErr,
					}, nil
			},
			wantErr: "loading state from resources: map state failed",
		},
		{
			name: "resource graph error",
			setup: func(t *testing.T) (*mockProject, *mockImportProvider, func(t *testing.T, location string)) {
				return &mockProject{
						location: t.TempDir(),
						graphErr: graphErr,
					}, &mockImportProvider{
						loadRemoteVal: resources.NewRemoteResources(),
						mapStateVal:   syncedState(),
					}, nil
			},
			wantErr: "getting resource graph: graph unavailable",
		},
		{
			name: "project not synced",
			setup: func(t *testing.T) (*mockProject, *mockImportProvider, func(t *testing.T, location string)) {
				return &mockProject{
						location: t.TempDir(),
						graph:    resources.NewGraph(),
					}, &mockImportProvider{
						loadRemoteVal: resources.NewRemoteResources(),
						mapStateVal:   syncedState(),
					}, nil
			},
			wantErrIs: ErrProjectNotSynced,
		},
		{
			name: "no resources to import",
			setup: func(t *testing.T) (*mockProject, *mockImportProvider, func(t *testing.T, location string)) {
				return &mockProject{
						location: t.TempDir(),
						graph:    syncedGraph(),
					}, &mockImportProvider{
						loadRemoteVal:     resources.NewRemoteResources(),
						mapStateVal:       syncedState(),
						loadImportableVal: resources.NewRemoteResources(),
					}, nil
			},
		},
		{
			name: "load importable error",
			setup: func(t *testing.T) (*mockProject, *mockImportProvider, func(t *testing.T, location string)) {
				return &mockProject{
						location: t.TempDir(),
						graph:    syncedGraph(),
					}, &mockImportProvider{
						loadRemoteVal:     resources.NewRemoteResources(),
						mapStateVal:       syncedState(),
						loadImportableErr: importableErr,
					}, nil
			},
			wantErr: "loading importable resources: importable failed",
		},
		{
			name: "format for export error",
			setup: func(t *testing.T) (*mockProject, *mockImportProvider, func(t *testing.T, location string)) {
				return &mockProject{
						location: t.TempDir(),
						graph:    syncedGraph(),
					}, &mockImportProvider{
						loadRemoteVal: resources.NewRemoteResources(),
						mapStateVal:   syncedState(),
						loadImportableVal: func() *resources.RemoteResources {
							collection := resources.NewRemoteResources()
							collection.Set("books", map[string]*resources.RemoteResource{
								"book-a": {ID: "book-a", ExternalID: "book-a", Data: map[string]any{"title": "A"}},
							})
							return collection
						}(),
						formatErr: formatErr,
					}, nil
			},
			wantErr: "normalizing for import: format failed",
		},
		{
			name: "writes imported specs",
			setup: func(t *testing.T) (*mockProject, *mockImportProvider, func(t *testing.T, location string)) {
				location := t.TempDir()
				return &mockProject{
						location: location,
						graph:    syncedGraph(),
					}, &mockImportProvider{
						loadRemoteVal: resources.NewRemoteResources(),
						mapStateVal:   syncedState(),
						loadImportableVal: func() *resources.RemoteResources {
							collection := resources.NewRemoteResources()
							collection.Set("books", map[string]*resources.RemoteResource{
								"book-a": {ID: "book-a", ExternalID: "book-a", Data: map[string]any{"title": "A"}},
							})
							return collection
						}(),
						formatVal: []writer.FormattableEntity{
							{Content: "title: A", RelativePath: "books/book-a.yaml"},
						},
					}, func(t *testing.T, loc string) {
						t.Helper()
						imported := filepath.Join(loc, ImportedDir, "books", "book-a.yaml")
						info, err := os.Stat(imported)
						require.NoError(t, err)
						assert.False(t, info.IsDir())
						content, err := os.ReadFile(imported)
						require.NoError(t, err)
						assert.NotEmpty(t, content)
					}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			project, provider, checkImport := tt.setup(t)
			err := WorkspaceImport(context.Background(), project, provider)
			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
				return
			}
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			if checkImport != nil {
				checkImport(t, project.location)
			}
		})
	}
}

func TestWorkspaceImport_WithExampleProvider(t *testing.T) {
	t.Parallel()

	b := backend.NewBackend()
	provider := example.NewProvider(b)
	testDir := t.TempDir()

	proj := project.New(provider)
	require.NoError(t, proj.Load(testDir))

	writer, err := b.CreateWriter("Writer A", "")
	require.NoError(t, err)
	_, err = b.CreateBook("Book A", writer.ID, "", "")
	require.NoError(t, err)

	require.NoError(t, WorkspaceImport(context.Background(), proj, provider))

	importedDir := filepath.Join(testDir, ImportedDir)
	info, err := os.Stat(importedDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	entries, err := os.ReadDir(importedDir)
	require.NoError(t, err)
	assert.NotEmpty(t, entries, fmt.Sprintf("expected imported specs under %s", importedDir))
}
