package datacatalog_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/testutils/factory"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	rstate "github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
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
	createCalled         bool
	getIdentifiersCalled bool
	setExternalIdCalled  bool
	deleteEventCalled    bool
	updateEventCalled    bool
	deleteEventCallCount int
	updateEventCallCount int
}

func (m *MockTrackingPlanCatalog) CreateTrackingPlan(ctx context.Context, trackingPlanCreate catalog.TrackingPlanCreate) (*catalog.TrackingPlan, error) {
	m.createCalled = true
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

func (m *MockTrackingPlanCatalog) GetTrackingPlan(ctx context.Context, id string) (*catalog.TrackingPlan, error) {
	return m.tp, m.err
}

func (m *MockTrackingPlanCatalog) DeleteTrackingPlanEvent(ctx context.Context, trackingPlanID string, eventID string) error {
	m.deleteEventCalled = true
	m.deleteEventCallCount++
	return m.err
}

func (m *MockTrackingPlanCatalog) UpsertTrackingPlan(ctx context.Context, trackingPlanID string, trackingPlanUpsertEvent catalog.TrackingPlanUpsertEvent) (*catalog.TrackingPlan, error) {
	return m.tp, m.err
}

func (m *MockTrackingPlanCatalog) GetTrackingPlanWithIdentifiers(ctx context.Context, id string, rebuildSchemas bool) (*catalog.TrackingPlanWithIdentifiers, error) {
	m.getIdentifiersCalled = true
	return m.tpWithIdentifiers, m.err
}

func (m *MockTrackingPlanCatalog) GetTrackingPlanWithSchemas(ctx context.Context, id string) (*catalog.TrackingPlanWithSchemas, error) {
	return m.tpWithSchema, m.err
}

func (m *MockTrackingPlanCatalog) GetTrackingPlanEventSchema(ctx context.Context, id string, eventId string) (*catalog.TrackingPlanEventSchema, error) {
	return m.tpes, m.err
}

func (m *MockTrackingPlanCatalog) UpdateTrackingPlanEvents(ctx context.Context, id string, input []catalog.EventIdentifierDetail, rebuildSchemas bool) error {
	for range input {
		m.updateEventCalled = true
		m.updateEventCallCount++
	}
	return m.err
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
	m.createCalled = false
	m.getIdentifiersCalled = false
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
							"id":                   "upstream-property-id",
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
		WithTrackingPlanArgs(*oldsArgs).
		Build()

	// // the default tracking plan with version
	updatedTP := defaultTrackingPlanFactory().
		WithDescription("tracking-plan-updated-description"). // updated description
		WithVersion(2).
		Build()

	mockCatalog.SetTrackingPlan(&updatedTP)

	toArgs := defaultTrackingPlanArgsFactory().
		WithDescription("tracking-plan-updated-description"). // updated description
		WithEvent(&state.TrackingPlanEventArgs{
			ID:             "upstream-event-id",
			LocalID:        "event-id",
			AllowUnplanned: false,
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					ID:       "upstream-property-id",
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
		"trackingPlanArgs": map[string]interface{}{
			"name":        "tracking-plan",
			"localId":     "tracking-plan-id",
			"description": "tracking-plan-updated-description", // updated description
			"events": []map[string]interface{}{
				{
					"localId":         "event-id",
					"id":              "upstream-event-id",
					"allowUnplanned":  false,
					"identitySection": "",
					"properties": []map[string]interface{}{
						{
							"id":                   "upstream-property-id",
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
					ID:       "upstream-property-id-1",
					LocalID:  "property-id-1",
					Required: false,
				},
			},
		}).Build()

	updatedTP := defaultTrackingPlanFactory().
		WithDescription("tracking-plan-updated-description"). // updated description
		WithVersion(2).
		Build()

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
							"id":                   "upstream-property-id-1",
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
			ID:             "event-id",
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
			ID:             "event-id-1",
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
			ID:             "event-id",
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
			ID:             "event-id-2",
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
		ID:             "event-id-2",
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
		ID:             "event-id-1",
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
		ID:             "event-id",
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
				TrackingPlan: catalog.TrackingPlan{ID: "remote-tp-id", Name: "TestTP", Description: strptr("desc"), Version: 1, CreationType: "backend", WorkspaceID: "ws-id", CreatedAt: created, UpdatedAt: updated},
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
				"creationType": "backend",
				"workspaceId":  "ws-id",
				"createdAt":    created.String(),
				"updatedAt":    updated.String(),
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
				TrackingPlan: catalog.TrackingPlan{ID: "remote-tp-id", Name: "TestTP", Description: strptr("desc"), Version: 1, CreationType: "backend", WorkspaceID: "ws-id", CreatedAt: created, UpdatedAt: updated},
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
				"creationType": "backend",
				"workspaceId":  "ws-id",
				"createdAt":    created.String(),
				"updatedAt":    updated.String(),
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
				TrackingPlan: catalog.TrackingPlan{ID: "remote-tp-id", Name: "TestTP", Description: strptr("desc"), Version: 1, CreationType: "backend", WorkspaceID: "ws-id", CreatedAt: created, UpdatedAt: updated},
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

func trackingPlanProviderWithImport(t *testing.T, mock *MockTrackingPlanCatalog, name string) *datacatalog.Provider {
	t.Helper()
	p := datacatalog.New(mock)
	p.SetWorkspaceID("ws-a")
	s := &specs.Spec{Kind: "tracking-plan", Version: specs.SpecVersionV1,
		Spec: map[string]any{"id": "local-tp", "display_name": name},
		Metadata: map[string]any{"import": map[string]any{"workspaces": []any{map[string]any{
			"workspace_id": "ws-a", "resources": []any{map[string]any{"urn": "tracking-plan:local-tp", "remote_id": "remote-tp"}},
		}}}},
	}
	require.NoError(t, p.LoadSpec("tp.yaml", s))
	_, err := p.ResourceGraph()
	require.NoError(t, err)
	return p
}

func TestTrackingPlanLegacyNameGuards(t *testing.T) {
	const legacy = "Sales & Marketing"
	ctx := context.Background()
	args := state.TrackingPlanArgs{LocalID: "local-tp", Name: legacy}
	remote := &catalog.TrackingPlanWithIdentifiers{TrackingPlan: catalog.TrackingPlan{
		ID: "remote-tp", Name: legacy, WorkspaceID: "ws-a",
	}}

	t.Run("create rejects invalid names without writing", func(t *testing.T) {
		for _, name := range []string{legacy, "Plan@Name", "Ab", "Name "} {
			mock := &MockTrackingPlanCatalog{}
			in := args
			in.Name = name
			_, err := datacatalog.NewTrackingPlanProvider(mock, "data-catalog").Create(ctx, "local-tp", in.ToResourceData())
			require.Error(t, err, name)
			assert.False(t, mock.createCalled)
		}
	})

	t.Run("import changes events without writing a legacy plan name", func(t *testing.T) {
		mock := &MockTrackingPlanCatalog{tpWithIdentifiers: remote}
		in := args
		in.Events = []*state.TrackingPlanEventArgs{{ID: "event-1", LocalID: "event-1"}}
		p := trackingPlanProviderWithImport(t, mock, legacy)
		_, err := p.Import(ctx, "local-tp", "tracking-plan", in.ToResourceData(), "remote-tp")
		require.NoError(t, err)
		assert.False(t, mock.updateCalled)
		assert.Equal(t, 1, mock.updateEventCallCount)
		assert.True(t, mock.setExternalIdCalled)
	})

	t.Run("import rejects legacy description change before event and external ID writes", func(t *testing.T) {
		mock := &MockTrackingPlanCatalog{tpWithIdentifiers: remote}
		in := args
		in.Description = "new description"
		in.Events = []*state.TrackingPlanEventArgs{{ID: "event-1", LocalID: "event-1"}}
		p := trackingPlanProviderWithImport(t, mock, legacy)
		_, err := p.Import(ctx, "local-tp", "tracking-plan", in.ToResourceData(), "remote-tp")
		require.ErrorContains(t, err, "cannot update a legacy")
		assert.False(t, mock.updateCalled)
		assert.False(t, mock.updateEventCalled)
		assert.False(t, mock.setExternalIdCalled)
	})

	t.Run("import rejects incorrect remote workspace before writes", func(t *testing.T) {
		mock := &MockTrackingPlanCatalog{tpWithIdentifiers: &catalog.TrackingPlanWithIdentifiers{TrackingPlan: catalog.TrackingPlan{
			ID: "remote-tp", Name: legacy, WorkspaceID: "ws-b",
		}}}
		p := trackingPlanProviderWithImport(t, mock, legacy)
		_, err := p.Import(ctx, "local-tp", "tracking-plan", args.ToResourceData(), "remote-tp")
		require.ErrorContains(t, err, "unverified legacy")
		assert.False(t, mock.updateCalled)
		assert.False(t, mock.updateEventCalled)
		assert.False(t, mock.setExternalIdCalled)
	})

	t.Run("import rejects a mismatched name", func(t *testing.T) {
		mock := &MockTrackingPlanCatalog{tpWithIdentifiers: remote}
		in := args
		in.Name = "Other & Plan"
		p := trackingPlanProviderWithImport(t, mock, legacy)
		_, err := p.Import(ctx, "local-tp", "tracking-plan", in.ToResourceData(), "remote-tp")
		require.Error(t, err)
		assert.False(t, mock.updateCalled)
		assert.False(t, mock.setExternalIdCalled)
	})

	t.Run("direct update of verified legacy name changes events only", func(t *testing.T) {
		mock := &MockTrackingPlanCatalog{tpWithIdentifiers: &catalog.TrackingPlanWithIdentifiers{TrackingPlan: catalog.TrackingPlan{
			ID: "remote-tp", Name: legacy, WorkspaceID: "ws-a", ExternalID: "local-tp",
		}}, tp: &catalog.TrackingPlan{ID: "remote-tp", Name: legacy, WorkspaceID: "ws-a"}}
		prev := state.TrackingPlanState{TrackingPlanArgs: args, ID: "remote-tp", Name: legacy, WorkspaceID: "ws-a"}
		in := args
		in.Events = []*state.TrackingPlanEventArgs{{ID: "event-1", LocalID: "event-1"}}
		p := trackingPlanProviderWithImport(t, mock, legacy)
		_, err := p.Update(ctx, "local-tp", "tracking-plan", in.ToResourceData(), prev.ToResourceData())
		require.NoError(t, err)
		assert.False(t, mock.updateCalled)
		assert.Equal(t, 1, mock.updateEventCallCount)
	})

	t.Run("direct update refuses description or name changes", func(t *testing.T) {
		for _, changed := range []state.TrackingPlanArgs{
			{LocalID: "local-tp", Name: legacy, Description: "changed"},
			{LocalID: "local-tp", Name: "Other & Plan"},
		} {
			mock := &MockTrackingPlanCatalog{tpWithIdentifiers: &catalog.TrackingPlanWithIdentifiers{TrackingPlan: catalog.TrackingPlan{
				ID: "remote-tp", Name: legacy, WorkspaceID: "ws-a", ExternalID: "local-tp",
			}}}
			prev := state.TrackingPlanState{TrackingPlanArgs: args, ID: "remote-tp", Name: legacy, WorkspaceID: "ws-a"}
			p := trackingPlanProviderWithImport(t, mock, legacy)
			_, err := p.Update(ctx, "local-tp", "tracking-plan", changed.ToResourceData(), prev.ToResourceData())
			require.Error(t, err)
			assert.False(t, mock.updateCalled)
			assert.False(t, mock.updateEventCalled)
		}
	})
}

func TestTrackingPlanCentralManifestWorkspaceScope(t *testing.T) {
	for _, tc := range []struct {
		kind, version string
	}{
		{"tp", specs.SpecVersionV0_1}, {"tp", specs.SpecVersionV0_1Variant}, {"tracking-plan", specs.SpecVersionV1},
	} {
		for _, manifestWorkspace := range []string{"ws-a", "ws-b"} {
			t.Run(tc.version+"/manifest-"+manifestWorkspace, func(t *testing.T) {
				mock := &MockTrackingPlanCatalog{tpWithIdentifiers: &catalog.TrackingPlanWithIdentifiers{TrackingPlan: catalog.TrackingPlan{
					ID: "remote-tp", Name: "Sales & Marketing", WorkspaceID: "ws-a",
				}}}
				p := datacatalog.New(mock)
				p.SetWorkspaceID("ws-a")
				s := &specs.Spec{Version: tc.version, Kind: tc.kind,
					Spec: map[string]any{"id": "local-tp", "display_name": "Sales & Marketing"}}
				var err error
				if tc.kind == "tp" {
					err = p.LoadLegacySpec("tp.yaml", s)
				} else {
					err = p.LoadSpec("tp.yaml", s)
				}
				require.NoError(t, err)
				require.NoError(t, p.LoadImportManifest(&specs.WorkspaceImportMetadata{
					WorkspaceID: manifestWorkspace, Resources: []specs.ImportIds{{URN: "tracking-plan:local-tp", RemoteID: "remote-tp"}},
				}))
				graph, err := p.ResourceGraph()
				require.NoError(t, err)
				resource, ok := graph.GetResource("tracking-plan:local-tp")
				require.True(t, ok)
				if manifestWorkspace == "ws-b" {
					assert.Nil(t, resource.ImportMetadata())
				} else {
					require.NotNil(t, resource.ImportMetadata())
					assert.Equal(t, "remote-tp", resource.ImportMetadata().RemoteId)
				}
				err = p.CheckPlan(context.Background(), &planner.Plan{Operations: []*planner.Operation{{Type: planner.Import, Resource: resource}}}, rstate.EmptyState())
				if manifestWorkspace == "ws-b" {
					require.ErrorContains(t, err, "requires a declared import")
					assert.False(t, mock.getIdentifiersCalled)
				} else {
					require.NoError(t, err)
					assert.True(t, mock.getIdentifiersCalled)
				}
			})
		}
	}
}

func TestTrackingPlanImportGraphWorkspaceScope(t *testing.T) {
	for _, tc := range []struct {
		kind, version string
	}{
		{"tp", specs.SpecVersionV0_1}, {"tp", specs.SpecVersionV0_1Variant}, {"tracking-plan", specs.SpecVersionV1},
	} {
		t.Run(tc.version, func(t *testing.T) {
			p := datacatalog.New(&MockTrackingPlanCatalog{})
			p.SetWorkspaceID("ws-a")
			s := &specs.Spec{Version: tc.version, Kind: tc.kind, Metadata: map[string]any{"import": map[string]any{"workspaces": []any{
				map[string]any{"workspace_id": "ws-a", "resources": []any{map[string]any{"urn": "tracking-plan:tp-1", "remote_id": "remote-a"}}},
				map[string]any{"workspace_id": "ws-b", "resources": []any{map[string]any{"urn": "tracking-plan:tp-1", "remote_id": "remote-b"}}},
			}}}, Spec: map[string]any{"id": "tp-1", "display_name": "Sales & Marketing"}}
			var err error
			if tc.kind == "tp" {
				err = p.LoadLegacySpec("tp.yaml", s)
			} else {
				err = p.LoadSpec("tp.yaml", s)
			}
			require.NoError(t, err)
			graph, err := p.ResourceGraph()
			require.NoError(t, err)
			resource, ok := graph.GetResource("tracking-plan:tp-1")
			require.True(t, ok)
			require.NotNil(t, resource.ImportMetadata())
			assert.Equal(t, "remote-a", resource.ImportMetadata().RemoteId)
		})
	}
}

func TestTrackingPlanUnresolvedWorkspaceCannotGrandfather(t *testing.T) {
	mock := &MockTrackingPlanCatalog{tpWithIdentifiers: &catalog.TrackingPlanWithIdentifiers{TrackingPlan: catalog.TrackingPlan{
		ID: "remote-tp", Name: "Sales & Marketing", WorkspaceID: "ws-a",
	}}}
	p := datacatalog.New(mock)
	s := &specs.Spec{Kind: "tracking-plan", Version: specs.SpecVersionV1,
		Spec: map[string]any{"id": "local-tp", "display_name": "Sales & Marketing"},
		Metadata: map[string]any{"import": map[string]any{"workspaces": []any{map[string]any{
			"workspace_id": "ws-a", "resources": []any{map[string]any{"urn": "tracking-plan:local-tp", "remote_id": "remote-tp"}},
		}}}},
	}
	require.NoError(t, p.LoadSpec("tp.yaml", s))
	graph, err := p.ResourceGraph()
	require.NoError(t, err)
	resource, ok := graph.GetResource("tracking-plan:local-tp")
	require.True(t, ok)
	err = p.CheckPlan(context.Background(), &planner.Plan{Operations: []*planner.Operation{{Type: planner.Import, Resource: resource}}}, rstate.EmptyState())
	require.ErrorContains(t, err, "requires a declared import")
	assert.False(t, mock.getIdentifiersCalled)
	args := state.TrackingPlanArgs{LocalID: "local-tp", Name: "Sales & Marketing"}
	_, err = p.Import(context.Background(), "local-tp", "tracking-plan", args.ToResourceData(), "remote-tp")
	require.ErrorContains(t, err, "unverified legacy")
	assert.False(t, mock.setExternalIdCalled)
}

func TestTrackingPlanCheckPlanLegacyIdentity(t *testing.T) {
	const legacy = "Sales & Marketing"
	for _, tc := range []struct {
		name, workspace, remoteName, remoteWorkspace, remoteID, externalID string
		operation                                                          planner.OperationType
		wantErr                                                            bool
		wantFetch                                                          bool
	}{
		{name: "matching import", workspace: "ws-a", remoteName: legacy, remoteWorkspace: "ws-a", remoteID: "remote-tp", operation: planner.Import, wantFetch: true},
		{name: "workspace mismatch", workspace: "ws-b", remoteName: legacy, remoteWorkspace: "ws-a", remoteID: "remote-tp", operation: planner.Import, wantErr: true},
		{name: "remote name mismatch", workspace: "ws-a", remoteName: "Other & Plan", remoteWorkspace: "ws-a", remoteID: "remote-tp", operation: planner.Import, wantErr: true, wantFetch: true},
		{name: "remote workspace mismatch", workspace: "ws-a", remoteName: legacy, remoteWorkspace: "ws-b", remoteID: "remote-tp", operation: planner.Import, wantErr: true, wantFetch: true},
		{name: "remote identity mismatch", workspace: "ws-a", remoteName: legacy, remoteWorkspace: "ws-a", remoteID: "other", operation: planner.Import, wantErr: true, wantFetch: true},
		{name: "external ID mismatch", workspace: "ws-a", remoteName: legacy, remoteWorkspace: "ws-a", remoteID: "remote-tp", externalID: "other", operation: planner.Import, wantErr: true, wantFetch: true},
		{name: "create must not grandfather", workspace: "ws-a", remoteName: legacy, remoteWorkspace: "ws-a", remoteID: "remote-tp", operation: planner.Create, wantErr: true},
		{name: "update without prior state must not grandfather", workspace: "ws-a", remoteName: legacy, remoteWorkspace: "ws-a", remoteID: "remote-tp", operation: planner.Update, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mock := &MockTrackingPlanCatalog{tpWithIdentifiers: &catalog.TrackingPlanWithIdentifiers{TrackingPlan: catalog.TrackingPlan{
				ID: tc.remoteID, Name: tc.remoteName, WorkspaceID: tc.remoteWorkspace, ExternalID: tc.externalID,
			}}}
			p := trackingPlanProviderWithImport(t, mock, legacy)
			resource := resources.NewResource("local-tp", "tracking-plan", resources.ResourceData{"name": legacy}, nil,
				resources.WithResourceImportMetadata("remote-tp", tc.workspace))
			err := p.CheckPlan(context.Background(), &planner.Plan{Operations: []*planner.Operation{{Type: tc.operation, Resource: resource}}}, rstate.EmptyState())
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.wantFetch, mock.getIdentifiersCalled)
		})
	}
}

func TestTrackingPlanCheckPlanManagedLegacyUpdate(t *testing.T) {
	const legacy = "Sales & Marketing"
	const description = "original description"

	for _, tc := range []struct {
		name, nextName, nextDescription, stateName, stateDescription, argsName, argsDescription string
		remoteID, remoteName, remoteWorkspace, externalID, declaredWorkspace                    string
		wantErr, wantFetch, wantEventWrite                                                      bool
	}{
		{name: "event-only update", wantFetch: true, wantEventWrite: true},
		{name: "description change", nextDescription: "changed description", wantErr: true},
		{name: "rename", nextName: "Other & Plan", wantErr: true},
		{name: "state name mismatch", stateName: "Other & Plan", wantErr: true},
		{name: "state args name mismatch", argsName: "Other & Plan", wantErr: true},
		{name: "state description mismatch", stateDescription: "other description", wantErr: true},
		{name: "state args description mismatch", argsDescription: "other description", wantErr: true},
		{name: "remote id mismatch", remoteID: "other", wantErr: true, wantFetch: true},
		{name: "remote name mismatch", remoteName: "Other & Plan", wantErr: true, wantFetch: true},
		{name: "remote workspace mismatch", remoteWorkspace: "ws-b", wantErr: true, wantFetch: true},
		{name: "remote external ID mismatch", externalID: "other", wantErr: true, wantFetch: true},
		{name: "declared workspace mismatch", declaredWorkspace: "ws-b", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defaultString := func(value, fallback string) string {
				if value != "" {
					return value
				}
				return fallback
			}
			mock := &MockTrackingPlanCatalog{tpWithIdentifiers: &catalog.TrackingPlanWithIdentifiers{TrackingPlan: catalog.TrackingPlan{
				ID: defaultString(tc.remoteID, "remote-tp"), Name: defaultString(tc.remoteName, legacy),
				WorkspaceID: defaultString(tc.remoteWorkspace, "ws-a"), ExternalID: defaultString(tc.externalID, "local-tp"),
			}}, tp: &catalog.TrackingPlan{ID: "remote-tp", Name: legacy, WorkspaceID: "ws-a"}}
			p := trackingPlanProviderWithImport(t, mock, legacy)
			args := state.TrackingPlanArgs{LocalID: "local-tp", Name: legacy, Description: description}
			prevArgs := args
			prevArgs.Name = defaultString(tc.argsName, legacy)
			prevArgs.Description = defaultString(tc.argsDescription, description)
			prev := state.TrackingPlanState{
				TrackingPlanArgs: prevArgs, ID: "remote-tp", Name: defaultString(tc.stateName, legacy),
				Description: defaultString(tc.stateDescription, description), WorkspaceID: "ws-a",
			}
			st := rstate.EmptyState()
			st.AddResource(&rstate.ResourceState{ID: "local-tp", Type: "tracking-plan", Input: args.ToResourceData(), Output: marshalUnmarshal(t, prev.ToResourceData())})
			args.Name = defaultString(tc.nextName, legacy)
			args.Description = defaultString(tc.nextDescription, description)
			args.Events = []*state.TrackingPlanEventArgs{{ID: "event-1", LocalID: "event-1"}}
			workspace := defaultString(tc.declaredWorkspace, "ws-a")
			resource := resources.NewResource("local-tp", "tracking-plan", args.ToResourceData(), nil,
				resources.WithResourceImportMetadata("remote-tp", workspace))
			err := p.CheckPlan(context.Background(), &planner.Plan{Operations: []*planner.Operation{{Type: planner.Update, Resource: resource}}}, st)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				_, err = p.Update(context.Background(), "local-tp", "tracking-plan", args.ToResourceData(), st.GetResource(resource.URN()).Data())
				require.NoError(t, err)
			}
			assert.Equal(t, tc.wantFetch, mock.getIdentifiersCalled)
			assert.False(t, mock.updateCalled)
			assert.Equal(t, tc.wantEventWrite, mock.updateEventCalled)
			if tc.wantErr && tc.declaredWorkspace == "" {
				_, err = p.Update(context.Background(), "local-tp", "tracking-plan", args.ToResourceData(), st.GetResource(resource.URN()).Data())
				require.Error(t, err)
				assert.False(t, mock.updateCalled)
				assert.False(t, mock.updateEventCalled)
			}
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
				ID:       "upstream-property-id",
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
	tp := f.Build()
	return &tp
}
