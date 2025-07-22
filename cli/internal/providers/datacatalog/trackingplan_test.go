package datacatalog_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/testutils/factory"
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
					"name":            "event",
					"localId":         "event-id",
					"categoryId":      "",
					"description":     "event-description",
					"type":            "event-type",
					"allowUnplanned":  false,
					"identitySection": "",
					"properties": []map[string]interface{}{
						{
							"name":             "property",
							"localId":          "property-id",
							"description":      "property-description",
							"type":             "string",
							"required":         true,
							"config":           map[string]interface{}(nil),
							"hasCustomTypeRef": false,
							"hasItemTypesRef":  false,
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
					"name":            "event",
					"localId":         "event-id",
					"categoryId":      "",
					"description":     "event-description",
					"type":            "event-type",
					"allowUnplanned":  false,
					"identitySection": "",
					"properties": []map[string]interface{}{
						{
							"name":             "property",
							"localId":          "property-id",
							"description":      "property-description",
							"type":             "string",
							"required":         true,
							"config":           map[string]interface{}(nil),
							"hasCustomTypeRef": false,
							"hasItemTypesRef":  false,
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
					"name":            "event-1",
					"localId":         "event-id-1",
					"categoryId":      "",
					"description":     "event-description-1",
					"type":            "event-type-1",
					"allowUnplanned":  true,
					"identitySection": "",
					"properties": []map[string]interface{}{
						{
							"name":             "property-1",
							"localId":          "property-id-1",
							"description":      "property-description-1",
							"type":             "string",
							"required":         false,
							"config":           map[string]interface{}{"enum": []string{"value1", "value2"}},
							"hasCustomTypeRef": false,
							"hasItemTypesRef":  false,
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
		provider = datacatalog.NewTrackingPlanProvider(&MockTrackingPlanCatalog{})
	)

	err := provider.Delete(ctx, "tracking-plan-id", resources.ResourceData{"id": "upstream-tracking-plan-id"})
	require.Nil(t, err)
}

func getTrackingPlanArgs() *state.TrackingPlanArgs {

	f := defaultTrackingPlanArgsFactory()
	f = f.WithEvent(&state.TrackingPlanEventArgs{
		Name:           "event",
		LocalID:        "event-id",
		Type:           "event-type",
		Description:    "event-description",
		AllowUnplanned: false,
		CategoryId:     nil,
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

func TestGetUpsertEventWithCustomTypeRefs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		property *state.TrackingPlanPropertyArgs
		expected catalog.TrackingPlanUpsertEvent
	}{
		{
			name: "Regular string type",
			property: &state.TrackingPlanPropertyArgs{
				Name:             "prop1",
				LocalID:          "prop-id-1",
				Description:      "Property description",
				Type:             "string",
				Required:         true,
				Config:           map[string]interface{}{"enum": []string{"value1", "value2"}},
				HasCustomTypeRef: false,
				HasItemTypesRef:  false,
			},
			expected: catalog.TrackingPlanUpsertEvent{
				Name:            "event1",
				Description:     "Event description",
				EventType:       "track",
				IdentitySection: "properties",
				Rules: catalog.TrackingPlanUpsertEventRules{
					Type: "object",
					Properties: struct {
						Properties *catalog.TrackingPlanUpsertEventProperties            `json:"properties,omitempty"`
						Traits     *catalog.TrackingPlanUpsertEventProperties            `json:"traits,omitempty"`
						Context    *catalog.TrackingPlanUpsertEventContextTraitsIdentity `json:"context,omitempty"`
					}{
						Properties: &catalog.TrackingPlanUpsertEventProperties{
							Type:                 "object",
							AdditionalProperties: false,
							Required:             []string{"prop1"},
							Properties: map[string]interface{}{
								"prop1": map[string]interface{}{
									"type": []string{"string"},
									"enum": []string{"value1", "value2"},
								},
							},
						},
						Traits:  nil,
						Context: nil,
					},
				},
			},
		},
		{
			name: "Custom type reference",
			property: &state.TrackingPlanPropertyArgs{
				Name:             "prop2",
				LocalID:          "prop-id-2",
				Description:      "Custom type property",
				Type:             "CustomType", // This would be the resolved value after dereferencing
				Required:         true,
				HasCustomTypeRef: true,
				HasItemTypesRef:  false,
			},
			expected: catalog.TrackingPlanUpsertEvent{
				Name:            "event1",
				Description:     "Event description",
				EventType:       "track",
				IdentitySection: "properties",
				Rules: catalog.TrackingPlanUpsertEventRules{
					Type: "object",
					Properties: struct {
						Properties *catalog.TrackingPlanUpsertEventProperties            `json:"properties,omitempty"`
						Traits     *catalog.TrackingPlanUpsertEventProperties            `json:"traits,omitempty"`
						Context    *catalog.TrackingPlanUpsertEventContextTraitsIdentity `json:"context,omitempty"`
					}{
						Properties: &catalog.TrackingPlanUpsertEventProperties{
							Type:                 "object",
							AdditionalProperties: false,
							Required:             []string{"prop2"},
							Properties: map[string]interface{}{
								"prop2": map[string]interface{}{
									"$ref": "#/$defs/CustomType",
								},
							},
						},
						Traits:  nil,
						Context: nil,
					},
				},
			},
		},
		{
			name: "Array with regular itemTypes",
			property: &state.TrackingPlanPropertyArgs{
				Name:             "prop3",
				LocalID:          "prop-id-3",
				Description:      "Array property",
				Type:             "array",
				Required:         false,
				Config:           map[string]interface{}{"itemTypes": []any{"string"}},
				HasCustomTypeRef: false,
				HasItemTypesRef:  false,
			},
			expected: catalog.TrackingPlanUpsertEvent{
				Name:            "event1",
				Description:     "Event description",
				EventType:       "track",
				IdentitySection: "properties",
				Rules: catalog.TrackingPlanUpsertEventRules{
					Type: "object",
					Properties: struct {
						Properties *catalog.TrackingPlanUpsertEventProperties            `json:"properties,omitempty"`
						Traits     *catalog.TrackingPlanUpsertEventProperties            `json:"traits,omitempty"`
						Context    *catalog.TrackingPlanUpsertEventContextTraitsIdentity `json:"context,omitempty"`
					}{
						Properties: &catalog.TrackingPlanUpsertEventProperties{
							Type:                 "object",
							AdditionalProperties: false,
							Required:             []string{},
							Properties: map[string]interface{}{
								"prop3": map[string]interface{}{
									"type": []string{"array"},
									"items": map[string]interface{}{
										"type": []any{"string"},
									},
								},
							},
						},
						Traits:  nil,
						Context: nil,
					},
				},
			},
		},
		{
			name: "Array with custom type in itemTypes",
			property: &state.TrackingPlanPropertyArgs{
				Name:             "prop4",
				LocalID:          "prop-id-4",
				Description:      "Array with custom type",
				Type:             "array",
				Required:         false,
				Config:           map[string]interface{}{"itemTypes": []any{"CustomType"}},
				HasCustomTypeRef: false,
				HasItemTypesRef:  true,
			},
			expected: catalog.TrackingPlanUpsertEvent{
				Name:            "event1",
				Description:     "Event description",
				EventType:       "track",
				IdentitySection: "properties",
				Rules: catalog.TrackingPlanUpsertEventRules{
					Type: "object",
					Properties: struct {
						Properties *catalog.TrackingPlanUpsertEventProperties            `json:"properties,omitempty"`
						Traits     *catalog.TrackingPlanUpsertEventProperties            `json:"traits,omitempty"`
						Context    *catalog.TrackingPlanUpsertEventContextTraitsIdentity `json:"context,omitempty"`
					}{
						Properties: &catalog.TrackingPlanUpsertEventProperties{
							Type:                 "object",
							AdditionalProperties: false,
							Required:             []string{},
							Properties: map[string]interface{}{
								"prop4": map[string]interface{}{
									"type": []string{"array"},
									"items": map[string]interface{}{
										"$ref": "#/$defs/CustomType",
									},
								},
							},
						},
						Traits:  nil,
						Context: nil,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create an event with the test property
			event := &state.TrackingPlanEventArgs{
				Name:           "event1",
				LocalID:        "event-id-1",
				Type:           "track",
				Description:    "Event description",
				AllowUnplanned: false,
				Properties:     []*state.TrackingPlanPropertyArgs{tc.property},
			}

			// Call GetUpsertEvent
			actual := datacatalog.GetUpsertEvent(event)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
