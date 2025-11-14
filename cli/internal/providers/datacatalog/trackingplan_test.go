package datacatalog_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/testutils/factory"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

var _ catalog.DataCatalog = &MockTrackingPlanCatalog{}

type MockTrackingPlanCatalog struct {
	datacatalog.EmptyCatalog
	tp                   *catalog.TrackingPlan
	tpWithSchema         *catalog.TrackingPlanWithSchemas
	tpWithIdentifiers    *catalog.TrackingPlanWithIdentifiers
	tpes                 *catalog.TrackingPlanEventSchema
	err                  error
	updateCalled         bool
	setExternalIdCalled  bool
	deleteEventCalled    bool
	updateEventCalled    bool
	deleteEventCallCount int
	updateEventCallCount int
}

func (m *MockTrackingPlanCatalog) CreateTrackingPlan(ctx context.Context, trackingPlanCreate catalog.TrackingPlanCreate) (*catalog.TrackingPlan, error) {
	return m.tp, m.err
}

func (m *MockTrackingPlanCatalog) UpdateTrackingPlan(ctx context.Context, trackingPlanID string, name string, description string) (*catalog.TrackingPlan, error) {
	m.updateCalled = true
	if m.tp != nil {
		m.tp.Name = name
		m.tp.Description = &description
	}
	if m.tpWithIdentifiers != nil {
		m.tpWithIdentifiers.Name = name
		m.tpWithIdentifiers.Description = &description
	}
	return m.tp, m.err
}

func (m *MockTrackingPlanCatalog) DeleteTrackingPlan(ctx context.Context, trackingPlanID string) error {
	return m.err
}

func (m *MockTrackingPlanCatalog) DeleteTrackingPlanEvent(ctx context.Context, trackingPlanID string, eventID string) error {
	m.deleteEventCalled = true
	m.deleteEventCallCount++
	return m.err
}

func (m *MockTrackingPlanCatalog) UpsertTrackingPlan(ctx context.Context, trackingPlanID string, trackingPlanUpsertEvent catalog.TrackingPlanUpsertEvent) (*catalog.TrackingPlan, error) {
	return m.tp, m.err
}

func (m *MockTrackingPlanCatalog) GetTrackingPlan(ctx context.Context, id string) (*catalog.TrackingPlanWithIdentifiers, error) {
	return m.tpWithIdentifiers, m.err
}

func (m *MockTrackingPlanCatalog) GetTrackingPlanWithSchemas(ctx context.Context, id string) (*catalog.TrackingPlanWithSchemas, error) {
	return m.tpWithSchema, m.err
}

func (m *MockTrackingPlanCatalog) GetTrackingPlanEventSchema(ctx context.Context, id string, eventId string) (*catalog.TrackingPlanEventSchema, error) {
	return m.tpes, m.err
}

func (m *MockTrackingPlanCatalog) UpdateTrackingPlanEvents(ctx context.Context, id string, inputs []catalog.EventIdentifierDetail) (*catalog.TrackingPlan, error) {
	m.updateEventCalled = true
	m.updateEventCallCount++
	return m.tp, m.err
}

func (m *MockTrackingPlanCatalog) SetTrackingPlan(tp *catalog.TrackingPlan) {
	m.tp = tp
}

func (m *MockTrackingPlanCatalog) SetTrackingPlanEventSchema(tpes *catalog.TrackingPlanEventSchema) {
	m.tpes = tpes
}

func (m *MockTrackingPlanCatalog) SetError(err error) {
	m.err = err
}

func (m *MockTrackingPlanCatalog) SetTrackingPlanExternalId(ctx context.Context, id string, externalId string) error {
	m.setExternalIdCalled = true
	if m.tpWithIdentifiers != nil {
		m.tpWithIdentifiers.ExternalID = externalId
	}
	return m.err
}

func (m *MockTrackingPlanCatalog) SetTrackingPlanWithIdentifiers(tpWithIdentifiers *catalog.TrackingPlanWithIdentifiers) {
	m.tpWithIdentifiers = tpWithIdentifiers
}

func (m *MockTrackingPlanCatalog) ResetSpies() {
	m.updateCalled = false
	m.setExternalIdCalled = false
	m.deleteEventCalled = false
	m.updateEventCalled = false
	m.deleteEventCallCount = 0
	m.updateEventCallCount = 0
}

