package datacatalog_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ catalog.DataCatalog = &MockEventCatalog{}

type MockEventCatalog struct {
	datacatalog.EmptyCatalog
	mockEvent           *catalog.Event
	err                 error
	updateCalled        bool
	setExternalIdCalled bool
}

func (m *MockEventCatalog) SetEvent(event *catalog.Event) {
	m.mockEvent = event
}

func (m *MockEventCatalog) SetError(err error) {
	m.err = err
}

func (m *MockEventCatalog) CreateEvent(ctx context.Context, eventCreate catalog.EventCreate) (*catalog.Event, error) {
	return m.mockEvent, m.err
}

func (m *MockEventCatalog) UpdateEvent(ctx context.Context, id string, eventUpdate *catalog.EventUpdate) (*catalog.Event, error) {
	m.updateCalled = true
	if m.mockEvent == nil {
		return nil, m.err
	}
	m.mockEvent.Name = eventUpdate.Name
	m.mockEvent.Description = eventUpdate.Description
	m.mockEvent.EventType = eventUpdate.EventType
	m.mockEvent.CategoryId = eventUpdate.CategoryId

	return m.mockEvent, m.err
}

func (m *MockEventCatalog) DeleteEvent(ctx context.Context, eventID string) error {
	return m.err
}

func (m *MockEventCatalog) GetEvent(ctx context.Context, id string) (*catalog.Event, error) {
	return m.mockEvent, m.err
}

func (m *MockEventCatalog) SetEventExternalId(ctx context.Context, id string, externalId string) error {
	m.setExternalIdCalled = true
	if m.mockEvent == nil {
		return m.err
	}
	m.mockEvent.ExternalID = externalId
	return m.err
}

func (m *MockEventCatalog) ResetSpies() {
	m.updateCalled = false
	m.setExternalIdCalled = false
}

func TestEventProviderOperations(t *testing.T) {

	var (
		ctx           = context.Background()
		mockCatalog   = &MockEventCatalog{}
		eventProvider = datacatalog.NewEventProvider(mockCatalog, "test-import-dir")
		created, _    = time.Parse(time.RFC3339, "2021-09-01T00:00:00Z")
		updated, _    = time.Parse(time.RFC3339, "2021-09-02T00:00:00Z")
	)

	toArgs := state.EventArgs{
		Name:        "event",
		Description: "event description",
		EventType:   "event type",
		CategoryId:  nil,
	}

	t.Run("Create", func(t *testing.T) {
		mockCatalog.SetEvent(&catalog.Event{
			ID:          "upstream-event-catalog-id",
			Name:        "event",
			Description: "event description",
			EventType:   "event type",
			WorkspaceId: "workspace-id",
			ExternalID:  "event-id-1",
			CategoryId:  nil,
			CreatedAt:   created,
			UpdatedAt:   updated,
		})

		createdResource, err := eventProvider.Create(ctx, "event-id-1", toArgs.ToResourceData())
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
				"categoryId":  nil,
			},
		}, *createdResource)
	})

	t.Run("Update", func(t *testing.T) {

		newArgs := state.EventArgs{
			Name:        "event",
			Description: "event new description",
			EventType:   "event type",
			CategoryId:  "Marketing",
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
				CategoryId:  nil,
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
		mockCatalog.SetEvent(&catalog.Event{
			ID:          "upstream-event-catalog-id",
			Name:        "event",
			Description: "event new description",
			EventType:   "event type",
			WorkspaceId: "workspace-id",
			ExternalID:  "test-project-id",
			CategoryId:  strptr("Marketing"),
			CreatedAt:   created,
			UpdatedAt:   updated,
		})

		updatedResource, err := eventProvider.Update(ctx, "event-id-1", newArgs.ToResourceData(), olds)
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
				"categoryId":  "Marketing",
			},
		}, *updatedResource)

	})

	t.Run("Delete", func(t *testing.T) {
		mockCatalog.SetError(nil)
		err := eventProvider.Delete(
			ctx,
			"event-id-1",
			resources.ResourceData{
				"id": "upstream-event-catalog-id",
			})
		require.Nil(t, err)
	})

	t.Run("Import", func(t *testing.T) {
		tests := []struct {
			name           string
			localArgs      state.EventArgs
			remoteEvent    *catalog.Event
			mockErr        error
			expectErr      bool
			expectUpdate   bool
			expectSetExtId bool
			expectResource resources.ResourceData
		}{
			{
				name:           "successful import no differences",
				localArgs:      state.EventArgs{Name: "test-event", Description: "desc", EventType: "track", CategoryId: nil},
				remoteEvent:    &catalog.Event{ID: "remote-id", Name: "test-event", Description: "desc", EventType: "track", CategoryId: nil, WorkspaceId: "ws-id", CreatedAt: created, UpdatedAt: updated},
				mockErr:        nil,
				expectErr:      false,
				expectUpdate:   false,
				expectSetExtId: true,
				expectResource: resources.ResourceData{
					"id":          "remote-id",
					"name":        "test-event",
					"description": "desc",
					"eventType":   "track",
					"categoryId":  (*string)(nil),
					"workspaceId": "ws-id",
					"createdAt":   created.String(),
					"updatedAt":   updated.String(),
					"eventArgs":   map[string]interface{}{"name": "test-event", "description": "desc", "eventType": "track", "categoryId": nil},
				},
			},
			{
				name:           "successful import with differences",
				localArgs:      state.EventArgs{Name: "new-name", Description: "new-desc", EventType: "new-type", CategoryId: "cat-id"},
				remoteEvent:    &catalog.Event{ID: "remote-id", Name: "old-name", Description: "old-desc", EventType: "old-type", CategoryId: nil, WorkspaceId: "ws-id", CreatedAt: created, UpdatedAt: updated},
				mockErr:        nil,
				expectErr:      false,
				expectUpdate:   true,
				expectSetExtId: true,
				expectResource: resources.ResourceData{
					"id":          "remote-id",
					"name":        "new-name",
					"description": "new-desc",
					"eventType":   "new-type",
					"categoryId":  strptr("cat-id"),
					"workspaceId": "ws-id",
					"createdAt":   created.String(),
					"updatedAt":   updated.String(), // Note: Actual test will use the updated time from mock
					"eventArgs":   map[string]interface{}{"name": "new-name", "description": "new-desc", "eventType": "new-type", "categoryId": "cat-id"},
				},
			},
			{
				name:        "error on get event",
				localArgs:   state.EventArgs{Name: "test"},
				remoteEvent: nil,
				mockErr:     errors.New("get failed"),
				expectErr:   true,
			},
			{
				name:         "error on update",
				localArgs:    state.EventArgs{Name: "new-name"},
				remoteEvent:  &catalog.Event{ID: "remote-id", Name: "old-name"},
				mockErr:      errors.New("update failed"),
				expectErr:    true,
				expectUpdate: true,
			},
			{
				name:           "error on set external ID",
				localArgs:      state.EventArgs{Name: "test"},
				remoteEvent:    &catalog.Event{ID: "remote-id", Name: "test"},
				mockErr:        errors.New("set ext id failed"),
				expectErr:      true,
				expectSetExtId: true,
			},
			// Additional case: Handling nil CategoryId and special EventType like "identify" with empty name
			{
				name:           "successful import for identify event",
				localArgs:      state.EventArgs{Name: "", Description: "identify desc", EventType: "identify", CategoryId: nil},
				remoteEvent:    &catalog.Event{ID: "remote-id", Name: "", Description: "identify desc", EventType: "identify", CategoryId: nil, WorkspaceId: "ws-id", CreatedAt: created, UpdatedAt: updated},
				mockErr:        nil,
				expectErr:      false,
				expectUpdate:   false,
				expectSetExtId: true,
				expectResource: resources.ResourceData{
					"id":          "remote-id",
					"name":        "",
					"description": "identify desc",
					"eventType":   "identify",
					"categoryId":  (*string)(nil),
					"workspaceId": "ws-id",
					"createdAt":   created.String(),
					"updatedAt":   updated.String(),
					"eventArgs":   map[string]interface{}{"name": "", "description": "identify desc", "eventType": "identify", "categoryId": nil},
				},
			},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				mockCatalog.ResetSpies()
				mockCatalog.SetEvent(tt.remoteEvent)
				mockCatalog.SetError(tt.mockErr)

				res, err := eventProvider.Import(ctx, "local-event-id", tt.localArgs.ToResourceData(), "remote-id")

				if tt.expectErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				var (
					expected = tt.expectResource
					actual   = *res
				)
				assert.Equal(t, expected, actual)
				assert.Equal(t, tt.expectUpdate, mockCatalog.updateCalled)
				assert.Equal(t, tt.expectSetExtId, mockCatalog.setExternalIdCalled)
			})
		}
	})
}

