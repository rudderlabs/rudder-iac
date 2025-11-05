package datacatalog_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Extend MockPropertyCatalog for Import testing
type MockPropertyCatalog struct {
	datacatalog.EmptyCatalog
	property            *catalog.Property
	err                 error
	updateCalled        bool // Spy to track if UpdateProperty was called
	setExternalIdCalled bool // Spy to track if SetPropertyExternalId was called
}

func (m *MockPropertyCatalog) CreateProperty(ctx context.Context, propertyCreate catalog.PropertyCreate) (*catalog.Property, error) {
	return m.property, m.err
}

func (m *MockPropertyCatalog) UpdateProperty(ctx context.Context, id string, propertyUpdate *catalog.PropertyUpdate) (*catalog.Property, error) {
	m.updateCalled = true
	m.property.Name = propertyUpdate.Name
	m.property.Description = propertyUpdate.Description
	m.property.Type = propertyUpdate.Type
	m.property.Config = propertyUpdate.Config
	return m.property, m.err
}

func (m *MockPropertyCatalog) DeleteProperty(ctx context.Context, propertyID string) error {
	return m.err
}

func (m *MockPropertyCatalog) GetProperty(ctx context.Context, id string) (*catalog.Property, error) {
	return m.property, m.err
}

func (m *MockPropertyCatalog) SetPropertyExternalId(ctx context.Context, propertyID, externalID string) error {
	m.setExternalIdCalled = true
	return m.err
}

func (m *MockPropertyCatalog) SetProperty(property *catalog.Property) {
	m.property = property
}

func (m *MockPropertyCatalog) SetError(err error) {
	m.err = err
}

func (m *MockPropertyCatalog) ResetSpies() {
	m.updateCalled = false
	m.setExternalIdCalled = false
}

