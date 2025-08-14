package state_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
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
				LocalID:        "event-local-id",
				AllowUnplanned: false,
				Properties: []*state.TrackingPlanPropertyArgs{
					{
						LocalID:  "property-local-id",
						Required: true,
					},
					{
						ID:      "other-property-name",
						LocalID: "other-property-local-id",
						Properties: []*state.TrackingPlanPropertyArgs{
							{
								ID:       "nested-property-name",
								LocalID:  "nested-property-local-id",
								Required: true,
							},
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
				LocalID:         "event-local-id-updated", // added
				AllowUnplanned:  false,
				IdentitySection: "traits",
			}).
			WithEvent(&state.TrackingPlanEventArgs{
				LocalID:         "event-local-id-1",
				AllowUnplanned:  true, // updated
				IdentitySection: "",
			}).Build()

		fromArgs := factory.NewTrackingPlanArgsFactory().
			WithEvent(&state.TrackingPlanEventArgs{
				LocalID:         "event-local-id",
				AllowUnplanned:  true,
				IdentitySection: "context.traits",
			}).
			WithEvent(&state.TrackingPlanEventArgs{
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
			LocalID:        "event-local-id",
			AllowUnplanned: false,
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					LocalID:  "property-local-id",
					Required: false,
				},
			},
		}).Build()

		fromArgs := factory.NewTrackingPlanArgsFactory().WithEvent(&state.TrackingPlanEventArgs{
			LocalID:        "event-local-id",
			AllowUnplanned: false,
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					LocalID:  "property-local-id",
					Required: true, // Same properties length
				},
			},
		}).Build()

		diffed := fromArgs.Diff(toArgs)
		assert.Equal(t, 0, len(diffed.Added))
		assert.Equal(t, 1, len(diffed.Updated))
		assert.Equal(t, 0, len(diffed.Deleted))

	})

	t.Run("nested property diff - nested property changed", func(t *testing.T) {
		t.Parallel()

		propertyWithNested := &state.TrackingPlanPropertyArgs{
			ID:       "user-profile",
			LocalID:  "user-profile-id",
			Required: true,
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					ID:       "profile-name",
					LocalID:  "profile-name-id",
					Required: true, // Changed to false later
				},
			},
		}

		toArgs := factory.NewTrackingPlanArgsFactory().WithEvent(&state.TrackingPlanEventArgs{
			LocalID:        "event-local-id",
			AllowUnplanned: false,
			Properties:     []*state.TrackingPlanPropertyArgs{propertyWithNested},
		}).Build()

		propertyWithNestedChanged := &state.TrackingPlanPropertyArgs{
			ID:       "user-profile",
			LocalID:  "user-profile-id",
			Required: true,
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					ID:       "profile-name",
					LocalID:  "profile-name-id",
					Required: false, // Different from above
				},
			},
		}

		fromArgs := factory.NewTrackingPlanArgsFactory().WithEvent(&state.TrackingPlanEventArgs{
			LocalID:        "event-local-id",
			AllowUnplanned: false,
			Properties:     []*state.TrackingPlanPropertyArgs{propertyWithNestedChanged},
		}).Build()

		diffed := fromArgs.Diff(toArgs)
		assert.Equal(t, 0, len(diffed.Added))
		assert.Equal(t, 1, len(diffed.Updated))
		assert.Equal(t, 0, len(diffed.Deleted))
	})

	t.Run("nested property diff - nested property added", func(t *testing.T) {
		t.Parallel()

		// Property with no nested properties
		propertyWithoutNested := &state.TrackingPlanPropertyArgs{
			ID:       "user-profile",
			LocalID:  "user-profile-id",
			Required: true,
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					ID:       "profile-name",
					LocalID:  "profile-name-id",
					Required: true,
				},
			},
		}

		fromArgs := factory.NewTrackingPlanArgsFactory().WithEvent(&state.TrackingPlanEventArgs{
			LocalID:        "event-local-id",
			AllowUnplanned: false,
			Properties:     []*state.TrackingPlanPropertyArgs{propertyWithoutNested},
		}).Build()

		// Property with one nested property
		propertyWithOneNested := &state.TrackingPlanPropertyArgs{
			ID:       "user-profile",
			LocalID:  "user-profile-id",
			Required: true,
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					ID:       "profile-name",
					LocalID:  "profile-name-id",
					Required: true,
				},
				{
					ID:       "profile-email", // Added nested property
					LocalID:  "profile-email-id",
					Required: false,
				},
			},
		}

		toArgs := factory.NewTrackingPlanArgsFactory().WithEvent(&state.TrackingPlanEventArgs{
			LocalID:        "event-local-id",
			AllowUnplanned: false,
			Properties:     []*state.TrackingPlanPropertyArgs{propertyWithOneNested},
		}).Build()

		diffed := fromArgs.Diff(toArgs)
		assert.Equal(t, 0, len(diffed.Added))
		assert.Equal(t, 1, len(diffed.Updated)) // Should detect difference in nested properties count
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
				Ref:         "#/properties/mypropertygroup/test-property-id",
				Description: "Test property description",
				Type:        "string",
				Required:    true,
				Config: map[string]interface{}{
					"enum": []string{"value1", "value2"},
				},
			},
			urnFromRef: func(ref string) string { return "property:test-property-id" },
			expected: &state.TrackingPlanPropertyArgs{
				LocalID:  "test-property-id",
				ID:       resources.PropertyRef{URN: "property:test-property-id", Property: "id"},
				Required: true,
			},
		},
		{
			name: "Custom type reference in Type",
			prop: &localcatalog.TPEventProperty{
				Name:        "test-property",
				LocalID:     "test-property-id",
				Ref:         "#/properties/mypropertygroup/test-property-id",
				Description: "Test property description",
				Type:        "#/custom-types/group/type-id",
				Required:    true,
			},
			urnFromRef: func(ref string) string {
				if ref == "#/custom-types/group/type-id" {
					return "urn:custom-type:type-id"
				}
				if ref == "#/properties/mypropertygroup/test-property-id" {
					return "property:test-property-id"
				}
				return ""
			},
			expected: &state.TrackingPlanPropertyArgs{
				LocalID:  "test-property-id",
				ID:       resources.PropertyRef{URN: "property:test-property-id", Property: "id"},
				Required: true,
			},
		},
		{
			name: "Array property with custom type reference in itemTypes",
			prop: &localcatalog.TPEventProperty{
				Name:        "test-array",
				LocalID:     "test-array-id",
				Ref:         "#/properties/mypropertygroup/test-array-id",
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
				if ref == "#/properties/mypropertygroup/test-array-id" {
					return "property:test-array-id"
				}
				return ""
			},
			expected: &state.TrackingPlanPropertyArgs{
				LocalID:  "test-array-id",
				ID:       resources.PropertyRef{URN: "property:test-array-id", Property: "id"},
				Required: false,
			},
		},
		{
			name: "Invalid custom type reference in Type", // this test does not throw an error anymore as we only store the propId now and not the whole inflated property definition
			prop: &localcatalog.TPEventProperty{
				Name:        "test-property",
				LocalID:     "test-property-id",
				Ref:         "#/properties/mypropertygroup/test-property-id",
				Description: "Test property description",
				Type:        "#/custom-types/group/invalid-id",
				Required:    true,
			},
			urnFromRef: func(ref string) string {
				if ref == "#/properties/mypropertygroup/test-property-id" {
					return "property:test-property-id"
				}
				return "" // Simulate not finding the custom type
			},
			expected: &state.TrackingPlanPropertyArgs{
				LocalID:  "test-property-id",
				ID:       resources.PropertyRef{URN: "property:test-property-id", Property: "id"},
				Required: true,
			},
		},
		{
			name: "Invalid custom type reference in itemTypes",
			prop: &localcatalog.TPEventProperty{
				Name:        "test-array",
				LocalID:     "test-array-id",
				Ref:         "#/properties/mypropertygroup/test-array-id",
				Description: "Test array property",
				Type:        "array",
				Required:    false,
				Config: map[string]interface{}{
					"itemTypes": []any{"#/custom-types/group/invalid-id"},
				},
			},
			urnFromRef: func(ref string) string {
				if ref == "#/properties/mypropertygroup/test-array-id" {
					return "property:test-array-id"
				}
				return "" // Simulate not finding the custom type
			},
			expected: &state.TrackingPlanPropertyArgs{
				LocalID:  "test-array-id",
				ID:       resources.PropertyRef{URN: "property:test-array-id", Property: "id"},
				Required: false,
			},
		},
		{
			name: "Property with nested properties (2 levels)",
			prop: &localcatalog.TPEventProperty{
				Name:        "user-profile",
				Ref:         "#/properties/mypropertygroup/user-profile-id",
				LocalID:     "user-profile-id",
				Description: "User profile object",
				Type:        "object",
				Required:    true,
				Properties: []*localcatalog.TPEventProperty{
					{
						Name:        "profile-name",
						Ref:         "#/properties/mypropertygroup/profile-name-id",
						LocalID:     "profile-name-id",
						Description: "User's profile name",
						Type:        "string",
						Required:    true,
					},
					{
						Name:        "profile-email",
						Ref:         "#/properties/mypropertygroup/profile-email-id",
						LocalID:     "profile-email-id",
						Description: "User's profile email",
						Type:        "string",
						Required:    false,
					},
				},
			},
			urnFromRef: func(ref string) string {
				if ref == "#/properties/mypropertygroup/user-profile-id" {
					return "property:user-profile-id"
				}
				if ref == "#/properties/mypropertygroup/profile-name-id" {
					return "property:profile-name-id"
				}
				if ref == "#/properties/mypropertygroup/profile-email-id" {
					return "property:profile-email-id"
				}
				return ""
			},
			expected: &state.TrackingPlanPropertyArgs{
				ID:                   resources.PropertyRef{URN: "property:user-profile-id", Property: "id"},
				LocalID:              "user-profile-id",
				Required:             true,
				AdditionalProperties: true,
				Properties: []*state.TrackingPlanPropertyArgs{
					{
						ID:       resources.PropertyRef{URN: "property:profile-name-id", Property: "id"},
						LocalID:  "profile-name-id",
						Required: true,
					},
					{
						ID:       resources.PropertyRef{URN: "property:profile-email-id", Property: "id"},
						LocalID:  "profile-email-id",
						Required: false,
					},
				},
			},
		},
		{
			name: "Property with deeply nested properties (3 levels)",
			prop: &localcatalog.TPEventProperty{
				Name:        "user-profile",
				Ref:         "#/properties/mypropertygroup/user-profile-id",
				LocalID:     "user-profile-id",
				Description: "User profile object",
				Type:        "object",
				Required:    true,
				Properties: []*localcatalog.TPEventProperty{
					{
						Name:        "profile-settings",
						Ref:         "#/properties/mypropertygroup/profile-settings-id",
						LocalID:     "profile-settings-id",
						Description: "User's profile settings",
						Type:        "object",
						Required:    false,
						Properties: []*localcatalog.TPEventProperty{
							{
								Name:        "theme-preference",
								Ref:         "#/properties/mypropertygroup/theme-preference-id",
								LocalID:     "theme-preference-id",
								Description: "User's theme preference",
								Type:        "string",
								Required:    false,
								Config: map[string]interface{}{
									"enum": []string{"light", "dark"},
								},
							},
							{
								Name:        "notifications",
								Ref:         "#/properties/mypropertygroup/notifications-id",
								LocalID:     "notifications-id",
								Description: "Notification preferences",
								Type:        "object",
								Required:    true,
								Properties: []*localcatalog.TPEventProperty{
									{
										Name:        "email-enabled",
										Ref:         "#/properties/mypropertygroup/email-enabled-id",
										LocalID:     "email-enabled-id",
										Description: "Email notifications enabled",
										Type:        "boolean",
										Required:    true,
									},
								},
							},
						},
					},
				},
			},
			urnFromRef: func(ref string) string {
				if ref == "#/properties/mypropertygroup/user-profile-id" {
					return "property:user-profile-id"
				}
				if ref == "#/properties/mypropertygroup/profile-settings-id" {
					return "property:profile-settings-id"
				}
				if ref == "#/properties/mypropertygroup/theme-preference-id" {
					return "property:theme-preference-id"
				}
				if ref == "#/properties/mypropertygroup/notifications-id" {
					return "property:notifications-id"
				}
				if ref == "#/properties/mypropertygroup/email-enabled-id" {
					return "property:email-enabled-id"
				}
				return ""
			},
			expected: &state.TrackingPlanPropertyArgs{
				ID:                   resources.PropertyRef{URN: "property:user-profile-id", Property: "id"},
				LocalID:              "user-profile-id",
				Required:             true,
				AdditionalProperties: true,
				Properties: []*state.TrackingPlanPropertyArgs{
					{
						ID:                   resources.PropertyRef{URN: "property:profile-settings-id", Property: "id"},
						LocalID:              "profile-settings-id",
						Required:             false,
						AdditionalProperties: true,
						Properties: []*state.TrackingPlanPropertyArgs{
							{
								ID:       resources.PropertyRef{URN: "property:theme-preference-id", Property: "id"},
								LocalID:  "theme-preference-id",
								Required: false,
							},
							{
								ID:                   resources.PropertyRef{URN: "property:notifications-id", Property: "id"},
								LocalID:              "notifications-id",
								Required:             true,
								AdditionalProperties: true,
								Properties: []*state.TrackingPlanPropertyArgs{
									{
										ID:       resources.PropertyRef{URN: "property:email-enabled-id", Property: "id"},
										LocalID:  "email-enabled-id",
										Required: true,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Nested property with custom type reference",
			prop: &localcatalog.TPEventProperty{
				Name:        "user-profile",
				Ref:         "#/properties/mypropertygroup/user-profile-id",
				LocalID:     "user-profile-id",
				Description: "User profile object",
				Type:        "object",
				Required:    true,
				Properties: []*localcatalog.TPEventProperty{
					{
						Name:        "email-address",
						Ref:         "#/properties/mypropertygroup/email-address-id",
						LocalID:     "email-address-id",
						Description: "User's email address",
						Type:        "#/custom-types/user-types/email-type",
						Required:    true,
					},
				},
			},
			urnFromRef: func(ref string) string {
				if ref == "#/custom-types/user-types/email-type" {
					return "urn:custom-type:email-type"
				}
				if ref == "#/properties/mypropertygroup/user-profile-id" {
					return "property:user-profile-id"
				}
				if ref == "#/properties/mypropertygroup/email-address-id" {
					return "property:email-address-id"
				}
				return ""
			},
			expected: &state.TrackingPlanPropertyArgs{
				ID:                   resources.PropertyRef{URN: "property:user-profile-id", Property: "id"},
				LocalID:              "user-profile-id",
				Required:             true,
				AdditionalProperties: true,
				Properties: []*state.TrackingPlanPropertyArgs{
					{
						ID:       resources.PropertyRef{URN: "property:email-address-id", Property: "id"},
						LocalID:  "email-address-id",
						Required: true,
					},
				},
			},
		},
		{
			name: "Invalid property reference in nested property",
			prop: &localcatalog.TPEventProperty{
				Name:        "user-profile",
				LocalID:     "user-profile-id",
				Ref:         "#/properties/mypropertygroup/user-profile-id",
				Description: "User profile object",
				Type:        "object",
				Required:    true,
				Properties: []*localcatalog.TPEventProperty{
					{
						Name:        "invalid-nested",
						LocalID:     "invalid-nested-id",
						Ref:         "#/properties/mypropertygroup/invalid-nested-id",
						Description: "Nested property with invalid custom type",
						Type:        "string",
						Required:    true,
					},
				},
			},
			urnFromRef: func(ref string) string {
				if ref == "#/properties/mypropertygroup/user-profile-id" {
					return "property:user-profile-id"
				}
				return "" // Simulate not finding the custom type
			},
			expectedErrMsg: "processing nested property invalid-nested-id: unable to resolve ref to the property urn: #/properties/mypropertygroup/invalid-nested-id",
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

func TestTrackingPlanPropertyArgs_ToResourceDataAndFromResourceData(t *testing.T) {
	tests := []struct {
		name     string
		property *state.TrackingPlanPropertyArgs
	}{
		{
			name: "Simple property without nested properties",
			property: &state.TrackingPlanPropertyArgs{
				ID:       resources.PropertyRef{URN: "property:simple-property-id", Property: "id"},
				LocalID:  "simple-property-id",
				Required: true,
			},
		},
		{
			name: "Property with deeply nested properties (3 levels)",
			property: &state.TrackingPlanPropertyArgs{
				ID:       resources.PropertyRef{URN: "property:user-profile-id", Property: "id"},
				LocalID:  "user-profile-id",
				Required: true,
				Properties: []*state.TrackingPlanPropertyArgs{
					{
						ID:       resources.PropertyRef{URN: "property:profile-settings-id", Property: "id"},
						LocalID:  "profile-settings-id",
						Required: false,
						Properties: []*state.TrackingPlanPropertyArgs{
							{
								ID:       resources.PropertyRef{URN: "property:theme-preference-id", Property: "id"},
								LocalID:  "theme-preference-id",
								Required: false,
							},
							{
								ID:       resources.PropertyRef{URN: "property:notifications-id", Property: "id"},
								LocalID:  "notifications-id",
								Required: true,
								Properties: []*state.TrackingPlanPropertyArgs{
									{
										ID:       resources.PropertyRef{URN: "property:email-enabled-id", Property: "id"},
										LocalID:  "email-enabled-id",
										Required: true,
									},
									{
										ID:       resources.PropertyRef{URN: "property:push-enabled-id", Property: "id"},
										LocalID:  "push-enabled-id",
										Required: false,
									},
								},
							},
						},
					},
					{
						ID:       resources.PropertyRef{URN: "property:profile-visibility-id", Property: "id"},
						LocalID:  "profile-visibility-id",
						Required: true,
					},
				},
			},
		},
		{
			name: "Property with custom type reference and nested properties",
			property: &state.TrackingPlanPropertyArgs{
				ID:       resources.PropertyRef{URN: "property:user-data-id", Property: "id"},
				LocalID:  "user-data-id",
				Required: true,
				Properties: []*state.TrackingPlanPropertyArgs{
					{
						ID:       resources.PropertyRef{URN: "property:custom-field-id", Property: "id"},
						LocalID:  "custom-field-id",
						Required: true,
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Test ToResourceData
			resourceData := tc.property.ToResourceData()

			// Verify basic fields are present
			assert.Equal(t, tc.property.ID, resourceData["id"])
			assert.Equal(t, tc.property.LocalID, resourceData["localId"])
			assert.Equal(t, tc.property.Required, resourceData["required"])

			// Verify nested properties are present if they exist
			if len(tc.property.Properties) > 0 {
				nestedProps, ok := resourceData["properties"].([]map[string]interface{})
				require.True(t, ok, "properties should be []map[string]interface{}")
				assert.Len(t, nestedProps, len(tc.property.Properties))
			}

			// Test FromResourceData roundtrip
			reconstructedProperty := &state.TrackingPlanPropertyArgs{}
			reconstructedProperty.FromResourceData(resourceData)

			// Verify the reconstructed property matches the original
			assert.Equal(t, "", reconstructedProperty.ID)
			assert.Equal(t, tc.property.LocalID, reconstructedProperty.LocalID)
			assert.Equal(t, tc.property.Required, reconstructedProperty.Required)

			// Verify nested properties match recursively
			assert.Len(t, reconstructedProperty.Properties, len(tc.property.Properties))
			for i, originalNested := range tc.property.Properties {
				reconstructedNested := reconstructedProperty.Properties[i]
				assert.Equal(t, "", reconstructedNested.ID)
				assert.Equal(t, originalNested.LocalID, reconstructedNested.LocalID)
				assert.Equal(t, originalNested.Required, reconstructedNested.Required)

				// For deeply nested, check one more level
				if len(originalNested.Properties) > 0 {
					assert.Len(t, reconstructedNested.Properties, len(originalNested.Properties))
					for j, originalDeepNested := range originalNested.Properties {
						reconstructedDeepNested := reconstructedNested.Properties[j]
						assert.Equal(t, "", reconstructedDeepNested.ID)
						assert.Equal(t, originalDeepNested.LocalID, reconstructedDeepNested.LocalID)
						assert.Equal(t, originalDeepNested.Required, reconstructedDeepNested.Required)
					}
				}
			}
		})
	}
}
