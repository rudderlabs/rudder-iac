package state_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/pkg/provider/state"
	"github.com/stretchr/testify/assert"
)

func TestPropertyArgs_ResourceData(t *testing.T) {

	argsWithConfig := state.PropertyArgs{
		Name:        "property-name",
		Description: "property-description",
		Type:        "property-type",
		Config: map[string]interface{}{
			"enum": []string{"value1", "value2"},
		},
	}

	argsWithoutConfig := state.PropertyArgs{
		Name:        "property-name",
		Description: "property-description",
		Type:        "property-type",
		Config:      nil,
	}

	t.Run("to resource data", func(t *testing.T) {
		t.Parallel()

		resourceData := argsWithConfig.ToResourceData()
		assert.Equal(t, resources.ResourceData{
			"name":        "property-name",
			"description": "property-description",
			"type":        "property-type",
			"config": map[string]interface{}{
				"enum": []string{"value1", "value2"},
			},
		}, resourceData)

		resourceData = argsWithoutConfig.ToResourceData()
		assert.Equal(t, resources.ResourceData{
			"name":        "property-name",
			"description": "property-description",
			"type":        "property-type",
			"config":      map[string]interface{}(nil),
		}, resourceData)
	})

	t.Run("from resource data", func(t *testing.T) {
		t.Parallel()

		loopback := state.PropertyArgs{}
		loopback.FromResourceData(argsWithConfig.ToResourceData())
		assert.Equal(t, argsWithConfig, loopback)

		loopback = state.PropertyArgs{}
		loopback.FromResourceData(argsWithoutConfig.ToResourceData())
		assert.Equal(t, argsWithoutConfig, loopback)
	})

}

func TestPropertyState_ResourceData(t *testing.T) {

	propertyState := state.PropertyState{
		PropertyArgs: state.PropertyArgs{
			Name:        "property-name",
			Description: "property-description",
			Type:        "property-type",
			Config:      nil,
		},
		ID:          "upstream-property-id",
		Name:        "property-name",
		Description: "property-description",
		Type:        "property-type",
		WorkspaceID: "workspace-id",
		CreatedAt:   "2021-09-01T00:00:00Z",
		UpdatedAt:   "2021-09-01T00:00:00Z",
		Config:      nil,
	}

	t.Run("to resource data", func(t *testing.T) {
		t.Parallel()

		resourceData := propertyState.ToResourceData()
		assert.Equal(t, resources.ResourceData{
			"id":          "upstream-property-id",
			"name":        "property-name",
			"description": "property-description",
			"type":        "property-type",
			"config":      map[string]interface{}(nil),
			"workspaceId": "workspace-id",
			"createdAt":   "2021-09-01T00:00:00Z",
			"updatedAt":   "2021-09-01T00:00:00Z",
			"propertyArgs": map[string]interface{}{
				"name":        "property-name",
				"description": "property-description",
				"type":        "property-type",
				"config":      map[string]interface{}(nil),
			},
		}, resourceData)
	})

	t.Run("from resource data", func(t *testing.T) {
		t.Parallel()

		loopback := state.PropertyState{}
		loopback.FromResourceData(propertyState.ToResourceData())
		assert.Equal(t, propertyState, loopback)
	})

}
