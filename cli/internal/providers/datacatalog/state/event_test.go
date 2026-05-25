package state_test

import (
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
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
				"categoryId":  nil,
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
			"categoryId":  nil,
		}, resourceData)

	})

	t.Run("from resource data", func(t *testing.T) {
		t.Parallel()

		loopback := state.EventArgs{}
		loopback.FromResourceData(args.ToResourceData())
		assert.Equal(t, args, loopback)
	})

}

func TestEventArgs_FromCatalogEvent(t *testing.T) {
	t.Parallel()

	categoryRef := "category-local-id"
	args := &state.EventArgs{}
	args.FromCatalogEvent(&localcatalog.EventV1{
		LocalID:     "event-local-id",
		Name:        "Checkout Started",
		Type:        "track",
		Description: "User started checkout",
		CategoryRef: &categoryRef,
	}, func(ref string) string {
		return "category:" + ref
	})

	assert.Equal(t, state.EventArgs{
		Name:        "Checkout Started",
		Description: "User started checkout",
		EventType:   "track",
		CategoryId: &resources.PropertyRef{
			URN:      "category:category-local-id",
			Property: "id",
		},
	}, *args)
}

func TestEventArgs_FromCatalogEvent_NoCategory(t *testing.T) {
	t.Parallel()

	args := &state.EventArgs{}
	args.FromCatalogEvent(&localcatalog.EventV1{
		LocalID: "event-local-id",
		Name:    "Page Viewed",
		Type:    "page",
	}, func(ref string) string {
		return "category:" + ref
	})

	assert.Equal(t, state.EventArgs{
		Name:      "Page Viewed",
		EventType: "page",
	}, *args)
}

func TestEventArgs_DiffUpstream(t *testing.T) {
	cases := []struct {
		name     string
		args     state.EventArgs
		upstream *catalog.Event
		diffed   bool
	}{
		{
			name: "no diff",
			args: state.EventArgs{
				Name:        "event-name",
				Description: "event-description",
				EventType:   "track",
			},
			upstream: &catalog.Event{
				Name:        "event-name",
				Description: "event-description",
				EventType:   "track",
			},
			diffed: false,
		},
		{
			name: "name changed",
			args: state.EventArgs{Name: "old-name"},
			upstream: &catalog.Event{
				Name: "new-name",
			},
			diffed: true,
		},
		{
			name: "category added upstream",
			args: state.EventArgs{CategoryId: nil},
			upstream: &catalog.Event{
				CategoryId: stringPtr("category-id"),
			},
			diffed: true,
		},
		{
			name: "category id mismatch",
			args: state.EventArgs{CategoryId: "local-category"},
			upstream: &catalog.Event{
				CategoryId: stringPtr("remote-category"),
			},
			diffed: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.diffed, tc.args.DiffUpstream(tc.upstream))
		})
	}
}

func TestEventState_FromRemoteEvent(t *testing.T) {
	t.Parallel()

	categoryID := "category-123"
	now := time.Now()
	remoteEvent := &catalog.Event{
		ID:          "event-123",
		Name:        "Test Event",
		Description: "Test Description",
		EventType:   "track",
		CategoryId:  &categoryID,
		WorkspaceId: "workspace-789",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	eventState := &state.EventState{}
	err := eventState.FromRemoteEvent(remoteEvent, func(resourceType string, remoteId string) (string, error) {
		return "", resources.ErrRemoteResourceNotFound
	})

	require.NoError(t, err)
	assert.Equal(t, state.EventState{
		EventArgs: state.EventArgs{
			Name:        "Test Event",
			Description: "Test Description",
			EventType:   "track",
			CategoryId:  categoryID,
		},
		ID:          "event-123",
		Name:        "Test Event",
		Description: "Test Description",
		EventType:   "track",
		WorkspaceID: "workspace-789",
		CategoryID:  &categoryID,
		CreatedAt:   now.String(),
		UpdatedAt:   now.String(),
	}, *eventState)
}

func stringPtr(value string) *string {
	return &value
}

func TestEventArgs_FromRemoteEvent(t *testing.T) {
	t.Parallel()

	categoryID := "category-123"
	now := time.Now()

	// Create a mock getURNFromRemoteId function for the test
	getURNFromRemoteId := func(resourceType string, remoteId string) (string, error) {
		if resourceType == "category" && remoteId == categoryID {
			return "category:category-123-local", nil
		}
		return "", resources.ErrRemoteResourceNotFound
	}

	remoteEvent := &catalog.Event{
		ID:          "event-123",
		Name:        "Test Event",
		Description: "Test Description",
		EventType:   "track",
		CategoryId:  &categoryID,
		ExternalID:  "category-123-local",
		WorkspaceId: "workspace-789",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	args := &state.EventArgs{}
	err := args.FromRemoteEvent(remoteEvent, getURNFromRemoteId)

	assert.NoError(t, err)
	assert.Equal(t, "Test Event", args.Name)
	assert.Equal(t, "Test Description", args.Description)
	assert.Equal(t, "track", args.EventType)

	assert.NotNil(t, args.CategoryId)
	assert.IsType(t, &resources.PropertyRef{}, args.CategoryId)
	assert.Equal(t, "category:category-123-local", args.CategoryId.(*resources.PropertyRef).URN)
	assert.Equal(t, "id", args.CategoryId.(*resources.PropertyRef).Property)
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
		ExternalID:  "project-456",
		WorkspaceId: "workspace-789",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Create a mock getURNFromRemoteId function for the test
	getURNFromRemoteId := func(resourceType string, remoteId string) (string, error) {
		return "", resources.ErrRemoteResourceNotFound
	}

	args := &state.EventArgs{}
	err := args.FromRemoteEvent(remoteEvent, getURNFromRemoteId)

	assert.NoError(t, err)
	assert.Equal(t, "Test Event", args.Name)
	assert.Equal(t, "Test Description", args.Description)
	assert.Equal(t, "track", args.EventType)
	assert.Nil(t, args.CategoryId)
}

func TestEventArgs_FromRemoteEvent_NonCLIManagedCategory(t *testing.T) {
	t.Parallel()

	now := time.Now()

	categoryID := "category-123"
	remoteEvent := &catalog.Event{
		ID:          "event-123",
		Name:        "Test Event",
		Description: "Test Description",
		EventType:   "track",
		CategoryId:  &categoryID,
		ExternalID:  "project-456",
		WorkspaceId: "workspace-789",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Create a mock getURNFromRemoteId function for the test
	// we return ErrRemoteResourceExternalIdNotFound from our mock getURNFromRemoteId to simulate the case of a CLI managed event being connected to a non CLI managed category
	getURNFromRemoteId := func(resourceType string, remoteId string) (string, error) {
		return "", resources.ErrRemoteResourceExternalIdNotFound
	}

	args := &state.EventArgs{}
	err := args.FromRemoteEvent(remoteEvent, getURNFromRemoteId)

	assert.NoError(t, err)
	assert.Equal(t, "Test Event", args.Name)
	assert.Equal(t, "Test Description", args.Description)
	assert.Equal(t, "track", args.EventType)
	assert.Nil(t, args.CategoryId)
}