func TestTrackingPlanProvider_Create(t *testing.T) {
	t.Parallel()

	var (
		ctx         = context.Background()
		mockCatalog = &MockTrackingPlanCatalog{}
		provider    = datacatalog.NewTrackingPlanProvider(mockCatalog, "data-catalog")
	)

	var (
		toArgs = getTrackingPlanArgs()
	)

	mockCatalog.SetTrackingPlan(getTestTrackingPlan())
	newState, err := provider.Create(ctx, "tracking-plan-id", toArgs.ToResourceData())
	require.Nil(t, err)
	assert.Equal(t, resources.ResourceData{
		"id":           "upstream-tracking-plan-id",
		"name":         "tracking-plan",
		"description":  "tracking-plan-description",
		"workspaceId":  "workspace-id",
		"creationType": "backend",
		"createdAt":    "2021-09-01 00:00:00 +0000 UTC",
		"updatedAt":    "2021-09-02 00:00:00 +0000 UTC",
		"version":      1,
		"events": []map[string]interface{}{
			{
				"eventId": "upstream-event-id",
				"localId": "event-id",
			},
		},
		"trackingPlanArgs": map[string]interface{}{
			"name":        "tracking-plan",
			"localId":     "tracking-plan-id",
			"description": "tracking-plan-description",
			"events": []map[string]interface{}{
				{
					"id":              "upstream-event-id",
					"localId":         "event-id",
					"allowUnplanned":  false,
					"identitySection": "",
					"properties": []map[string]interface{}{
						{
							"localId":              "property-id",
							"id":                   "",
							"required":             true,
							"additionalProperties": false,
						},
					},
					"variants": []map[string]interface{}{},
				},
			},
		},
	}, *newState)

}

func TestTrackingPlanProvider_Update(t *testing.T) {
	t.Parallel()

	var (
		ctx         = context.Background()
		mockCatalog = &MockTrackingPlanCatalog{}
		provider    = datacatalog.NewTrackingPlanProvider(mockCatalog, "data-catalog")
	)

	var (
		oldsArgs = getTrackingPlanArgs()
	)

	oldsState := defaultTrackingPlanStateFactory().
		WithEvent(&state.TrackingPlanEventState{
			LocalID: "event-id",
			EventID: "upstream-event-id",
		}).
		WithTrackingPlanArgs(*oldsArgs).
		Build()

	// // the default tracking plan with version
	updatedTP := defaultTrackingPlanFactory().
		WithDescription("tracking-plan-updated-description"). // updated description
		WithEvent(catalog.TrackingPlanEvent{
			ID:             "upstream-tracking-plan-event-id",
			TrackingPlanID: "tracking-plan-id",
			SchemaID:       "upstream-schema-id",
			EventID:        "upstream-event-id",
		}).
		WithVersion(2).
		Build()

	mockCatalog.SetTrackingPlan(&updatedTP)

	toArgs := defaultTrackingPlanArgsFactory().
		WithDescription("tracking-plan-updated-description"). // updated description
		WithEvent(&state.TrackingPlanEventArgs{
			LocalID:        "event-id",
			AllowUnplanned: false,
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					LocalID:  "property-id",
					Required: true,
				},
			},
		}).Build()

	olds := marshalUnmarshal(t, oldsState.ToResourceData())
	newState, err := provider.Update(ctx, "tracking-plan-id", toArgs.ToResourceData(), olds)
	require.Nil(t, err)

	assert.Equal(t, resources.ResourceData{
		"id":           "upstream-tracking-plan-id",
		"name":         "tracking-plan",
		"description":  "tracking-plan-updated-description", // updated description
		"workspaceId":  "workspace-id",
		"creationType": "backend",
		"createdAt":    "2021-09-01 00:00:00 +0000 UTC",
		"updatedAt":    "2021-09-02 00:00:00 +0000 UTC",
		"version":      2,
		"events": []map[string]interface{}{
			{
				"eventId": "upstream-event-id",
				"localId": "event-id",
			},
		},
		"trackingPlanArgs": map[string]interface{}{
			"name":        "tracking-plan",
			"localId":     "tracking-plan-id",
			"description": "tracking-plan-updated-description", // updated description
			"events": []map[string]interface{}{
				{
					"localId":         "event-id",
					"id":              "",
					"allowUnplanned":  false,
					"identitySection": "",
					"properties": []map[string]interface{}{
						{
							"id":                   "",
							"localId":              "property-id",
							"required":             true,
							"additionalProperties": false,
						},
					},
					"variants": []map[string]interface{}{},
				},
			},
		},
	}, *newState)
}

