package state_test

import (
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventState_ResourceData(t *testing.T) {

	eventState := state.EventState{
		ID:          "upstream-event-catalog-id",
		Name:        "event-name",
		Description: "event-description",
		EventType:   "event-type",
		WorkspaceID: "workspace-id",
		CreatedAt:   "2021-09-01T00:00:00Z",
		UpdatedAt:   "2021-09-01T00:00:00Z",
		EventArgs: state.EventArgs{
			Name:        "event-name",
			Description: "event-description",
			EventType:   "event-type",
			CategoryId:  nil,
		},
	}

	t.Run("to resource data", func(t *testing.T) {
		t.Parallel()

		resourceData := eventState.ToResourceData()
		assert.Equal(t, resources.ResourceData{
			"id":          "upstream-event-catalog-id",
			"name":        "event-name",
			"description": "event-description",
			"eventType":   "event-type",
			"workspaceId": "workspace-id",
			"categoryId":  (*string)(nil),
			"createdAt":   "2021-09-01T00:00:00Z",
			"updatedAt":   "2021-09-01T00:00:00Z",
			"eventArgs": map[string]interface{}{
				"name":        "event-name",
				"description": "event-description",
				"eventType":   "event-type",
				"categoryId":  (*resources.PropertyRef)(nil),
			},
		}, resourceData)
	})

	t.Run("from resource data", func(t *testing.T) {
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
		CategoryId:  nil,
	}

	t.Run("to resource data", func(t *testing.T) {
		t.Parallel()

		resourceData := args.ToResourceData()
		assert.Equal(t, resources.ResourceData{
			"name":        "event-name",
			"description": "event-description",
			"eventType":   "event-type",
			"categoryId":  (*resources.PropertyRef)(nil),
		}, resourceData)

	})

	t.Run("from resource data", func(t *testing.T) {
		t.Parallel()

		loopback := state.EventArgs{}
		loopback.FromResourceData(args.ToResourceData())
		assert.Equal(t, args, loopback)
	})

}

func TestEventArgs_FromRemoteEvent(t *testing.T) {
	t.Parallel()

	categoryID := "category-123"
	now := time.Now()

	// Create a resource collection for the test
	resourceCollection := resources.NewResourceCollection()
	resourceCollection.SetCategories([]*catalog.Category{
		{
			ID:        categoryID,
			ProjectId: "category-123-local",
			Name:      "Test Category",
		},
	})

	remoteEvent := &catalog.Event{
		ID:          "event-123",
		Name:        "Test Event",
		Description: "Test Description",
		EventType:   "track",
		CategoryId:  &categoryID,
		ProjectId:   "project-456",
		WorkspaceId: "workspace-789",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	args := &state.EventArgs{}
	args.FromRemoteEvent(remoteEvent, resourceCollection)

	assert.Equal(t, "project-456", args.ProjectId)
	assert.Equal(t, "Test Event", args.Name)
	assert.Equal(t, "Test Description", args.Description)
	assert.Equal(t, "track", args.EventType)

	assert.NotNil(t, args.CategoryId)
	assert.Equal(t, "category:category-123-local", args.CategoryId.URN)
	assert.Equal(t, "id", args.CategoryId.Property)
}

func TestEventArgs_FromRemoteEvent_NoCategory(t *testing.T) {
	t.Parallel()

	now := time.Now()

	remoteEvent := &catalog.Event{
		ID:          "event-123",
		Name:        "Test Event",
		Description: "Test Description",
		EventType:   "track",
		CategoryId:  nil,
		ProjectId:   "project-456",
		WorkspaceId: "workspace-789",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Create a resource collection for the test
	resourceCollection := resources.NewResourceCollection()
	require.NotNil(t, resourceCollection)

	args := &state.EventArgs{}
	args.FromRemoteEvent(remoteEvent, resourceCollection)

	assert.Equal(t, "project-456", args.ProjectId)
	assert.Equal(t, "Test Event", args.Name)
	assert.Equal(t, "Test Description", args.Description)
	assert.Equal(t, "track", args.EventType)
	assert.Nil(t, args.CategoryId)
}
