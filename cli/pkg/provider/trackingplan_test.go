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
	"github.com/rudderlabs/rudder-iac/cli/pkg/provider/testutils/factory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrackingPlanProvider_Create(t *testing.T) {
	t.Parallel()

	var (
		ctx      = context.Background()
		catalog  = &MockTrackingPlanCatalog{}
		provider = provider.NewTrackingPlanProvider(catalog)
	)

	var (
		toArgs = getTrackingPlanArgs()
	)

	catalog.SetTrackingPlan(getTestTrackingPlan())
	newState, err := provider.Create(ctx, "tracking-plan-id", typeTrackingPlan, toArgs.ToResourceData())
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
					"name":            "event",
					"localId":         "event-id",
					"description":     "event-description",
					"type":            "event-type",
					"allowUnplanned":  false,
					"identityApplied": "",
					"properties": []map[string]interface{}{
						{
							"name":        "property",
							"localId":     "property-id",
							"description": "property-description",
							"type":        "string",
							"required":    true,
							"config":      map[string]interface{}(nil),
						},
					},
				},
			},
		},
	}, *newState)

}

func TestTrackingPlanProvider_Update(t *testing.T) {
	t.Parallel()

	var (
		ctx      = context.Background()
		catalog  = &MockTrackingPlanCatalog{}
		provider = provider.NewTrackingPlanProvider(catalog)
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
		WithEvent(client.TrackingPlanEvent{
			ID:             "upstream-tracking-plan-event-id",
			TrackingPlanID: "tracking-plan-id",
			SchemaID:       "upstream-schema-id",
			EventID:        "upstream-event-id",
		}).
		WithVersion(2).
		Build()

	catalog.SetTrackingPlan(&updatedTP)

	toArgs := defaultTrackingPlanArgsFactory().
		WithDescription("tracking-plan-updated-description"). // updated description
		WithEvent(&state.TrackingPlanEventArgs{
			Name:           "event",
			LocalID:        "event-id",
			Type:           "event-type",
			Description:    "event-description",
			AllowUnplanned: false,
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					Name:        "property",
					LocalID:     "property-id",
					Description: "property-description",
					Type:        "string",
					Required:    true,
					Config:      nil,
				},
			},
		}).Build()

	olds := marshalUnmarshal(t, oldsState.ToResourceData())
	newState, err := provider.Update(ctx, "tracking-plan-id", typeTrackingPlan, toArgs.ToResourceData(), olds)
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
					"name":            "event",
					"localId":         "event-id",
					"description":     "event-description",
					"type":            "event-type",
					"allowUnplanned":  false,
					"identityApplied": "",
					"properties": []map[string]interface{}{
						{
							"name":        "property",
							"localId":     "property-id",
							"description": "property-description",
							"type":        "string",
							"required":    true,
							"config":      map[string]interface{}(nil),
						},
					},
				},
			},
		},
	}, *newState)
}

