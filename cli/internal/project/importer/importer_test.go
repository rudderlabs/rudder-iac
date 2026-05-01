package importer

import (
	"context"
	"errors"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
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
