package importer

import (
	"context"
	"errors"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockImportProvider implements ImportProvider for testing
type mockImportProvider struct {
	managedResources   *resources.RemoteResources
	unmanagedResources *resources.RemoteResources
	loadRemoteErr      error
	loadImportableErr  error
	mapToStateErr      error
	formatExportErr    error
}

func (m *mockImportProvider) LoadResourcesFromRemote(ctx context.Context) (*resources.RemoteResources, error) {
	if m.loadRemoteErr != nil {
		return nil, m.loadRemoteErr
	}
	return m.managedResources, nil
}

func (m *mockImportProvider) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.RemoteResources, error) {
	if m.loadImportableErr != nil {
		return nil, m.loadImportableErr
	}
	return m.unmanagedResources, nil
}

func (m *mockImportProvider) MapRemoteToState(collection *resources.RemoteResources) (*state.State, error) {
	if m.mapToStateErr != nil {
		return nil, m.mapToStateErr
	}
	return state.EmptyState(), nil
}

func (m *mockImportProvider) FormatForExport(
	collection *resources.RemoteResources,
	idNamer namer.Namer,
	resolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, error) {
	if m.formatExportErr != nil {
		return nil, m.formatExportErr
	}
	return []writer.FormattableEntity{}, nil
}

// mockProject implements Project for testing
type mockProject struct {
	graph    *resources.Graph
	location string
	graphErr error
}

func (m *mockProject) ResourceGraph() (*resources.Graph, error) {
	if m.graphErr != nil {
		return nil, m.graphErr
	}
	return m.graph, nil
}

func (m *mockProject) Location() string {
	return m.location
}

func TestFilterOption_Values(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filter   FilterOption
		expected string
	}{
		{
			name:     "unmanaged filter value",
			filter:   FilterUnmanaged,
			expected: "unmanaged",
		},
		{
			name:     "managed filter value",
			filter:   FilterManaged,
			expected: "managed",
		},
		{
			name:     "all filter value",
			filter:   FilterAll,
			expected: "all",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.filter))
		})
	}
}

func TestLoadResourcesForImport(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	idNamer := namer.NewExternalIdNamer(namer.NewKebabCase())

	// Create test resources
	managedResource := &resources.RemoteResource{
		ID:         "remote-1",
		ExternalID: "managed-resource-1",
		Data:       map[string]interface{}{"name": "Managed Resource"},
	}

	unmanagedResource := &resources.RemoteResource{
		ID:         "remote-2",
		ExternalID: "unmanaged-resource-1",
		Data:       map[string]interface{}{"name": "Unmanaged Resource"},
	}

	managedCollection := resources.NewRemoteResources()
	managedCollection.Set("test-type", map[string]*resources.RemoteResource{
		"remote-1": managedResource,
	})

	unmanagedCollection := resources.NewRemoteResources()
	unmanagedCollection.Set("test-type-unmanaged", map[string]*resources.RemoteResource{
		"remote-2": unmanagedResource,
	})

	tests := []struct {
		name          string
		filter        FilterOption
		provider      *mockImportProvider
		expectedLen   int
		expectedError string
	}{
		{
			name:   "FilterUnmanaged returns unmanaged resources",
			filter: FilterUnmanaged,
			provider: &mockImportProvider{
				managedResources:   managedCollection,
				unmanagedResources: unmanagedCollection,
			},
			expectedLen: 1,
		},
		{
			name:   "FilterManaged returns managed resources",
			filter: FilterManaged,
			provider: &mockImportProvider{
				managedResources:   managedCollection,
				unmanagedResources: unmanagedCollection,
			},
			expectedLen: 1,
		},
		{
			name:   "FilterAll returns all resources",
			filter: FilterAll,
			provider: &mockImportProvider{
				managedResources:   managedCollection,
				unmanagedResources: unmanagedCollection,
			},
			expectedLen: 2,
		},
		{
			name:   "FilterUnmanaged with error",
			filter: FilterUnmanaged,
			provider: &mockImportProvider{
				managedResources:   managedCollection,
				loadImportableErr:  errors.New("failed to load"),
			},
			expectedError: "failed to load",
		},
		{
			name:   "FilterAll with unmanaged error",
			filter: FilterAll,
			provider: &mockImportProvider{
				managedResources:   managedCollection,
				loadImportableErr:  errors.New("failed to load unmanaged"),
			},
			expectedError: "loading unmanaged resources",
		},
		{
			name:          "Unknown filter returns error",
			filter:        FilterOption("invalid"),
			provider:      &mockImportProvider{},
			expectedError: "unknown filter option",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := loadResourcesForImport(ctx, tt.provider, tt.filter, idNamer, tt.provider.managedResources)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedLen, result.Len())
		})
	}
}

