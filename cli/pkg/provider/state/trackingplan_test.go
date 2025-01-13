package state_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/pkg/provider/state"
	"github.com/rudderlabs/rudder-iac/cli/pkg/provider/testutils/factory"
	"github.com/stretchr/testify/assert"
)

func TestTrackingPlanArgs_Diff(t *testing.T) {

	t.Run("no diff", func(t *testing.T) {
		t.Parallel()

		toArgs := factory.NewTrackingPlanArgsFactory().
			WithEvent(&state.TrackingPlanEventArgs{
				Name:           "event-name",
				Description:    "event-description",
				Type:           "event-type",
				LocalID:        "event-local-id",
				AllowUnplanned: false,
				Properties: []*state.TrackingPlanPropertyArgs{
					{
						Name:        "property-name",
						Description: "property-description",
						Type:        "property-type",
						Config: map[string]interface{}{
							"enum": []string{"value1", "value2"},
						},
					},
				},
			}).Build()

		diffed := toArgs.Diff(toArgs)
		assert.Equal(t, 0, len(diffed.Added))
		assert.Equal(t, 0, len(diffed.Updated))
		assert.Equal(t, 0, len(diffed.Deleted))

	})

	t.Run("event diff", func(t *testing.T) {
		t.Parallel()

		toArgs := factory.NewTrackingPlanArgsFactory().
			WithEvent(&state.TrackingPlanEventArgs{
				Name:            "event-name",
				Description:     "event-description",
				Type:            "event-type",
				LocalID:         "event-local-id-updated", // added
				AllowUnplanned:  false,
				IdentitySection: "traits",
			}).
			WithEvent(&state.TrackingPlanEventArgs{
				Name:            "event-name-1",
				Description:     "event-description-1",
				Type:            "event-type-1",
				LocalID:         "event-local-id-1",
				AllowUnplanned:  true, // updated
				IdentitySection: "",
			}).Build()

		fromArgs := factory.NewTrackingPlanArgsFactory().
			WithEvent(&state.TrackingPlanEventArgs{
				Name:            "event-name",
				Description:     "event-description",
				Type:            "event-type",
				LocalID:         "event-local-id",
				AllowUnplanned:  true,
				IdentitySection: "context.traits",
			}).
			WithEvent(&state.TrackingPlanEventArgs{
				Name:            "event-name-1",
				Description:     "event-description-1",
				Type:            "event-type-1",
				LocalID:         "event-local-id-1",
				AllowUnplanned:  false,
				IdentitySection: "",
			}).Build()

		diffed := fromArgs.Diff(toArgs)
		assert.Equal(t, 1, len(diffed.Added))
		assert.Equal(t, 1, len(diffed.Updated))
		assert.Equal(t, 1, len(diffed.Deleted))
	})

	t.Run("property diff", func(t *testing.T) {
		t.Parallel()

		toArgs := factory.NewTrackingPlanArgsFactory().WithEvent(&state.TrackingPlanEventArgs{
			Name:           "event-name",
			Description:    "event-description",
			Type:           "event-type",
			LocalID:        "event-local-id",
			AllowUnplanned: false,
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					Name:        "property-name",
					Description: "property-description",
					Type:        "property-type",
					LocalID:     "property-local-id",
					Config:      nil,
					Required:    false,
				},
			},
		}).Build()

		fromArgs := factory.NewTrackingPlanArgsFactory().WithEvent(&state.TrackingPlanEventArgs{
			Name:           "event-name",
			Description:    "event-description",
			Type:           "event-type",
			LocalID:        "event-local-id",
			AllowUnplanned: false,
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					Name:        "property-name",
					Description: "property-description",
					Type:        "property-type",
					LocalID:     "property-local-id",
					Config:      nil,
					Required:    true, // Same properties length
				},
			},
		}).Build()

		diffed := fromArgs.Diff(toArgs)
		assert.Equal(t, 0, len(diffed.Added))
		assert.Equal(t, 1, len(diffed.Updated))
		assert.Equal(t, 0, len(diffed.Deleted))

	})
}
