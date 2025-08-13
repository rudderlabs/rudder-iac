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
						Name:        "other-property-name",
						Description: "other-property-description",
						Type:        "object",
						LocalID:     "other-property-local-id",
						Properties: []*state.TrackingPlanPropertyArgs{
							{
								Name:        "nested-property-name",
								Description: "nested-property-description",
								Type:        "string",
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
			Name:        "user-profile",
			Description: "User profile object",
			Type:        "object",
			LocalID:     "user-profile-id",
			Required:    true,
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					Name:        "profile-name",
					Description: "User's profile name",
					Type:        "string",
					LocalID:     "profile-name-id",
					Required:    true, // Changed to false later
				},
			},
		}

		toArgs := factory.NewTrackingPlanArgsFactory().WithEvent(&state.TrackingPlanEventArgs{
			Name:           "event-name",
			Description:    "event-description",
			Type:           "event-type",
			LocalID:        "event-local-id",
			AllowUnplanned: false,
			Properties:     []*state.TrackingPlanPropertyArgs{propertyWithNested},
		}).Build()

		propertyWithNestedChanged := &state.TrackingPlanPropertyArgs{
			Name:        "user-profile",
			Description: "User profile object",
			Type:        "object",
			LocalID:     "user-profile-id",
			Required:    true,
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					Name:        "profile-name",
					Description: "User's profile name",
					Type:        "string",
					LocalID:     "profile-name-id",
					Required:    false, // Different from above
				},
			},
		}

		fromArgs := factory.NewTrackingPlanArgsFactory().WithEvent(&state.TrackingPlanEventArgs{
			Name:           "event-name",
			Description:    "event-description",
			Type:           "event-type",
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
			Name:        "user-profile",
			Description: "User profile object",
			Type:        "object",
			LocalID:     "user-profile-id",
			Required:    true,
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					Name:        "profile-name",
					Description: "User's profile name",
					Type:        "string",
					LocalID:     "profile-name-id",
					Required:    true,
				},
			},
		}

		fromArgs := factory.NewTrackingPlanArgsFactory().WithEvent(&state.TrackingPlanEventArgs{
			Name:           "event-name",
			Description:    "event-description",
			Type:           "event-type",
			LocalID:        "event-local-id",
			AllowUnplanned: false,
			Properties:     []*state.TrackingPlanPropertyArgs{propertyWithoutNested},
		}).Build()

		// Property with one nested property
		propertyWithOneNested := &state.TrackingPlanPropertyArgs{
			Name:        "user-profile",
			Description: "User profile object",
			Type:        "object",
			LocalID:     "user-profile-id",
			Required:    true,
			Properties: []*state.TrackingPlanPropertyArgs{
				{
					Name:        "profile-name",
					Description: "User's profile name",
					Type:        "string",
					LocalID:     "profile-name-id",
					Required:    true,
				},
				{
					Name:        "profile-email", // Added nested property
					Description: "User's profile email",
					Type:        "string",
					LocalID:     "profile-email-id",
					Required:    false,
				},
			},
		}

		toArgs := factory.NewTrackingPlanArgsFactory().WithEvent(&state.TrackingPlanEventArgs{
			Name:           "event-name",
			Description:    "event-description",
			Type:           "event-type",
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
		{
			name: "Property with nested properties (2 levels)",
			prop: &localcatalog.TPEventProperty{
				Name:        "user-profile",
				LocalID:     "user-profile-id",
				Description: "User profile object",
				Type:        "object",
				Required:    true,
				Properties: []*localcatalog.TPEventProperty{
					{
						Name:        "profile-name",
						LocalID:     "profile-name-id",
						Description: "User's profile name",
						Type:        "string",
						Required:    true,
					},
					{
						Name:        "profile-email",
						LocalID:     "profile-email-id",
						Description: "User's profile email",
						Type:        "string",
						Required:    false,
					},
				},
			},
			urnFromRef: func(ref string) string { return "" },
			expected: &state.TrackingPlanPropertyArgs{
				Name:             "user-profile",
				LocalID:          "user-profile-id",
				Description:      "User profile object",
				Type:             "object",
				Required:         true,
				HasCustomTypeRef: false,
				HasItemTypesRef:  false,
				Properties: []*state.TrackingPlanPropertyArgs{
					{
						Name:        "profile-name",
						LocalID:     "profile-name-id",
						Description: "User's profile name",
						Type:        "string",
						Required:    true,
					},
					{
						Name:        "profile-email",
						LocalID:     "profile-email-id",
						Description: "User's profile email",
						Type:        "string",
						Required:    false,
					},
				},
			},
		},
		{
			name: "Property with deeply nested properties (3 levels)",
			prop: &localcatalog.TPEventProperty{
				Name:        "user-profile",
				LocalID:     "user-profile-id",
				Description: "User profile object",
				Type:        "object",
				Required:    true,
				Properties: []*localcatalog.TPEventProperty{
					{
						Name:        "profile-settings",
						LocalID:     "profile-settings-id",
						Description: "User's profile settings",
						Type:        "object",
						Required:    false,
						Properties: []*localcatalog.TPEventProperty{
							{
								Name:        "theme-preference",
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
								LocalID:     "notifications-id",
								Description: "Notification preferences",
								Type:        "object",
								Required:    true,
								Properties: []*localcatalog.TPEventProperty{
									{
										Name:        "email-enabled",
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
			urnFromRef: func(ref string) string { return "" },
			expected: &state.TrackingPlanPropertyArgs{
				Name:             "user-profile",
				LocalID:          "user-profile-id",
				Description:      "User profile object",
				Type:             "object",
				Required:         true,
				HasCustomTypeRef: false,
				HasItemTypesRef:  false,
				Properties: []*state.TrackingPlanPropertyArgs{
					{
						Name:        "profile-settings",
						LocalID:     "profile-settings-id",
						Description: "User's profile settings",
						Type:        "object",
						Required:    false,
						Properties: []*state.TrackingPlanPropertyArgs{
							{
								Name:        "theme-preference",
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
								LocalID:     "notifications-id",
								Description: "Notification preferences",
								Type:        "object",
								Required:    true,
								Properties: []*state.TrackingPlanPropertyArgs{
									{
										Name:        "email-enabled",
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
		},
		{
			name: "Nested property with custom type reference",
			prop: &localcatalog.TPEventProperty{
				Name:        "user-profile",
				LocalID:     "user-profile-id",
				Description: "User profile object",
				Type:        "object",
				Required:    true,
				Properties: []*localcatalog.TPEventProperty{
					{
						Name:        "email-address",
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
				return ""
			},
			expected: &state.TrackingPlanPropertyArgs{
				Name:             "user-profile",
				LocalID:          "user-profile-id",
				Description:      "User profile object",
				Type:             "object",
				Required:         true,
				HasCustomTypeRef: false,
				HasItemTypesRef:  false,
				Properties: []*state.TrackingPlanPropertyArgs{
					{
						Name:             "email-address",
						LocalID:          "email-address-id",
						Description:      "User's email address",
						Type:             resources.PropertyRef{URN: "urn:custom-type:email-type", Property: "name"},
						Required:         true,
						HasCustomTypeRef: true,
						HasItemTypesRef:  false,
					},
				},
			},
		},
		{
			name: "Error in nested property processing",
			prop: &localcatalog.TPEventProperty{
				Name:        "user-profile",
				LocalID:     "user-profile-id",
				Description: "User profile object",
				Type:        "object",
				Required:    true,
				Properties: []*localcatalog.TPEventProperty{
					{
						Name:        "invalid-nested",
						LocalID:     "invalid-nested-id",
						Description: "Nested property with invalid custom type",
						Type:        "#/custom-types/invalid/type",
						Required:    true,
					},
				},
			},
			urnFromRef: func(ref string) string {
				return "" // Simulate not finding the custom type
			},
			expectedErrMsg: "processing nested property invalid-nested-id: unable to resolve custom type reference urn: #/custom-types/invalid/type",
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
				Name:             "simple-property",
				LocalID:          "simple-property-id",
				Description:      "Simple property description",
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
			name: "Property with deeply nested properties (3 levels)",
			property: &state.TrackingPlanPropertyArgs{
				Name:             "user-profile",
				LocalID:          "user-profile-id",
				Description:      "User profile object",
				Type:             "object",
				Required:         true,
				HasCustomTypeRef: false,
				HasItemTypesRef:  false,
				Properties: []*state.TrackingPlanPropertyArgs{
					{
						Name:        "profile-settings",
						LocalID:     "profile-settings-id",
						Description: "User's profile settings",
						Type:        "object",
						Required:    false,
						Properties: []*state.TrackingPlanPropertyArgs{
							{
								Name:        "theme-preference",
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
								LocalID:     "notifications-id",
								Description: "Notification preferences",
								Type:        "object",
								Required:    true,
								Properties: []*state.TrackingPlanPropertyArgs{
									{
										Name:        "email-enabled",
										LocalID:     "email-enabled-id",
										Description: "Email notifications enabled",
										Type:        "boolean",
										Required:    true,
									},
									{
										Name:        "push-enabled",
										LocalID:     "push-enabled-id",
										Description: "Push notifications enabled",
										Type:        "boolean",
										Required:    false,
									},
								},
							},
						},
					},
					{
						Name:        "profile-visibility",
						LocalID:     "profile-visibility-id",
						Description: "Profile visibility setting",
						Type:        "string",
						Required:    true,
						Config: map[string]interface{}{
							"enum": []string{"public", "private", "friends"},
						},
					},
				},
			},
		},
		{
			name: "Property with custom type reference and nested properties",
			property: &state.TrackingPlanPropertyArgs{
				Name:             "user-data",
				LocalID:          "user-data-id",
				Description:      "User data object",
				Type:             resources.PropertyRef{URN: "urn:custom-type:user-data", Property: "name"},
				Required:         true,
				HasCustomTypeRef: true,
				HasItemTypesRef:  false,
				Properties: []*state.TrackingPlanPropertyArgs{
					{
						Name:             "custom-field",
						LocalID:          "custom-field-id",
						Description:      "Custom field with custom type",
						Type:             resources.PropertyRef{URN: "urn:custom-type:custom-field", Property: "name"},
						Required:         true,
						HasCustomTypeRef: true,
						HasItemTypesRef:  false,
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
			assert.Equal(t, tc.property.Name, resourceData["name"])
			assert.Equal(t, tc.property.LocalID, resourceData["localId"])
			assert.Equal(t, tc.property.Description, resourceData["description"])
			assert.Equal(t, tc.property.Type, resourceData["type"])
			assert.Equal(t, tc.property.Required, resourceData["required"])
			assert.Equal(t, tc.property.HasCustomTypeRef, resourceData["hasCustomTypeRef"])
			assert.Equal(t, tc.property.HasItemTypesRef, resourceData["hasItemTypesRef"])

			if tc.property.Config != nil {
				assert.Equal(t, tc.property.Config, resourceData["config"])
			}

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
			assert.Equal(t, tc.property.Name, reconstructedProperty.Name)
			assert.Equal(t, tc.property.LocalID, reconstructedProperty.LocalID)
			assert.Equal(t, tc.property.Description, reconstructedProperty.Description)
			assert.Equal(t, tc.property.Type, reconstructedProperty.Type)
			assert.Equal(t, tc.property.Required, reconstructedProperty.Required)
			assert.Equal(t, tc.property.HasCustomTypeRef, reconstructedProperty.HasCustomTypeRef)
			assert.Equal(t, tc.property.HasItemTypesRef, reconstructedProperty.HasItemTypesRef)
			assert.Equal(t, tc.property.Config, reconstructedProperty.Config)

			// Verify nested properties match recursively
			assert.Len(t, reconstructedProperty.Properties, len(tc.property.Properties))
			for i, originalNested := range tc.property.Properties {
				reconstructedNested := reconstructedProperty.Properties[i]
				assert.Equal(t, originalNested.Name, reconstructedNested.Name)
				assert.Equal(t, originalNested.LocalID, reconstructedNested.LocalID)
				assert.Equal(t, originalNested.Description, reconstructedNested.Description)
				assert.Equal(t, originalNested.Type, reconstructedNested.Type)
				assert.Equal(t, originalNested.Required, reconstructedNested.Required)
				assert.Equal(t, originalNested.HasCustomTypeRef, reconstructedNested.HasCustomTypeRef)
				assert.Equal(t, originalNested.HasItemTypesRef, reconstructedNested.HasItemTypesRef)
				assert.Equal(t, originalNested.Config, reconstructedNested.Config)

				// For deeply nested, check one more level
				if len(originalNested.Properties) > 0 {
					assert.Len(t, reconstructedNested.Properties, len(originalNested.Properties))
					for j, originalDeepNested := range originalNested.Properties {
						reconstructedDeepNested := reconstructedNested.Properties[j]
						assert.Equal(t, originalDeepNested.Name, reconstructedDeepNested.Name)
						assert.Equal(t, originalDeepNested.LocalID, reconstructedDeepNested.LocalID)
						assert.Equal(t, originalDeepNested.Required, reconstructedDeepNested.Required)
					}
				}
			}
		})
	}
}