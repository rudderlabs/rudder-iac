package datacatalog_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/testutils/factory"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ catalog.DataCatalog = &MockTrackingPlanCatalog{}

type MockTrackingPlanCatalog struct {
	datacatalog.EmptyCatalog
	tp           *catalog.TrackingPlan
	tpWithSchema *catalog.TrackingPlanWithSchemas
	tpes         *catalog.TrackingPlanEventSchema
	err          error
}

func (m *MockTrackingPlanCatalog) CreateTrackingPlan(ctx context.Context, trackingPlanCreate catalog.TrackingPlanCreate) (*catalog.TrackingPlan, error) {
	return m.tp, m.err
}

func (m *MockTrackingPlanCatalog) UpdateTrackingPlan(ctx context.Context, trackingPlanID string, name string, description string) (*catalog.TrackingPlan, error) {
	return m.tp, m.err
}

func (m *MockTrackingPlanCatalog) DeleteTrackingPlan(ctx context.Context, trackingPlanID string) error {
	return m.err
}

func (m *MockTrackingPlanCatalog) DeleteTrackingPlanEvent(ctx context.Context, trackingPlanID string, eventID string) error {
	return m.err
}

func (m *MockTrackingPlanCatalog) UpsertTrackingPlan(ctx context.Context, trackingPlanID string, trackingPlanUpsertEvent catalog.TrackingPlanUpsertEvent) (*catalog.TrackingPlan, error) {
	return m.tp, m.err
}

func (m *MockTrackingPlanCatalog) GetTrackingPlan(ctx context.Context, id string) (*catalog.TrackingPlanWithSchemas, error) {
	return m.tpWithSchema, m.err
}

func (m *MockTrackingPlanCatalog) GetTrackingPlanEventSchema(ctx context.Context, id string, eventId string) (*catalog.TrackingPlanEventSchema, error) {
	return m.tpes, m.err
}

func (m *MockTrackingPlanCatalog) UpdateTrackingPlanEvent(ctx context.Context, id string, input catalog.EventIdentifierDetail) (*catalog.TrackingPlan, error) {
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

func TestTrackingPlanProvider_Create(t *testing.T) {
	t.Parallel()

	var (
		ctx         = context.Background()
		mockCatalog = &MockTrackingPlanCatalog{}
		provider    = datacatalog.NewTrackingPlanProvider(mockCatalog)
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
				"id":      "upstream-tracking-plan-event-id",
				"eventId": "upsream-event-id",
				"localId": "event-id",
			},
		},
		"trackingPlanArgs": map[string]interface{}{
			"name":        "tracking-plan",
			"localId":     "tracking-plan-id",
			"description": "tracking-plan-description",
			"events": []map[string]interface{}{
				{
					"id":              "",
					"localId":         "event-id",
					"allowUnplanned":  false,
					"identitySection": "",
					"properties": []map[string]interface{}{
						{
							"localId":  "property-id",
							"id":       "",
							"required": true,
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
		provider    = datacatalog.NewTrackingPlanProvider(mockCatalog)
	)

	var (
		oldsArgs = getTrackingPlanArgs()
	)

	oldsState := defaultTrackingPlanStateFactory().
		WithEvent(&state.TrackingPlanEventState{
			ID:      "upstream-tracking-plan-event-id",
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
				"id":      "upstream-tracking-plan-event-id",
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
							"id":       "",
							"localId":  "property-id",
							"required": true,
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
		provider    = datacatalog.NewTrackingPlanProvider(mockCatalog)
	)

	var (
		oldsArgs = getTrackingPlanArgs()
	)

	oldsState := defaultTrackingPlanStateFactory().
		WithEvent(&state.TrackingPlanEventState{
			ID:      "upstream-tracking-plan-event-id",
			LocalID: "event-id",
			EventID: "upstream-event-id",
		}).
		WithTrackingPlanArgs(*oldsArgs).
		Build()

	toArgs := defaultTrackingPlanArgsFactory().
		WithDescription("tracking-plan-updated-description"). // updated description
		WithEvent(&state.TrackingPlanEventArgs{               // updated events under the trackingplan +1 Added -1 Removed
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
				"id":      "upstream-tracking-plan-event-id-1",
				"eventId": "upsream-event-id-1",
				"localId": "event-id-1",
			},
		},
		"trackingPlanArgs": map[string]interface{}{
			"name":        "tracking-plan",
			"localId":     "tracking-plan-id",
			"description": "tracking-plan-updated-description", // updated description
			"events": []map[string]interface{}{
				{
					"id":              "",
					"localId":         "event-id-1",
					"allowUnplanned":  true,
					"identitySection": "",
					"properties": []map[string]interface{}{
						{
							"id":       "",
							"localId":  "property-id-1",
							"required": false,
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
		provider = datacatalog.NewTrackingPlanProvider(&MockTrackingPlanCatalog{})
	)

	err := provider.Delete(ctx, "tracking-plan-id", resources.ResourceData{"id": "upstream-tracking-plan-id"})
	require.Nil(t, err)
}

func getTrackingPlanArgs() *state.TrackingPlanArgs {

	f := defaultTrackingPlanArgsFactory()
	f = f.WithEvent(&state.TrackingPlanEventArgs{
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
