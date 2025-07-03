package state_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/testutils/factory"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestTrackingPlanPropertyArgs_FromCatalogTrackingPlanEventProperty(t *testing.T) {
	tests := []struct {
		name           string
		prop           *localcatalog.TPEventProperty
		urnFromRef     func(string) string
		expected       *state.TrackingPlanPropertyArgs
		expectedErrMsg string
	}{
		{
			name: "Regular string type property",
			prop: &localcatalog.TPEventProperty{
				Name:        "test-property",
				LocalID:     "test-property-id",
				Description: "Test property description",
				Type:        "string",
				Required:    true,
				Config: map[string]interface{}{
					"enum": []string{"value1", "value2"},
				},
			},
			urnFromRef: func(ref string) string { return "" },
			expected: &state.TrackingPlanPropertyArgs{
				Name:             "test-property",
				LocalID:          "test-property-id",
				Description:      "Test property description",
				Type:             "string",
				Required:         true,
				HasCustomTypeRef: false,
				HasItemTypesRef:  false,
				Config: map[string]interface{}{
					"enum": []string{"value1", "value2"},
				},
			},
		},
		{
			name: "Custom type reference in Type",
			prop: &localcatalog.TPEventProperty{
				Name:        "test-property",
				LocalID:     "test-property-id",
				Description: "Test property description",
				Type:        "#/custom-types/group/type-id",
				Required:    true,
			},
			urnFromRef: func(ref string) string {
				if ref == "#/custom-types/group/type-id" {
					return "urn:custom-type:type-id"
				}
				return ""
			},
			expected: &state.TrackingPlanPropertyArgs{
				Name:             "test-property",
				LocalID:          "test-property-id",
				Description:      "Test property description",
				Type:             resources.PropertyRef{URN: "urn:custom-type:type-id", Property: "name"},
				Required:         true,
				HasCustomTypeRef: true,
				HasItemTypesRef:  false,
			},
		},
		{
			name: "Array property with custom type reference in itemTypes",
			prop: &localcatalog.TPEventProperty{
				Name:        "test-array",
				LocalID:     "test-array-id",
				Description: "Test array property",
				Type:        "array",
				Required:    false,
				Config: map[string]interface{}{
					"itemTypes": []any{"#/custom-types/group/type-id"},
				},
			},
			urnFromRef: func(ref string) string {
				if ref == "#/custom-types/group/type-id" {
					return "urn:custom-type:type-id"
				}
				return ""
			},
			expected: &state.TrackingPlanPropertyArgs{
				Name:             "test-array",
				LocalID:          "test-array-id",
				Description:      "Test array property",
				Type:             "array",
				Required:         false,
				HasCustomTypeRef: false,
				HasItemTypesRef:  true,
				Config: map[string]interface{}{
					"itemTypes": []any{
						resources.PropertyRef{URN: "urn:custom-type:type-id", Property: "name"},
					},
				},
			},
		},
		{
			name: "Invalid custom type reference in Type",
			prop: &localcatalog.TPEventProperty{
				Name:        "test-property",
				LocalID:     "test-property-id",
				Description: "Test property description",
				Type:        "#/custom-types/group/invalid-id",
				Required:    true,
			},
			urnFromRef: func(ref string) string {
				return "" // Simulate not finding the custom type
			},
			expectedErrMsg: "unable to resolve custom type reference urn: #/custom-types/group/invalid-id",
		},
		{
			name: "Invalid custom type reference in itemTypes",
			prop: &localcatalog.TPEventProperty{
				Name:        "test-array",
				LocalID:     "test-array-id",
				Description: "Test array property",
				Type:        "array",
				Required:    false,
				Config: map[string]interface{}{
					"itemTypes": []any{"#/custom-types/group/invalid-id"},
				},
			},
			urnFromRef: func(ref string) string {
				return "" // Simulate not finding the custom type
			},
			expectedErrMsg: "unable to resolve custom type reference urn in itemTypes: #/custom-types/group/invalid-id",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := &state.TrackingPlanPropertyArgs{}

			err := args.FromCatalogTrackingPlanEventProperty(tc.prop, tc.urnFromRef)

			if tc.expectedErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErrMsg)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expected, args)
		})
	}
}
