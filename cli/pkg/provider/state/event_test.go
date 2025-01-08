package state_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/pkg/provider/state"
	"github.com/stretchr/testify/assert"
)

func TestEventState_ResourceData(t *testing.T) {

	eventState := state.EventState{
		ID:          "upstream-event-catalog-id",
		Name:        "event-name",
		Description: "event-description",
		EventType:   "event-type",
		WorkspaceID: "workspace-id",
		CategoryID:  nil,
		CreatedAt:   "2021-09-01T00:00:00Z",
		UpdatedAt:   "2021-09-01T00:00:00Z",
		EventArgs: state.EventArgs{
			Name:        "event-name",
			Description: "event-description",
			EventType:   "event-type",
			CategoryID:  nil,
		},
	}

	t.Run("converting to resource data", func(t *testing.T) {
		t.Parallel()

		var categoryID *string
		resourceData := eventState.ToResourceData()
		assert.Equal(t, resources.ResourceData{
			"id":          "upstream-event-catalog-id",
			"name":        "event-name",
			"description": "event-description",
			"eventType":   "event-type",
			"workspaceId": "workspace-id",
			"categoryId":  categoryID,
			"createdAt":   "2021-09-01T00:00:00Z",
			"updatedAt":   "2021-09-01T00:00:00Z",
			"eventArgs": map[string]interface{}{
				"name":        "event-name",
				"description": "event-description",
				"eventType":   "event-type",
				"categoryId":  categoryID,
			},
		}, resourceData)
	})

	t.Run("converting from resource data", func(t *testing.T) {
		t.Parallel()

		loopback := state.EventState{}
		loopback.FromResourceData(eventState.ToResourceData())
		assert.Equal(t, eventState, loopback)
	})

}

func TestEventArgs_ResourceData(t *testing.T) {

	args := state.EventArgs{
		Name:        "event-name",
		Description: "event-description",
		EventType:   "event-type",
		CategoryID:  nil,
	}

	t.Run("converting to resource data", func(t *testing.T) {
		t.Parallel()

		var categoryID *string
		resourceData := args.ToResourceData()
		assert.Equal(t, resources.ResourceData{
			"name":        "event-name",
			"description": "event-description",
			"eventType":   "event-type",
			"categoryId":  categoryID,
		}, resourceData)

	})

	t.Run("converting from resource data", func(t *testing.T) {
		t.Parallel()

		loopback := state.EventArgs{}
		loopback.FromResourceData(args.ToResourceData())
		assert.Equal(t, args, loopback)
	})

}
