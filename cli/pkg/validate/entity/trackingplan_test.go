package entity_test

import (
	"fmt"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/pkg/validate/entity"
	"github.com/rudderlabs/rudder-iac/cli/pkg/validate/entity/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrackingPlanValidationRules_RequiredKeys(t *testing.T) {
	t.Parallel()

	rule := &entity.TrackingPlanRequiredKeysRule{}

	f1 := testutils.NewLocalCatalogTrackingPlanFactory().
		WithLocalID("").
		WithName("").
		WithDescription("Missing id and name trackingplan")

	f2 := testutils.NewLocalCatalogTrackingPlanFactory().
		WithLocalID("second_tracking_plan").
		WithName("Second Tracking Plan").
		WithRule(&localcatalog.TPRule{
			LocalID: "",                  // empty rule id
			Type:    "invalid_rule_type", // invalid rule type
		}).
		WithRule(&localcatalog.TPRule{
			LocalID: "rule_02",
			Type:    "event_rule",
			Event:   nil, // missing event
		})

	tp1 := f1.Build()
	tp2 := f2.Build()

	dc := testutils.NewDataCatalogFactory().
		WithTrackingPlan("trackingplan_1", tp1).
		WithTrackingPlan("trackingplan_2", tp2).
		Build()

	errs := rule.Validate("#/tp/trackingplan_1/", tp1, dc)
	require.Len(t, errs, 2)
	assert.Equal(t, errs, []entity.ValidationError{
		{
			Err:        entity.ErrMissingRequiredKeysID,
			Reference:  "#/tp/trackingplan_1/",
			EntityType: entity.TrackingPlan,
		},
		{
			Err:        entity.ErrMissingRequiredKeysName,
			Reference:  "#/tp/trackingplan_1/",
			EntityType: entity.TrackingPlan,
		},
	})

	errs = rule.Validate("#/tp/trackingplan_2/second_tracking_plan", tp2, dc)
	require.Len(t, errs, 3)

	assert.Equal(t, errs, []entity.ValidationError{
		{
			Err:        entity.ErrMissingRequiredKeysRuleID,
			Reference:  "#/tp/trackingplan_2/second_tracking_plan",
			EntityType: entity.TrackingPlan,
		},
		{
			Err:        fmt.Errorf("%w: %s", entity.ErrInvalidTrackingPlanEventRuleType, ""),
			Reference:  "#/tp/trackingplan_2/second_tracking_plan",
			EntityType: entity.TrackingPlan,
		},
		{
			Err:        fmt.Errorf("%w: %s", entity.ErrMissingRequiredKeysRuleEvent, "rule_02"),
			Reference:  "#/tp/trackingplan_2/second_tracking_plan",
			EntityType: entity.TrackingPlan,
		},
	})

}