func TestTrackingPlanProvider_UpdateWithUpsertEvent(t *testing.T) {
	t.Parallel()

	var (
		ctx         = context.Background()
		mockCatalog = &MockTrackingPlanCatalog{}
		provider    = datacatalog.NewTrackingPlanProvider(mockCatalog, "data-catalog")
	)

	var (
		oldsArgs = getTrackingPlanArgs()
	)

	oldsState := defaultTrackingPlanStateFactory().
		WithEvent(&state.TrackingPlanEventState{
			LocalID: "event-id",
			EventID: "upstream-event-id",
		}).
		WithTrackingPlanArgs(*oldsArgs).
		Build()

	toArgs := defaultTrackingPlanArgsFactory().
		WithDescription("tracking-plan-updated-description"). // updated description
		WithEvent(&state.TrackingPlanEventArgs{               // updated events under the trackingplan +1 Added -1 Removed
			ID:             "upstream-event-id-1",
			LocalID:        "event-id-1",
			AllowUnplanned: true,
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					LocalID:  "property-id-1",
					Required: false,
				},
			},
		}).Build()

	updatedTP := defaultTrackingPlanFactory().
		WithDescription("tracking-plan-updated-description"). // updated description
		WithVersion(2).
		WithEvent(catalog.TrackingPlanEvent{
			ID:             "upstream-tracking-plan-event-id-1",
			TrackingPlanID: "tracking-plan-id",
			SchemaID:       "upstream-schema-id-1",
			EventID:        "upsream-event-id-1",
		}).Build()

	mockCatalog.SetError(nil)
	mockCatalog.SetTrackingPlan(&updatedTP)

	olds := marshalUnmarshal(t, oldsState.ToResourceData())

	newState, err := provider.Update(ctx, "tracking-plan-id", toArgs.ToResourceData(), olds)
	require.Nil(t, err)

	require.Equal(t, resources.ResourceData{
		"id":           "upstream-tracking-plan-id",
		"name":         "tracking-plan",
		"description":  "tracking-plan-updated-description", // updated description
		"workspaceId":  "workspace-id",
		"creationType": "backend",
		"createdAt":    "2021-09-01 00:00:00 +0000 UTC",
		"updatedAt":    "2021-09-02 00:00:00 +0000 UTC",
		"version":      2,
		"events": []map[string]interface{}{
			{
				"eventId": "upstream-event-id-1",
				"localId": "event-id-1",
			},
		},
		"trackingPlanArgs": map[string]interface{}{
			"name":        "tracking-plan",
			"localId":     "tracking-plan-id",
			"description": "tracking-plan-updated-description", // updated description
			"events": []map[string]interface{}{
				{
					"id":              "upstream-event-id-1",
					"localId":         "event-id-1",
					"allowUnplanned":  true,
					"identitySection": "",
					"properties": []map[string]interface{}{
						{
							"id":                   "",
							"localId":              "property-id-1",
							"required":             false,
							"additionalProperties": false,
						},
					},
					"variants": []map[string]interface{}{},
				},
			},
		},
	}, *newState)

}