func strptr(str string) *string {
	return &str
}

// Testing the error message we show to user when api returns a duplicate event error
// For track event the api returns a message that contains the event name
// For the rest of events like identify the api returns the same message but with out the event name
// For example: Event with name  already exists" for identify event
// "Event with name Signup Click already exists" for track event
func TestEventProviderDuplicateError(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name     string
		args     state.EventArgs
		err      error
		expected string
	}{
		{
			name: "duplicate track event with name",
			args: state.EventArgs{
				Name:        "Signup Click",
				Description: "desc",
				EventType:   "track",
			},
			err: &client.APIError{
				HTTPStatusCode: 400,
				Message:        "Event with name Signup Click already exists",
			},
			expected: "track event 'Signup Click' already exists",
		},
		{
			name: "duplicate identify event without name",
			args: state.EventArgs{
				Name:        "",
				Description: "identify desc",
				EventType:   "identify",
			},
			err: &client.APIError{
				HTTPStatusCode: 400,
				Message:        "Event with name  already exists",
			},
			expected: "identify event already exists",
		},
		{
			name: "not an API error",
			args: state.EventArgs{
				Name:        "Signup Click",
				Description: "desc",
				EventType:   "track",
			},
			err:      errors.New("not an API error"),
			expected: "creating event in upstream catalog: not an API error",
		},
		{
			name: "internal server error",
			args: state.EventArgs{
				Name:        "Signup Click",
				Description: "desc",
				EventType:   "track",
			},
			err:      &client.APIError{HTTPStatusCode: 500, Message: "unexpected error", ErrorCode: "500"},
			expected: "creating event in upstream catalog: http status code: 500, error code: '500', error: 'unexpected error'",
		},
	}

	for _, tc := range cases {
		c := tc
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			mockCatalog := &MockEventCatalog{}
			mockCatalog.SetError(c.err)
			eventProvider := datacatalog.NewEventProvider(mockCatalog, "test-import-dir")

			_, err := eventProvider.Create(ctx, "event-id", c.args.ToResourceData())
			require.Error(t, err)
			assert.Equal(t, c.expected, err.Error())
		})
	}
}