func TestTrackingPlanValidationRules_DuplicateKeys(t *testing.T) {
	t.Parallel()

	f1 := testutils.NewLocalCatalogTrackingPlanFactory().
		WithLocalID("first_tracking_plan").
		WithName("First Tracking Plan").
		WithDescription("This is the first tracking plan").
		WithRule(&localcatalog.TPRule{
			LocalID: "rule_01",
			Type:    "event_rule",
		}).
		WithRule(&localcatalog.TPRule{
			LocalID: "rule_01", // similar rule_id
			Type:    "event_rule",
		})

	f2 := testutils.NewLocalCatalogTrackingPlanFactory().
		WithLocalID("first_tracking_plan"). // similar local_id
		WithName("First Tracking Plan").    // similar name as well
		WithDescription("This is the second tracking plan").
		WithRule(&localcatalog.TPRule{
			LocalID: "rule_02_01",
			Type:    "event_rule",
		}).
		WithRule(&localcatalog.TPRule{
			LocalID: "rule_01", // similar rule_id
			Type:    "event_rule",
		})

	tp1 := f1.Build()
	tp2 := f2.Build()

	cf := testutils.NewDataCatalogFactory().
		WithEvent("app_events", &localcatalog.Event{}).
		WithProperty("app_properties", &localcatalog.Property{}).
		WithTrackingPlan("trackingplan_1", tp1).
		WithTrackingPlan("trackingplan_2", tp2)

	catalog := cf.Build()

	rule := &entity.TrackingPlanDuplicateKeysRule{}
	errs := rule.Validate("#/tp/trackingplan_1/first_tracking_plan", tp1, catalog)
	require.Len(t, errs, 4)

	assert.Equal(t, errs, []entity.ValidationError{
		{
			Err:        entity.ErrDuplicateByID,
			Reference:  "#/tp/trackingplan_1/first_tracking_plan",
			EntityType: entity.TrackingPlan,
		},
		{
			Err:        entity.ErrDuplicateByName,
			Reference:  "#/tp/trackingplan_1/first_tracking_plan",
			EntityType: entity.TrackingPlan,
		},
		{
			Err:        fmt.Errorf("%w: rule: %s", entity.ErrDuplicateByID, "rule_01"),
			Reference:  "#/tp/trackingplan_1/first_tracking_plan",
			EntityType: entity.TrackingPlan,
		},
		{
			Err:        fmt.Errorf("%w: rule: %s", entity.ErrDuplicateByID, "rule_01"),
			Reference:  "#/tp/trackingplan_1/first_tracking_plan",
			EntityType: entity.TrackingPlan,
		},
	})

	errs = rule.Validate("#/tp/trackingplan_2/first_tracking_plan", tp2, catalog)
	require.Len(t, errs, 3)

	assert.Equal(t, errs, []entity.ValidationError{
		{
			Err:        entity.ErrDuplicateByID,
			Reference:  "#/tp/trackingplan_2/first_tracking_plan",
			EntityType: entity.TrackingPlan,
		},
		{
			Err:        entity.ErrDuplicateByName,
			Reference:  "#/tp/trackingplan_2/first_tracking_plan",
			EntityType: entity.TrackingPlan,
		},
		{
			Err:        fmt.Errorf("%w: rule: %s", entity.ErrDuplicateByID, "rule_01"),
			Reference:  "#/tp/trackingplan_2/first_tracking_plan",
			EntityType: entity.TrackingPlan,
		},
	})

}

func TestTrackingPlanValidationRules_RefKeys(t *testing.T) {
	t.Parallel()

	rule := entity.TrackingPlanRefRule{}

	f1 := testutils.NewLocalCatalogTrackingPlanFactory().
		WithLocalID("first_tracking_plan").
		WithName("First Tracking Plan").
		WithDescription("This is the first tracking plan").
		WithRule(&localcatalog.TPRule{
			LocalID: "rule_01",
			Type:    "event_rule",
			Event: &localcatalog.TPRuleEvent{
				Ref:            "app_events/invalid_event", // invalid event ref
				AllowUnplanned: true,
			},
			Properties: []*localcatalog.TPRuleProperty{
				{
					Ref: "app_properties/invalid_property", // invalid property ref
				},
				{
					Ref:      "#/properties/app_properties/some_property_ref", // valid ref but no property lookup
					Required: true,
				},
			},
		})

	dc := testutils.NewDataCatalogFactory().
		WithTrackingPlan("trackingplan_1", f1.Build()).
		Build()

	errs := rule.Validate("#/tp/trackingplan_1/first_tracking_plan", f1.Build(), dc)
	require.Len(t, errs, 3)

	assert.Equal(t, errs, []entity.ValidationError{
		{
			Err:        fmt.Errorf("%w: rule: %s event: %s", entity.ErrInvalidRefFormat, "rule_01", "app_events/invalid_event"),
			Reference:  "#/tp/trackingplan_1/first_tracking_plan",
			EntityType: entity.TrackingPlan,
		},
		{
			Err:        fmt.Errorf("%w: rule: %s property: %s", entity.ErrInvalidRefFormat, "rule_01", "app_properties/invalid_property"),
			Reference:  "#/tp/trackingplan_1/first_tracking_plan",
			EntityType: entity.TrackingPlan,
		},
		{
			Err:        fmt.Errorf("%w: rule: %s property: %s", entity.ErrMissingEntityFromRef, "rule_01", "#/properties/app_properties/some_property_ref"),
			Reference:  "#/tp/trackingplan_1/first_tracking_plan",
			EntityType: entity.TrackingPlan,
		},
	})
}
