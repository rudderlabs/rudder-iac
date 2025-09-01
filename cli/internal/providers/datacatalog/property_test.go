package datacatalog_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockPropertyCatalog struct {
	datacatalog.EmptyCatalog
	property *catalog.Property
	err      error
}

func (m *MockPropertyCatalog) CreateProperty(ctx context.Context, propertyCreate catalog.PropertyCreate) (*catalog.Property, error) {
	return m.property, m.err
}

func (m *MockPropertyCatalog) UpdateProperty(ctx context.Context, id string, propertyUpdate *catalog.Property) (*catalog.Property, error) {
	return m.property, m.err
}

func (m *MockPropertyCatalog) DeleteProperty(ctx context.Context, propertyID string) error {
	return m.err
}

func (m *MockPropertyCatalog) SetProperty(property *catalog.Property) {
	m.property = property
}

func (m *MockPropertyCatalog) GetProperty(ctx context.Context, id string) (*catalog.Property, error) {
	return m.property, m.err
}

func (m *MockPropertyCatalog) SetError(err error) {
	m.err = err
}

func TestPropertyProviderOperations(t *testing.T) {

	var (
		ctx              = context.Background()
		mockCatalog      = &MockPropertyCatalog{}
		propertyProvider = datacatalog.NewPropertyProvider(mockCatalog)
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
			ProjectId:   "test-project-id",
			Config:      map[string]interface{}{"key": "value"},
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		})

		toArgs := state.PropertyArgs{
			ProjectId:   "test-project-id",
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
				"projectId":   "test-project-id",
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
				ProjectId:   "test-project-id",
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
			ProjectId:   "test-project-id",
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
			ProjectId:   "test-project-id",
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
				"projectId":   "test-project-id",
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
}