func TestTrackingPlanProvider_Diff(t *testing.T) {
	t.Parallel()

	// 2 Added 1 Removed 1 Updated
	oldArgs := defaultTrackingPlanArgsFactory().
		WithEvent(&state.TrackingPlanEventArgs{
			LocalID:        "event-id",
			AllowUnplanned: false,
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					LocalID:  "property-id",
					Required: true,
				},
			},
		}).
		WithEvent(&state.TrackingPlanEventArgs{
			LocalID:        "event-id-1",
			AllowUnplanned: true,
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					LocalID:  "property-id",
					Required: true,
				},
			},
		}).
		Build()

	newArgs := defaultTrackingPlanArgsFactory().
		WithEvent(&state.TrackingPlanEventArgs{
			LocalID:        "event-id",
			AllowUnplanned: true, // updated
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					LocalID:  "property-id",
					Required: true,
				},
				{
					LocalID:  "property-id-1",
					Required: false,
				},
			},
		}).
		WithEvent(&state.TrackingPlanEventArgs{
			LocalID:        "event-id-2",
			AllowUnplanned: true,
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					LocalID:  "property-id-2",
					Required: false,
				},
			},
		}).Build()

	diffed := oldArgs.Diff(newArgs)
	require.Equal(t, 1, len(diffed.Added))
	require.Equal(t, 1, len(diffed.Updated))
	require.Equal(t, 1, len(diffed.Deleted))

	assert.Equal(t, &state.TrackingPlanEventArgs{
		LocalID:        "event-id-2",
		AllowUnplanned: true,
		Properties: []*state.TrackingPlanPropertyArgs{
			{
				LocalID:  "property-id-2",
				Required: false,
			},
		},
	}, diffed.Added[0])

	assert.Equal(t, &state.TrackingPlanEventArgs{
		LocalID:        "event-id-1",
		AllowUnplanned: true,
		Properties: []*state.TrackingPlanPropertyArgs{
			{
				LocalID:  "property-id",
				Required: true,
			},
		},
	}, diffed.Deleted[0])

	assert.Equal(t, &state.TrackingPlanEventArgs{
		LocalID:        "event-id",
		AllowUnplanned: true,
		Properties: []*state.TrackingPlanPropertyArgs{
			{
				LocalID:  "property-id",
				Required: true,
			},
			{
				LocalID:  "property-id-1",
				Required: false,
			},
		},
	}, diffed.Updated[0])

}

func marshalUnmarshal(t *testing.T, input resources.ResourceData) resources.ResourceData {
	byt, err := json.Marshal(input)
	require.Nil(t, err)

	var output resources.ResourceData
	err = json.Unmarshal(byt, &output)
	require.Nil(t, err)

	return output
}

func TestTrackingPlanProvider_Delete(t *testing.T) {
	t.Parallel()

	var (
		ctx      = context.Background()
		provider = datacatalog.NewTrackingPlanProvider(&MockTrackingPlanCatalog{}, "data-catalog")
	)

	err := provider.Delete(ctx, "tracking-plan-id", resources.ResourceData{"id": "upstream-tracking-plan-id"})
	require.Nil(t, err)
}