func TestPropertyProviderOperations(t *testing.T) {

	var (
		ctx              = context.Background()
		mockCatalog      = &MockPropertyCatalog{}
		propertyProvider = datacatalog.NewPropertyProvider(mockCatalog, "data-catalog")
		createdAt, _     = time.Parse(time.RFC3339, "2021-09-01T00:00:00Z")
		updatedAt, _     = time.Parse(time.RFC3339, "2021-09-02T00:00:00Z")
	)

	t.Run("Create", func(t *testing.T) {
		mockCatalog.SetProperty(&catalog.Property{
			ID:          "upstream-catalog-id",
			Name:        "property",
			Description: "property description",
			Type:        "property type",
			WorkspaceId: "workspace-id",
			ExternalId:  "test-project-id",
			Config:      map[string]interface{}{"key": "value"},
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		})

		toArgs := state.PropertyArgs{
			Name:        "property",
			Description: "property description",
			Type:        "property type",
			Config:      map[string]interface{}{"key": "value"},
		}

		resourceData, err := propertyProvider.Create(ctx, "property-id", toArgs.ToResourceData())
		require.Nil(t, err)
		assert.Equal(t, resources.ResourceData{
			"id":          "upstream-catalog-id",
			"name":        "property",
			"description": "property description",
			"type":        "property type",
			"config":      map[string]interface{}{"key": "value"},
			"workspaceId": "workspace-id",
			"createdAt":   "2021-09-01 00:00:00 +0000 UTC",
			"updatedAt":   "2021-09-02 00:00:00 +0000 UTC",
			"propertyArgs": map[string]interface{}{
				"name":        "property",
				"description": "property description",
				"type":        "property type",
				"config":      map[string]interface{}{"key": "value"},
			},
		}, *resourceData)
	})

	t.Run("Update", func(t *testing.T) {
		prevState := state.PropertyState{
			PropertyArgs: state.PropertyArgs{
				Name:        "property",
				Description: "property description",
				Type:        "property type",
				Config:      map[string]interface{}{"key": "value"},
			},
			ID:          "upstream-catalog-id",
			Name:        "property",
			Description: "property description",
			Type:        "property type",
			WorkspaceID: "workspace-id",
			CreatedAt:   "2021-09-01 00:00:00 +0000 UTC",
			UpdatedAt:   "2021-09-02 00:00:00 +0000 UTC",
			Config:      map[string]interface{}{"key": "value"},
		}

		toArgs := state.PropertyArgs{
			Name:        "property",
			Description: "property new description",
			Type:        "property type",
			Config:      map[string]interface{}{"key": "value", "key2": "value2"},
		}

		mockCatalog.SetProperty(&catalog.Property{
			ID:          "upstream-catalog-id",
			Name:        "property",
			Description: "property new description",
			Type:        "property type",
			WorkspaceId: "workspace-id",
			Config:      map[string]interface{}{"key": "value", "key2": "value2"},
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		})

		// Marshal <=> Unmarshal cycle to simulate the persistent state
		olds := prevState.ToResourceData()
		byt, err := json.Marshal(olds)
		require.Nil(t, err)

		err = json.Unmarshal(byt, &prevState)
		require.Nil(t, err)

		updatedResource, err := propertyProvider.Update(ctx, "property-id", toArgs.ToResourceData(), olds)
		require.Nil(t, err)
		assert.Equal(t, resources.ResourceData{
			"id":          "upstream-catalog-id",
			"name":        "property",
			"description": "property new description",
			"type":        "property type",
			"config":      map[string]interface{}{"key": "value", "key2": "value2"},
			"workspaceId": "workspace-id",
			"createdAt":   "2021-09-01 00:00:00 +0000 UTC",
			"updatedAt":   "2021-09-02 00:00:00 +0000 UTC",
			"propertyArgs": map[string]interface{}{
				"name":        "property",
				"description": "property new description",
				"type":        "property type",
				"config":      map[string]interface{}{"key": "value", "key2": "value2"},
			},
		}, *updatedResource)

	})

	t.Run("Delete", func(t *testing.T) {
		prevState := state.PropertyState{
			ID: "upstream-catalog-id",
		}
		mockCatalog.SetError(nil)

		err := propertyProvider.Delete(ctx, "property-id", prevState.ToResourceData())
		require.Nil(t, err)
	})

	t.Run("Import", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name           string
			localArgs      state.PropertyArgs
			remoteProperty *catalog.Property
			mockErr        error
			expectErr      bool
			expectUpdate   bool
			expectSetExtId bool
			expectResource *resources.ResourceData
		}{
			{
				name: "successful import no differences",
				localArgs: state.PropertyArgs{
					Name:        "test-prop",
					Description: "desc",
					Type:        "string",
					Config:      map[string]interface{}{"key": "val"},
				},
				remoteProperty: &catalog.Property{
					ID:          "remote-id",
					Name:        "test-prop",
					Description: "desc",
					Type:        "string",
					Config:      map[string]interface{}{"key": "val"},
					WorkspaceId: "ws-id",
					CreatedAt:   createdAt,
					UpdatedAt:   updatedAt,
				},
				expectErr:      false,
				expectUpdate:   false,
				expectSetExtId: true,
				expectResource: &resources.ResourceData{
					"id":          "remote-id",
					"name":        "test-prop",
					"description": "desc",
					"type":        "string",
					"config":      map[string]interface{}{"key": "val"},
					"workspaceId": "ws-id",
					"createdAt":   createdAt.String(),
					"updatedAt":   updatedAt.String(),
					"propertyArgs": map[string]interface{}{
						"name":        "test-prop",
						"description": "desc",
						"type":        "string",
						"config":      map[string]interface{}{"key": "val"},
					},
				},
			},
			{
				name: "successful import with differences",
				localArgs: state.PropertyArgs{
					Name:        "test-prop",
					Description: "new desc",
					Type:        "string",
					Config:      map[string]interface{}{"key": "new val"},
				},
				remoteProperty: &catalog.Property{
					ID:          "remote-id",
					Name:        "test-prop",
					Description: "old desc",
					Type:        "string",
					Config:      map[string]interface{}{"key": "old val"},
					WorkspaceId: "ws-id",
					CreatedAt:   createdAt,
					UpdatedAt:   updatedAt,
				},
				expectErr:      false,
				expectUpdate:   true,
				expectSetExtId: true,
				expectResource: &resources.ResourceData{
					"id":          "remote-id",
					"name":        "test-prop",
					"description": "new desc",
					"type":        "string",
					"config":      map[string]interface{}{"key": "new val"},
					"workspaceId": "ws-id",
					"createdAt":   createdAt.String(),
					"updatedAt":   updatedAt.String(),
					"propertyArgs": map[string]interface{}{
						"name":        "test-prop",
						"description": "new desc",
						"type":        "string",
						"config":      map[string]interface{}{"key": "new val"},
					},
				},
			},
			{
				name:           "error on get property",
				localArgs:      state.PropertyArgs{Name: "test-prop"},
				remoteProperty: nil,
				mockErr:        fmt.Errorf("error getting property"),
				expectErr:      true,
			},
			{
				name: "error on update property",
				localArgs: state.PropertyArgs{
					Name:        "test-prop",
					Description: "new desc",
				},
				remoteProperty: &catalog.Property{
					ID:          "remote-id",
					Name:        "test-prop",
					Description: "old desc",
				},
				mockErr:      fmt.Errorf("error updating property"),
				expectErr:    true,
				expectUpdate: true, // But will fail
			},
			{
				name: "error on set external ID",
				localArgs: state.PropertyArgs{
					Name: "test-prop",
				},
				remoteProperty: &catalog.Property{
					ID:   "remote-id",
					Name: "test-prop",
				},
				mockErr:        fmt.Errorf("error setting external ID"),
				expectErr:      true,
				expectSetExtId: true, // But will fail
			},
		}

		for _, tt := range tests {
			tt := tt // Capture range variable
			t.Run(tt.name, func(t *testing.T) {

				mockCatalog.ResetSpies()
				mockCatalog.SetProperty(tt.remoteProperty)
				mockCatalog.SetError(tt.mockErr)

				res, err := propertyProvider.Import(ctx, "local-id", tt.localArgs.ToResourceData(), "remote-id")

				if tt.expectErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tt.expectResource, res)
				assert.Equal(t, tt.expectUpdate, mockCatalog.updateCalled)
				assert.Equal(t, tt.expectSetExtId, mockCatalog.setExternalIdCalled)
			})
		}
	})
}
