package state_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/pkg/provider/state"
	"github.com/stretchr/testify/assert"
)

func TestCustomTypeArgs_ResourceData(t *testing.T) {
	args := state.CustomTypeArgs{
		Name:        "custom-type-name",
		LocalID:     "custom-type-local-id",
		Description: "custom-type-description",
		Type:        "custom-type-type",
		Config: map[string]interface{}{
			"enum": []string{"value1", "value2"},
		},
		Properties: []*state.CustomTypePropertyArgs{
			{
				Name:        "property-name",
				LocalID:     "property-local-id",
				Description: "property-description",
				Type:        "property-type",
				Config:      map[string]interface{}{},
				Required:    true,
			},
		},
	}

	t.Run("to resource data", func(t *testing.T) {
		t.Parallel()

		resourceData := args.ToResourceData()
		assert.Equal(t, resources.ResourceData{
			"name":        "custom-type-name",
			"localId":     "custom-type-local-id",
			"description": "custom-type-description",
			"type":        "custom-type-type",
			"config":      map[string]interface{}{"enum": []string{"value1", "value2"}},
			"properties": []map[string]interface{}{
				{
					"name":        "property-name",
					"localId":     "property-local-id",
					"description": "property-description",
					"type":        "property-type",
					"config":      map[string]interface{}{},
					"required":    true,
				},
			},
		}, resourceData)
	})

	t.Run("from resource data", func(t *testing.T) {
		t.Parallel()

		loopback := state.CustomTypeArgs{}
		loopback.FromResourceData(args.ToResourceData())
		assert.Equal(t, args, loopback)
	})

	t.Run("from resource data with empty values", func(t *testing.T) {
		t.Parallel()
		resourceData := resources.ResourceData{
			"name":        "custom-type-name",
			"localId":     "",
			"description": "",
			"type":        "",
			"config":      map[string]interface{}{},
			"properties":  []map[string]interface{}{},
		}

		args := state.CustomTypeArgs{}
		args.FromResourceData(resourceData)
		assert.Equal(t, state.CustomTypeArgs{
			Name:        "custom-type-name",
			LocalID:     "",
			Description: "",
			Type:        "",
			Config:      map[string]interface{}{},
			Properties:  []*state.CustomTypePropertyArgs{},
		}, args)
	})
}

func TestCustomTypeState_ResourceData(t *testing.T) {
	customTypeState := state.CustomTypeState{
		ID:              "upstream-custom-type-id",
		Name:            "custom-type-name",
		Description:     "custom-type-description",
		Type:            "custom-type-type",
		Version:         1,
		DataType:        "object",
		Rules:           map[string]interface{}{},
		ItemDefinitions: []string{"def1"},
		WorkspaceID:     "workspace-id",
		Config: map[string]interface{}{
			"itemTypes": []string{"def1"},
		},
		CreatedAt: "2021-09-01T00:00:00Z",
		UpdatedAt: "2021-09-01T00:00:00Z",
		Properties: []*state.CustomTypePropertyState{
			{
				ID:         "property-id",
				LocalID:    "property-local-id",
				PropertyID: "upstream-property-id",
			},
		},
		CustomTypeArgs: state.CustomTypeArgs{
			Name:        "custom-type-name",
			LocalID:     "custom-type-local-id",
			Description: "custom-type-description",
			Type:        "custom-type-type",
			Config: map[string]interface{}{
				"itemTypes": []string{"def1"},
			},
			Properties: []*state.CustomTypePropertyArgs{
				{
					Name:        "property-name",
					LocalID:     "property-local-id",
					Description: "property-description",
					Type:        "property-type",
					Config: map[string]interface{}{
						"enum": []interface{}{"value1", "value2"},
					},
					Required: true,
				},
			},
		},
	}

	t.Run("to resource data", func(t *testing.T) {
		t.Parallel()

		resourceData := customTypeState.ToResourceData()
		expected := resources.ResourceData{
			"id":              "upstream-custom-type-id",
			"name":            "custom-type-name",
			"description":     "custom-type-description",
			"type":            "custom-type-type",
			"version":         1,
			"dataType":        "object",
			"rules":           map[string]interface{}{},
			"itemDefinitions": []string{"def1"},
			"config": map[string]interface{}{
				"itemTypes": []string{"def1"},
			},
			"workspaceId": "workspace-id",
			"createdAt":   "2021-09-01T00:00:00Z",
			"updatedAt":   "2021-09-01T00:00:00Z",
			"properties": []map[string]interface{}{
				{
					"id":         "property-id",
					"localId":    "property-local-id",
					"propertyId": "upstream-property-id",
				},
			},
			"customTypeArgs": map[string]interface{}{
				"name":        "custom-type-name",
				"localId":     "custom-type-local-id",
				"description": "custom-type-description",
				"type":        "custom-type-type",
				"config": map[string]interface{}{
					"itemTypes": []interface{}{"def1"},
				},
				"properties": []map[string]interface{}{
					{
						"name":        "property-name",
						"localId":     "property-local-id",
						"description": "property-description",
						"type":        "property-type",
						"config":      map[string]interface{}{},
						"required":    true,
					},
				},
			},
		}
		assert.Equal(t, expected, resourceData)
	})

	t.Run("from resource data", func(t *testing.T) {
		t.Parallel()

		loopback := state.CustomTypeState{}
		loopback.FromResourceData(customTypeState.ToResourceData())
		assert.Equal(t, customTypeState, loopback)
	})
}