func TestTrackingPlanProvider_Import(t *testing.T) {
	var (
		ctx         = context.Background()
		mockCatalog = &MockTrackingPlanCatalog{}
		provider    = datacatalog.NewTrackingPlanProvider(mockCatalog, "data-catalog")
		created, _  = time.Parse(time.RFC3339, "2021-09-01T00:00:00Z")
		updated, _  = time.Parse(time.RFC3339, "2021-09-02T00:00:00Z")
	)

	tests := []struct {
		name                   string
		localArgs              state.TrackingPlanArgs
		remoteTrackingPlan     *catalog.TrackingPlanWithIdentifiers
		mockErr                error
		expectErr              bool
		expectUpdate           bool
		expectSetExtId         bool
		expectDeleteEventCount int
		expectUpdateEventCount int
		expectResource         *resources.ResourceData
	}{
		{
			name: "successful import no differences",
			localArgs: state.TrackingPlanArgs{
				LocalID:     "local-tp-id",
				Name:        "TestTP",
				Description: "desc",
				Events: []*state.TrackingPlanEventArgs{
					{
						LocalID:        "event-1",
						ID:             "remote-event-1",
						AllowUnplanned: false,
						Properties: []*state.TrackingPlanPropertyArgs{
							{LocalID: "prop-1", ID: "remote-prop-1", Required: true},
						},
					},
				},
			},
			remoteTrackingPlan: &catalog.TrackingPlanWithIdentifiers{
				ID:           "remote-tp-id",
				Name:         "TestTP",
				Description:  strptr("desc"),
				Version:      1,
				CreationType: "backend",
				WorkspaceID:  "ws-id",
				CreatedAt:    created,
				UpdatedAt:    updated,
				Events: []*catalog.TrackingPlanEventPropertyIdentifiers{
					{
						ID:                   "remote-event-1",
						AdditionalProperties: false,
						IdentitySection:      "",
						Properties: []*catalog.TrackingPlanEventProperty{
							{ID: "remote-prop-1", Required: true},
						},
					},
				},
			},
			mockErr:                nil,
			expectErr:              false,
			expectUpdate:           false,
			expectSetExtId:         true,
			expectDeleteEventCount: 0,
			expectUpdateEventCount: 0,
			expectResource: &resources.ResourceData{
				"id":           "remote-tp-id",
				"name":         "TestTP",
				"description":  "desc",
				"version":      1,
				"creationType": "backend",
				"workspaceId":  "ws-id",
				"createdAt":    created.String(),
				"updatedAt":    updated.String(),
				"events":       []map[string]any(nil),
				"trackingPlanArgs": map[string]any{
					"localId":     "local-tp-id",
					"name":        "TestTP",
					"description": "desc",
					"events": []map[string]any{
						{
							"localId":         "event-1",
							"id":              "remote-event-1",
							"allowUnplanned":  false,
							"identitySection": "",
							"properties": []map[string]any{
								{
									"localId":              "prop-1",
									"id":                   "remote-prop-1",
									"required":             true,
									"additionalProperties": false,
								},
							},
							"variants": []map[string]any{},
						},
					},
				},
			},
		},
		{
			name: "successful import with differences",
			localArgs: state.TrackingPlanArgs{
				LocalID:     "local-tp-id",
				Name:        "UpdatedTP",
				Description: "new desc",
				Events: []*state.TrackingPlanEventArgs{
					{
						// Event that exists in remote but with changes (UPDATE)
						LocalID:        "event-1",
						ID:             "remote-event-1",
						AllowUnplanned: true, // changed from false
						Properties: []*state.TrackingPlanPropertyArgs{
							{LocalID: "prop-1", ID: "remote-prop-1", Required: false}, // changed from true
						},
					},
					{
						// New event that doesn't exist in remote (ADD)
						LocalID:        "event-3",
						ID:             "remote-event-3",
						AllowUnplanned: false,
						Properties: []*state.TrackingPlanPropertyArgs{
							{LocalID: "prop-3", ID: "remote-prop-3", Required: true},
						},
					},
				},
			},
			remoteTrackingPlan: &catalog.TrackingPlanWithIdentifiers{
				ID:           "remote-tp-id",
				Name:         "TestTP",
				Description:  strptr("desc"),
				Version:      1,
				CreationType: "backend",
				WorkspaceID:  "ws-id",
				CreatedAt:    created,
				UpdatedAt:    updated,
				Events: []*catalog.TrackingPlanEventPropertyIdentifiers{
					{
						// Event that will be updated
						ID:                   "remote-event-1",
						AdditionalProperties: false,
						IdentitySection:      "",
						Properties: []*catalog.TrackingPlanEventProperty{
							{ID: "remote-prop-1", Required: true},
						},
					},
					{
						// Event that exists in remote but not in local (DELETE)
						ID:                   "remote-event-2",
						AdditionalProperties: true,
						IdentitySection:      "",
						Properties: []*catalog.TrackingPlanEventProperty{
							{ID: "remote-prop-2", Required: false},
						},
					},
				},
			},
			mockErr:                nil,
			expectErr:              false,
			expectUpdate:           true,
			expectSetExtId:         true,
			expectDeleteEventCount: 1, // one event deleted (event-2)
			expectUpdateEventCount: 2, // one event updated (event-1) + one event added (event-3)
			expectResource: &resources.ResourceData{
				"id":           "remote-tp-id",
				"name":         "UpdatedTP",
				"description":  "new desc",
				"version":      1,
				"creationType": "backend",
				"workspaceId":  "ws-id",
				"createdAt":    created.String(),
				"updatedAt":    updated.String(),
				"events":       []map[string]any(nil),
				"trackingPlanArgs": map[string]any{
					"localId":     "local-tp-id",
					"name":        "UpdatedTP",
					"description": "new desc",
					"events": []map[string]any{
						{
							"localId":         "event-1",
							"id":              "remote-event-1",
							"allowUnplanned":  true,
							"identitySection": "",
							"properties": []map[string]any{
								{
									"localId":              "prop-1",
									"id":                   "remote-prop-1",
									"required":             false,
									"additionalProperties": false,
								},
							},
							"variants": []map[string]any{},
						},
						{
							"localId":         "event-3",
							"id":              "remote-event-3",
							"allowUnplanned":  false,
							"identitySection": "",
							"properties": []map[string]any{
								{
									"localId":              "prop-3",
									"id":                   "remote-prop-3",
									"required":             true,
									"additionalProperties": false,
								},
							},
							"variants": []map[string]any{},
						},
					},
				},
			},
		},
		{
			name: "error on get tracking plan",
			localArgs: state.TrackingPlanArgs{
				LocalID:     "local-tp-id",
				Name:        "TestTP",
				Description: "desc",
			},
			remoteTrackingPlan: nil,
			mockErr:            assert.AnError,
			expectErr:          true,
		},
		{
			name: "error on set external ID",
			localArgs: state.TrackingPlanArgs{
				LocalID:     "local-tp-id",
				Name:        "TestTP",
				Description: "desc",
			},
			remoteTrackingPlan: &catalog.TrackingPlanWithIdentifiers{
				ID:           "remote-tp-id",
				Name:         "TestTP",
				Description:  strptr("desc"),
				Version:      1,
				CreationType: "backend",
				WorkspaceID:  "ws-id",
				CreatedAt:    created,
				UpdatedAt:    updated,
				Events:       []*catalog.TrackingPlanEventPropertyIdentifiers{},
			},
			mockErr:        assert.AnError,
			expectErr:      true,
			expectSetExtId: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCatalog.ResetSpies()
			mockCatalog.SetTrackingPlanWithIdentifiers(tt.remoteTrackingPlan)
			mockCatalog.SetError(tt.mockErr)

			res, err := provider.Import(ctx, "local-tp-id", tt.localArgs.ToResourceData(), "remote-tp-id")

			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			actual := *res
			expected := *tt.expectResource
			actual["updatedAt"] = expected["updatedAt"]
			assert.Equal(t, expected, actual)
			assert.Equal(t, tt.expectUpdate, mockCatalog.updateCalled)
			assert.Equal(t, tt.expectSetExtId, mockCatalog.setExternalIdCalled)
			assert.Equal(t, tt.expectDeleteEventCount, mockCatalog.deleteEventCallCount)
			assert.Equal(t, tt.expectUpdateEventCount, mockCatalog.updateEventCallCount)
		})
	}
}