func TestLoadManagedForExport(t *testing.T) {
	t.Parallel()

	managedResource := &resources.RemoteResource{
		ID:         "remote-1",
		ExternalID: "managed-resource-1",
		Data:       map[string]interface{}{"name": "Test Resource"},
		Reference:  "#test:managed-resource-1",
	}

	collection := resources.NewRemoteResources()
	collection.Set("test-type", map[string]*resources.RemoteResource{
		"remote-1": managedResource,
	})

	result, err := loadManagedForExport(collection)
	require.NoError(t, err)

	// The function should return the same collection
	assert.Equal(t, collection, result)
	assert.Equal(t, 1, result.Len())

	// Verify the resource is accessible
	resource, found := result.GetByID("test-type", "remote-1")
	require.True(t, found)
	assert.Equal(t, "managed-resource-1", resource.ExternalID)
}

func TestImportOptions_DefaultFilter(t *testing.T) {
	t.Parallel()

	opts := ImportOptions{}
	assert.Equal(t, FilterOption(""), opts.Filter)

	opts = ImportOptions{Filter: FilterManaged}
	assert.Equal(t, FilterManaged, opts.Filter)
}

func TestConflictResolution_Values(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		resolution ConflictResolution
		expected   string
	}{
		{
			name:       "keep-local conflict resolution value",
			resolution: ConflictKeepLocal,
			expected:   "keep-local",
		},
		{
			name:       "accept-incoming conflict resolution value",
			resolution: ConflictAcceptIncoming,
			expected:   "accept-incoming",
		},
		{
			name:       "keep-both conflict resolution value",
			resolution: ConflictKeepBoth,
			expected:   "keep-both",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.resolution))
		})
	}
}

func TestImportOptions_DefaultOnConflict(t *testing.T) {
	t.Parallel()

	opts := ImportOptions{}
	assert.Equal(t, ConflictResolution(""), opts.OnConflict)

	opts = ImportOptions{OnConflict: ConflictAcceptIncoming}
	assert.Equal(t, ConflictAcceptIncoming, opts.OnConflict)
}

// createTestSpec creates a test specs.Spec with the given URNs in import metadata
func createTestSpec(urns []string) *specs.Spec {
	importResources := make([]specs.ImportIds, 0, len(urns))
	for _, urn := range urns {
		importResources = append(importResources, specs.ImportIds{
			URN:      urn,
			RemoteID: "remote-" + urn,
		})
	}

	metadata := specs.Metadata{
		Name: "test-spec",
		Import: &specs.WorkspacesImportMetadata{
			Workspaces: []specs.WorkspaceImportMetadata{
				{
					WorkspaceID: "workspace-1",
					Resources:   importResources,
				},
			},
		},
	}

	metadataMap, _ := metadata.ToMap()

	return &specs.Spec{
		Version:  specs.SpecVersionV1,
		Kind:     "test-kind",
		Metadata: metadataMap,
		Spec:     map[string]any{"name": "test"},
	}
}

func TestExtractURNsFromEntity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		entity       writer.FormattableEntity
		expectedURNs []string
	}{
		{
			name: "extracts URNs from spec pointer",
			entity: writer.FormattableEntity{
				Content:      createTestSpec([]string{"source:my-source", "destination:my-dest"}),
				RelativePath: "test.yaml",
			},
			expectedURNs: []string{"source:my-source", "destination:my-dest"},
		},
		{
			name: "extracts URNs from spec value",
			entity: writer.FormattableEntity{
				Content:      *createTestSpec([]string{"source:single-source"}),
				RelativePath: "test.yaml",
			},
			expectedURNs: []string{"source:single-source"},
		},
		{
			name: "returns empty for non-spec content",
			entity: writer.FormattableEntity{
				Content:      map[string]any{"key": "value"},
				RelativePath: "test.yaml",
			},
			expectedURNs: nil,
		},
		{
			name: "returns empty for spec without import metadata",
			entity: writer.FormattableEntity{
				Content: &specs.Spec{
					Version:  specs.SpecVersionV1,
					Kind:     "test-kind",
					Metadata: map[string]any{"name": "test"},
					Spec:     map[string]any{"name": "test"},
				},
				RelativePath: "test.yaml",
			},
			expectedURNs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urns := extractURNsFromEntity(tt.entity)
			assert.Equal(t, tt.expectedURNs, urns)
		})
	}
}

func TestEntityHasConflict(t *testing.T) {
	t.Parallel()

	// Create a graph with existing resources
	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource("existing-source", "source", resources.ResourceData{"name": "Existing"}, nil))
	graph.AddResource(resources.NewResource("existing-dest", "destination", resources.ResourceData{"name": "Existing Dest"}, nil))

	tests := []struct {
		name        string
		entity      writer.FormattableEntity
		hasConflict bool
	}{
		{
			name: "detects conflict when resource exists",
			entity: writer.FormattableEntity{
				Content:      createTestSpec([]string{"source:existing-source"}),
				RelativePath: "sources/existing-source.yaml",
			},
			hasConflict: true,
		},
		{
			name: "detects conflict when any resource in entity exists",
			entity: writer.FormattableEntity{
				Content:      createTestSpec([]string{"source:new-source", "destination:existing-dest"}),
				RelativePath: "mixed.yaml",
			},
			hasConflict: true,
		},
		{
			name: "no conflict when resource does not exist",
			entity: writer.FormattableEntity{
				Content:      createTestSpec([]string{"source:new-source"}),
				RelativePath: "sources/new-source.yaml",
			},
			hasConflict: false,
		},
		{
			name: "no conflict for non-spec content",
			entity: writer.FormattableEntity{
				Content:      map[string]any{"key": "value"},
				RelativePath: "other.yaml",
			},
			hasConflict: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := entityHasConflict(tt.entity, graph)
			assert.Equal(t, tt.hasConflict, result)
		})
	}
}

func TestGenerateConflictPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		inputPath    string
		expectedPath string
	}{
		{
			name:         "adds suffix before yaml extension",
			inputPath:    "sources/my-source.yaml",
			expectedPath: "sources/my-source-imported.yaml",
		},
		{
			name:         "handles nested paths",
			inputPath:    "deep/nested/path/resource.yaml",
			expectedPath: "deep/nested/path/resource-imported.yaml",
		},
		{
			name:         "handles different extensions",
			inputPath:    "config/settings.txt",
			expectedPath: "config/settings-imported.txt",
		},
		{
			name:         "handles file without extension",
			inputPath:    "config/noextension",
			expectedPath: "config/noextension-imported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateConflictPath(tt.inputPath)
			assert.Equal(t, tt.expectedPath, result)
		})
	}
}

func TestApplyConflictResolution(t *testing.T) {
	t.Parallel()

	// Create a graph with existing resources
	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource("existing-source", "source", resources.ResourceData{"name": "Existing"}, nil))

	// Create test entities - one with conflict, one without
	conflictingEntity := writer.FormattableEntity{
		Content:      createTestSpec([]string{"source:existing-source"}),
		RelativePath: "sources/existing-source.yaml",
	}
	newEntity := writer.FormattableEntity{
		Content:      createTestSpec([]string{"source:new-source"}),
		RelativePath: "sources/new-source.yaml",
	}

	tests := []struct {
		name           string
		entities       []writer.FormattableEntity
		onConflict     ConflictResolution
		expectedCount  int
		expectedPaths  []string
	}{
		{
			name:           "keep-local skips conflicting entities",
			entities:       []writer.FormattableEntity{conflictingEntity, newEntity},
			onConflict:     ConflictKeepLocal,
			expectedCount:  1,
			expectedPaths:  []string{"sources/new-source.yaml"},
		},
		{
			name:           "accept-incoming keeps all entities",
			entities:       []writer.FormattableEntity{conflictingEntity, newEntity},
			onConflict:     ConflictAcceptIncoming,
			expectedCount:  2,
			expectedPaths:  []string{"sources/existing-source.yaml", "sources/new-source.yaml"},
		},
		{
			name:           "keep-both modifies path for conflicting entities",
			entities:       []writer.FormattableEntity{conflictingEntity, newEntity},
			onConflict:     ConflictKeepBoth,
			expectedCount:  2,
			expectedPaths:  []string{"sources/existing-source-imported.yaml", "sources/new-source.yaml"},
		},
		{
			name:           "empty entities returns empty",
			entities:       []writer.FormattableEntity{},
			onConflict:     ConflictKeepLocal,
			expectedCount:  0,
			expectedPaths:  nil,
		},
		{
			name:           "unknown conflict resolution defaults to keep-local behavior",
			entities:       []writer.FormattableEntity{conflictingEntity, newEntity},
			onConflict:     ConflictResolution("unknown"),
			expectedCount:  1,
			expectedPaths:  []string{"sources/new-source.yaml"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyConflictResolution(tt.entities, graph, tt.onConflict)
			assert.Equal(t, tt.expectedCount, len(result))

			if tt.expectedPaths != nil {
				actualPaths := make([]string, len(result))
				for i, e := range result {
					actualPaths[i] = e.RelativePath
				}
				assert.ElementsMatch(t, tt.expectedPaths, actualPaths)
			}
		})
	}
}

func TestApplyConflictResolution_OnlyAffectsManagedFilter(t *testing.T) {
	t.Parallel()

	// This test verifies that conflict resolution only applies when filter is managed or all.
	// The logic is in WorkspaceImport, but we test the applyConflictResolution function directly
	// to ensure it works correctly when called.

	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource("existing", "source", resources.ResourceData{}, nil))

	entity := writer.FormattableEntity{
		Content:      createTestSpec([]string{"source:existing"}),
		RelativePath: "sources/existing.yaml",
	}

	// With keep-local, conflicting entity should be skipped
	result := applyConflictResolution([]writer.FormattableEntity{entity}, graph, ConflictKeepLocal)
	assert.Equal(t, 0, len(result))

	// With accept-incoming, entity should be kept
	result = applyConflictResolution([]writer.FormattableEntity{entity}, graph, ConflictAcceptIncoming)
	assert.Equal(t, 1, len(result))

	// With keep-both, path should be modified
	result = applyConflictResolution([]writer.FormattableEntity{entity}, graph, ConflictKeepBoth)
	require.Equal(t, 1, len(result))
	assert.Equal(t, "sources/existing-imported.yaml", result[0].RelativePath)
}
