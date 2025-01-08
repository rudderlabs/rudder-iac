package provider_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/pkg/provider"
	"github.com/rudderlabs/rudder-iac/cli/pkg/provider/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ client.DataCatalog = &MockEventCatalog{}

type MockEventCatalog struct {
	EmptyCatalog
	mockEvent *client.Event
	err       error
}

func (m *MockEventCatalog) SetEvent(event *client.Event) {
	m.mockEvent = event
}

func (m *MockEventCatalog) SetError(err error) {
	m.err = err
}

func (m *MockEventCatalog) CreateEvent(ctx context.Context, eventCreate client.EventCreate) (*client.Event, error) {
	return m.mockEvent, m.err
}

func (m *MockEventCatalog) UpdateEvent(ctx context.Context, id string, eventUpdate *client.Event) (*client.Event, error) {
	return m.mockEvent, m.err
}

func (m *MockEventCatalog) DeleteEvent(ctx context.Context, eventID string) error {
	return m.err
}

func TestEventProviderOperations(t *testing.T) {

	var (
		ctx           = context.Background()
		catalog       = &MockEventCatalog{}
		eventProvider = provider.NewEventProvider(catalog)
		created, _    = time.Parse(time.RFC3339, "2021-09-01T00:00:00Z")
		updated, _    = time.Parse(time.RFC3339, "2021-09-02T00:00:00Z")
	)

	toArgs := state.EventArgs{
		Name:        "event",
		Description: "event description",
		EventType:   "event type",
		CategoryID:  nil,
	}

	t.Run("Create", func(t *testing.T) {
		catalog.SetEvent(&client.Event{
			ID:          "upstream-event-catalog-id",
			Name:        "event",
			Description: "event description",
			EventType:   "event type",
			WorkspaceId: "workspace-id",
			CategoryId:  nil,
			CreatedAt:   created,
			UpdatedAt:   updated,
		})

		createdResource, err := eventProvider.Create(ctx, "event-id-1", "event", toArgs.ToResourceData())
		require.Nil(t, err)

		assert.Equal(t, resources.ResourceData{
			"id":          "upstream-event-catalog-id",
			"name":        "event",
			"description": "event description",
			"eventType":   "event type",
			"workspaceId": "workspace-id",
			"categoryId":  (*string)(nil),
			"createdAt":   "2021-09-01 00:00:00 +0000 UTC",
			"updatedAt":   "2021-09-02 00:00:00 +0000 UTC",
			"eventArgs": map[string]interface{}{
				"name":        "event",
				"description": "event description",
				"eventType":   "event type",
				"categoryId":  (*string)(nil),
			},
		}, *createdResource)
	})

	t.Run("Update", func(t *testing.T) {

		newArgs := state.EventArgs{
			Name:        "event",
			Description: "event new description",
			EventType:   "event type",
			CategoryID:  strptr("Marketing"),
		}

		prevState := state.EventState{
			ID:          "upstream-event-catalog-id",
			Name:        "event",
			Description: "event description",
			EventType:   "event type",
			WorkspaceID: "workspace-id",
			CategoryID:  nil,
			CreatedAt:   "2021-09-01 00:00:00 +0000 UTC",
			UpdatedAt:   "2021-09-01 00:00:00 +0000 UTC",
			EventArgs: state.EventArgs{
				Name:        "event",
				Description: "event description",
				EventType:   "event type",
				CategoryID:  nil,
			},
		}

		olds := prevState.ToResourceData()

		// We need a round of marshal / unmarshal to loose
		// the type information which would naturally happen
		// if we would inflate / deflate the state from file
		byt, err := json.Marshal(olds)
		require.Nil(t, err)

		err = json.Unmarshal(byt, &olds)
		require.Nil(t, err)

		// set the updated event which will be returned by the mock catalog
		catalog.SetEvent(&client.Event{
			ID:          "upstream-event-catalog-id",
			Name:        "event",
			Description: "event new description",
			EventType:   "event type",
			WorkspaceId: "workspace-id",
			CategoryId:  strptr("Marketing"),
			CreatedAt:   created,
			UpdatedAt:   updated,
		})

		updatedResource, err := eventProvider.Update(ctx, "event-id-1", "event", newArgs.ToResourceData(), olds)
		require.Nil(t, err)

		assert.Equal(t, resources.ResourceData{
			"id":          "upstream-event-catalog-id",
			"name":        "event",
			"description": "event new description", // actual update and rest same
			"eventType":   "event type",
			"workspaceId": "workspace-id",
			"categoryId":  strptr("Marketing"),
			"createdAt":   "2021-09-01 00:00:00 +0000 UTC",
			"updatedAt":   "2021-09-02 00:00:00 +0000 UTC",
			"eventArgs": map[string]interface{}{
				"name":        "event",
				"description": "event new description",
				"eventType":   "event type",
				"categoryId":  strptr("Marketing"),
			},
		}, *updatedResource)

	})

	t.Run("Delete", func(t *testing.T) {
		catalog.SetError(nil)
		err := eventProvider.Delete(
			ctx,
			"event-id-1",
			"event",
			resources.ResourceData{
				"id": "upstream-event-catalog-id",
			})
		require.Nil(t, err)
	})

}

func strptr(str string) *string {
	return &str
}