func getTrackingPlanArgs() *state.TrackingPlanArgs {

	f := defaultTrackingPlanArgsFactory()
	f = f.WithEvent(&state.TrackingPlanEventArgs{
		ID:             "upstream-event-id",
		LocalID:        "event-id",
		AllowUnplanned: false,
		Properties: []*state.TrackingPlanPropertyArgs{
			{
				LocalID:  "property-id",
				Required: true,
			},
		},
	})

	args := f.Build()
	return &args
}

func defaultTrackingPlanFactory() *factory.TrackingPlanCatalogFactory {
	f := factory.NewTrackingPlanCatalogFactory()
	f.WithID("upstream-tracking-plan-id")
	f.WithName("tracking-plan")
	f.WithDescription("tracking-plan-description")
	f.WithWorkspaceID("workspace-id")
	f.WithCreationType("backend")
	f.WithCreatedAt(time.Date(2021, 9, 1, 0, 0, 0, 0, time.UTC))
	f.WithUpdatedAt(time.Date(2021, 9, 2, 0, 0, 0, 0, time.UTC))
	f.WithVersion(1)
	return f
}

func defaultTrackingPlanArgsFactory() *factory.TrackingPlanArgsFactory {
	f := factory.NewTrackingPlanArgsFactory()
	f.WithLocalID("tracking-plan-id")
	f.WithName("tracking-plan")
	f.WithDescription("tracking-plan-description")
	return f
}

func defaultTrackingPlanStateFactory() *factory.TrackingPlanStateFactory {
	f := factory.NewTrackingPlanStateFactory()
	f.WithID("tracking-plan-id")
	f.WithName("tracking-plan-name")
	f.WithDescription("tracking-plan-description")
	f.WithWorkspaceID("workspace-id")
	f.WithCreatedAt("2021-09-01 00:00:00 +0000 UTC")
	f.WithUpdatedAt("2021-09-02T00:00:00 +0000 UTC")
	return f

}

func getTestTrackingPlan() *catalog.TrackingPlan {
	f := defaultTrackingPlanFactory()
	f.WithEvent(catalog.TrackingPlanEvent{
		ID:             "upstream-tracking-plan-event-id",
		TrackingPlanID: "tracking-plan-id",
		SchemaID:       "upstream-schema-id",
		EventID:        "upsream-event-id",
	})

	tp := f.Build()
	return &tp
}