func TestTrackingPlanProvider_UpdateWithUpsertEvent(t *testing.T) {
	t.Parallel()

	var (
		ctx      = context.Background()
		catalog  = &MockTrackingPlanCatalog{}
		provider = provider.NewTrackingPlanProvider(catalog)
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
			Name:           "event-1",
			LocalID:        "event-id-1",
			Type:           "event-type-1",
			Description:    "event-description-1",
			AllowUnplanned: true,
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					Name:        "property-1",
					LocalID:     "property-id-1",
					Description: "property-description-1",
					Type:        "string",
					Required:    false,
					Config:      map[string]interface{}{"enum": []string{"value1", "value2"}},
				},
			},
		}).Build()

	updatedTP := defaultTrackingPlanFactory().
		WithDescription("tracking-plan-updated-description"). // updated description
		WithVersion(2).
		WithEvent(client.TrackingPlanEvent{
			ID:             "upstream-tracking-plan-event-id-1",
			TrackingPlanID: "tracking-plan-id",
			SchemaID:       "upstream-schema-id-1",
			EventID:        "upsream-event-id-1",
		}).Build()

	catalog.SetError(nil)
	catalog.SetTrackingPlan(&updatedTP)

	olds := marshalUnmarshal(t, oldsState.ToResourceData())

	newState, err := provider.Update(ctx, "tracking-plan-id", typeTrackingPlan, toArgs.ToResourceData(), olds)
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
					"name":            "event-1",
					"localId":         "event-id-1",
					"description":     "event-description-1",
					"type":            "event-type-1",
					"allowUnplanned":  true,
					"identityApplied": "",
					"properties": []map[string]interface{}{
						{
							"name":        "property-1",
							"localId":     "property-id-1",
							"description": "property-description-1",
							"type":        "string",
							"required":    false,
							"config":      map[string]interface{}{"enum": []string{"value1", "value2"}},
						},
					},
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
			Name:           "event",
			LocalID:        "event-id",
			Type:           "event-type",
			Description:    "event-description",
			AllowUnplanned: false,
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					Name:        "property",
					LocalID:     "property-id",
					Description: "property-description",
					Type:        "string",
					Required:    true,
					Config:      nil,
				},
			},
		}).
		WithEvent(&state.TrackingPlanEventArgs{
			Name:           "event-1",
			LocalID:        "event-id-1",
			Type:           "event-type-1",
			Description:    "event-description-1",
			AllowUnplanned: true,
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					Name:        "property",
					LocalID:     "property-id",
					Description: "property-description",
					Type:        "string",
					Required:    true,
					Config:      nil,
				},
			},
		}).
		Build()

	newArgs := defaultTrackingPlanArgsFactory().
		WithEvent(&state.TrackingPlanEventArgs{
			Name:           "event",
			LocalID:        "event-id",
			Type:           "event-type",
			Description:    "event-description",
			AllowUnplanned: true, // updated
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					Name:        "property",
					LocalID:     "property-id",
					Description: "property-description",
					Type:        "string",
					Required:    true,
					Config:      map[string]interface{}{"enum": []string{"value1", "value2"}}, // updated
				},
				{
					Name:        "property-1",
					LocalID:     "property-id-1",
					Description: "property-description-1",
					Type:        "int",
					Required:    false,
					Config:      map[string]interface{}{"enum": []int{1, 2, 3}},
				},
			},
		}).
		WithEvent(&state.TrackingPlanEventArgs{
			Name:           "event-2",
			LocalID:        "event-id-2",
			Type:           "event-type-2",
			Description:    "event-description-2",
			AllowUnplanned: true,
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					Name:        "property-2",
					LocalID:     "property-id-2",
					Description: "property-description-2",
					Type:        "string",
					Required:    false,
					Config:      nil,
				},
			},
		}).Build()

	diffed := oldArgs.Diff(newArgs)
	require.Equal(t, 1, len(diffed.Added))
	require.Equal(t, 1, len(diffed.Updated))
	require.Equal(t, 1, len(diffed.Deleted))

	assert.Equal(t, &state.TrackingPlanEventArgs{
		Name:           "event-2",
		LocalID:        "event-id-2",
		Type:           "event-type-2",
		Description:    "event-description-2",
		AllowUnplanned: true,
		Properties: []*state.TrackingPlanPropertyArgs{
			{
				Name:        "property-2",
				LocalID:     "property-id-2",
				Description: "property-description-2",
				Type:        "string",
				Required:    false,
				Config:      nil,
			},
		},
	}, diffed.Added[0])

	assert.Equal(t, &state.TrackingPlanEventArgs{
		Name:           "event-1",
		LocalID:        "event-id-1",
		Type:           "event-type-1",
		Description:    "event-description-1",
		AllowUnplanned: true,
		Properties: []*state.TrackingPlanPropertyArgs{
			{
				Name:        "property",
				LocalID:     "property-id",
				Description: "property-description",
				Type:        "string",
				Required:    true,
				Config:      nil,
			},
		},
	}, diffed.Deleted[0])

	assert.Equal(t, &state.TrackingPlanEventArgs{
		Name:           "event",
		LocalID:        "event-id",
		Type:           "event-type",
		Description:    "event-description",
		AllowUnplanned: true,
		Properties: []*state.TrackingPlanPropertyArgs{
			{
				Name:        "property",
				LocalID:     "property-id",
				Description: "property-description",
				Type:        "string",
				Required:    true,
				Config:      map[string]interface{}{"enum": []string{"value1", "value2"}},
			},
			{
				Name:        "property-1",
				LocalID:     "property-id-1",
				Description: "property-description-1",
				Type:        "int",
				Required:    false,
				Config:      map[string]interface{}{"enum": []int{1, 2, 3}},
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
		provider = provider.NewTrackingPlanProvider(&MockTrackingPlanCatalog{})
	)

	err := provider.Delete(ctx, "tracking-plan-id", typeTrackingPlan, resources.ResourceData{"id": "upstream-tracking-plan-id"})
	require.Nil(t, err)
}

type MockTrackingPlanCatalog struct {
	EmptyCatalog
	tp  *client.TrackingPlan
	err error
}

func (m *MockTrackingPlanCatalog) CreateTrackingPlan(ctx context.Context, trackingPlanCreate client.TrackingPlanCreate) (*client.TrackingPlan, error) {
	return m.tp, m.err
}

func (m *MockTrackingPlanCatalog) UpdateTrackingPlan(ctx context.Context, trackingPlanID string, name string, description string) (*client.TrackingPlan, error) {
	return m.tp, m.err
}

func (m *MockTrackingPlanCatalog) DeleteTrackingPlan(ctx context.Context, trackingPlanID string) error {
	return m.err
}

func (m *MockTrackingPlanCatalog) DeleteTrackingPlanEvent(ctx context.Context, trackingPlanID string, eventID string) error {
	return m.err
}

func (m *MockTrackingPlanCatalog) UpsertTrackingPlan(ctx context.Context, trackingPlanID string, trackingPlanUpsertEvent client.TrackingPlanUpsertEvent) (*client.TrackingPlan, error) {
	return m.tp, m.err
}

func (m *MockTrackingPlanCatalog) SetTrackingPlan(tp *client.TrackingPlan) {
	m.tp = tp
}

func (m *MockTrackingPlanCatalog) SetError(err error) {
	m.err = err
}

func getTrackingPlanArgs() *state.TrackingPlanArgs {

	f := defaultTrackingPlanArgsFactory()
	f = f.WithEvent(&state.TrackingPlanEventArgs{
		Name:           "event",
		LocalID:        "event-id",
		Type:           "event-type",
		Description:    "event-description",
		AllowUnplanned: false,
		Properties: []*state.TrackingPlanPropertyArgs{
			{
				Name:        "property",
				LocalID:     "property-id",
				Description: "property-description",
				Type:        "string",
				Required:    true,
				Config:      nil,
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

func getTestTrackingPlan() *client.TrackingPlan {
	f := defaultTrackingPlanFactory()
	f.WithEvent(client.TrackingPlanEvent{
		ID:             "upstream-tracking-plan-event-id",
		TrackingPlanID: "tracking-plan-id",
		SchemaID:       "upstream-schema-id",
		EventID:        "upsream-event-id",
	})

	tp := f.Build()
	return &tp
}
