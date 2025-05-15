package provider_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/pkg/provider"
	"github.com/rudderlabs/rudder-iac/cli/pkg/provider/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ catalog.DataCatalog = &MockCustomTypeCatalog{}

type MockCustomTypeCatalog struct {
	EmptyCatalog
	customType *catalog.CustomType
	err        error
}

func (m *MockCustomTypeCatalog) CreateCustomType(ctx context.Context, customTypeCreate catalog.CustomTypeCreate) (*catalog.CustomType, error) {
	return m.customType, m.err
}

func (m *MockCustomTypeCatalog) UpdateCustomType(ctx context.Context, id string, customTypeUpdate *catalog.CustomType) (*catalog.CustomType, error) {
	return m.customType, m.err
}

func (m *MockCustomTypeCatalog) DeleteCustomType(ctx context.Context, customTypeID string) error {
	return m.err
}

func (m *MockCustomTypeCatalog) SetCustomType(customType *catalog.CustomType) {
	m.customType = customType
}

func (m *MockCustomTypeCatalog) SetError(err error) {
	m.err = err
}

func TestCustomTypeProviderOperations(t *testing.T) {
	var (
		ctx                = context.Background()
		mockCatalog        = &MockCustomTypeCatalog{}
		customTypeProvider = provider.NewCustomTypeProvider(mockCatalog)
		createdAt, _       = time.Parse(time.RFC3339, "2021-09-01T00:00:00Z")
		updatedAt, _       = time.Parse(time.RFC3339, "2021-09-02T00:00:00Z")
	)

	t.Run("Create", func(t *testing.T) {
		mockCatalog.SetCustomType(&catalog.CustomType{
			ID:              "custom-type-id",
			Name:            "test-custom-type",
			Description:     "Test Custom Type",
			Type:            "object",
			WorkspaceId:     "workspace-id",
			Config:          map[string]interface{}{"key": "value"},
			Version:         1,
			ItemDefinitions: []string{"def1", "def2"},
			Rules:           map[string]interface{}{"rule1": "value1"},
			Properties: []catalog.CustomTypeProperty{
				{ID: "prop1", Required: true},
				{ID: "prop2", Required: false},
			},
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})

		customTypeArgs := state.CustomTypeArgs{
			Name:        "test-custom-type",
			LocalID:     "local-custom-type-id",
			Description: "Test Custom Type",
			Type:        "object",
			Config:      map[string]interface{}{"key": "value"},
			Properties: []*state.CustomTypePropertyArgs{
				{RefID: "ref1", PropertyID: "prop1", Required: true},
				{RefID: "ref2", PropertyID: "prop2", Required: false},
			},
		}

		resourceData, err := customTypeProvider.Create(ctx, "custom-type-id", customTypeArgs.ToResourceData())
		require.Nil(t, err)

		assert.Equal(t, resources.ResourceData{
			"id": "custom-type-id",
			"name": "test-custom-type",
			"description": "Test Custom Type",
			"type": "object",
			"config": map[string]interface{}{"key": "value"},
			"workspaceId": "workspace-id",
			"createdAt": "2021-09-01 00:00:00 +0000 UTC",
			"updatedAt": "2021-09-02 00:00:00 +0000 UTC",
			"properties": []map[string]interface{}{
				{
					"localId": "local-custom-type-id",
					"propertyId": "prop1",
					"required": true,
				},
				{
					"localId": "local-custom-type-id",
					"propertyId": "prop2",
					"required": false,
				},
			},
		}, resourceData)
	})

	t.Run("Update", func(t *testing.T) {
		// Previous state
		prevState := state.CustomTypeState{
			CustomTypeArgs: state.CustomTypeArgs{
				Name:        "test-custom-type",
				LocalID:     "local-custom-type-id",
				Description: "Test Custom Type",
				Type:        "object",
				Config:      map[string]interface{}{"key": "value"},
				Properties: []*state.CustomTypePropertyArgs{
					{RefID: "ref1", PropertyID: "prop1", Required: true},
					{RefID: "ref2", PropertyID: "prop2", Required: false},
				},
			},
			ID:              "custom-type-id",
			Name:            "test-custom-type",
			Description:     "Test Custom Type",
			Type:            "object",
			Config:          map[string]interface{}{"key": "value"},
			Version:         1,
			ItemDefinitions: []string{"def1", "def2"},
			Rules:           map[string]interface{}{"rule1": "value1"},
			WorkspaceID:     "workspace-id",
			CreatedAt:       createdAt.String(),
			UpdatedAt:       updatedAt.String(),
			Properties: []*state.CustomTypePropertyState{
				{LocalID: "local-prop1", PropertyID: "prop1", Required: true},
				{LocalID: "local-prop2", PropertyID: "prop2", Required: false},
			},
		}

		// New args to update
		updateArgs := state.CustomTypeArgs{
			Name:        "updated-custom-type",
			LocalID:     "local-custom-type-id",
			Description: "Updated Custom Type",
			Type:        "object",
			Config:      map[string]interface{}{"key": "updated-value"},
			Properties: []*state.CustomTypePropertyArgs{
				{RefID: "ref1", PropertyID: "prop1", Required: true},
				{RefID: "ref2", PropertyID: "prop2", Required: true}, // Changed required
				{RefID: "ref3", PropertyID: "prop3", Required: false}, // Added new
			},
		}

		// Set up mock response for update
		mockCatalog.SetCustomType(&catalog.CustomType{
			ID:              "custom-type-id",
			Name:            "updated-custom-type", 
			Description:     "Updated Custom Type",
			Type:            "object",
			WorkspaceId:     "workspace-id",
			Config:          map[string]interface{}{"key": "updated-value"},
			Version:         2, // Bumped version
			ItemDefinitions: []string{"def1", "def2"},
			Rules:           map[string]interface{}{"rule1": "value1"},
			Properties: []catalog.CustomTypeProperty{
				{ID: "prop1", Required: true},
				{ID: "prop2", Required: true}, 
				{ID: "prop3", Required: false},
			},
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})

		// Marshal <=> Unmarshal cycle to simulate the persistent state
		olds := prevState.ToResourceData()
		byt, err := json.Marshal(olds)
		require.Nil(t, err)

		var oldData resources.ResourceData
		err = json.Unmarshal(byt, &oldData)
		require.Nil(t, err)

		updatedResource, err := customTypeProvider.Update(ctx, "custom-type-id", updateArgs.ToResourceData(), oldData)
		require.Nil(t, err)

		// Validate the updated resource
		assert.Equal(t, "custom-type-id", (*updatedResource)["id"])
		assert.Equal(t, "updated-custom-type", (*updatedResource)["name"])
		assert.Equal(t, "Updated Custom Type", (*updatedResource)["description"])
		assert.Equal(t, map[string]interface{}{"key": "updated-value"}, (*updatedResource)["config"])
		assert.Equal(t, 2, (*updatedResource)["version"])
		
		// Check properties
		properties := (*updatedResource)["properties"].([]map[string]interface{})
		require.Len(t, properties, 3)
	})

	t.Run("Delete", func(t *testing.T) {
		prevState := state.CustomTypeState{
			ID: "custom-type-id",
		}
		mockCatalog.SetError(nil)

		err := customTypeProvider.Delete(ctx, "custom-type-id", prevState.ToResourceData())
		require.Nil(t, err)
	})
} 