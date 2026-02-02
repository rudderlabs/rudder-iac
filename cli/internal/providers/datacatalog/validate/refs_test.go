package validate

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	catalog "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/stretchr/testify/assert"
)

func TestPropertyItemTypesCustomTypeReferences(t *testing.T) {
	validator := &RefValidator{}

	// Create a test custom type
	testCustomType := catalog.CustomTypeV1{
		LocalID:     "EmailType",
		Name:        "EmailType",
		Description: "Custom type for email validation",
		Type:        "string",
		Config: map[string]interface{}{
			"format": "email",
		},
	}

	testCases := []struct {
		name          string
		properties    []catalog.PropertyV1
		customTypes   []catalog.CustomTypeV1
		expectedErrs  int
		errorContains []string
	}{
		{
			name: "valid custom type reference in property item_type",
			properties: []catalog.PropertyV1{
				{
					LocalID:     "emailList",
					Name:        "Email List",
					Description: "List of user emails",
					Type:        "array",
					ItemType:    "#custom-type:EmailType",
				},
			},
			customTypes: []catalog.CustomTypeV1{
				testCustomType,
			},
			expectedErrs: 0,
		},
		{
			name: "invalid custom type reference format in property item_type",
			properties: []catalog.PropertyV1{
				{
					LocalID:     "emailList",
					Name:        "Email List",
					Description: "List of user emails",
					Type:        "array",
					ItemType:    "#custom-type:", // Missing type ID
				},
			},
			customTypes: []catalog.CustomTypeV1{
				testCustomType,
			},
			expectedErrs:  1,
			errorContains: []string{"custom type reference in item_type has invalid format"},
		},
		{
			name: "reference to non-existent custom type in property item_type",
			properties: []catalog.PropertyV1{
				{
					LocalID:     "emailList",
					Name:        "Email List",
					Description: "List of user emails",
					Type:        "array",
					ItemType:    "#custom-type:NonExistentType",
				},
			},
			customTypes: []catalog.CustomTypeV1{
				testCustomType,
			},
			expectedErrs:  1,
			errorContains: []string{"custom type reference '#custom-type:NonExistentType' in item_type not found in catalog"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dc := &catalog.DataCatalog{
				Properties:  tc.properties,
				CustomTypes: tc.customTypes,
			}

			errs := validator.Validate(dc)

			assert.Len(t, errs, tc.expectedErrs)
			for i, errContains := range tc.errorContains {
				if i < len(errs) {
					assert.Contains(t, errs[i].error.Error(), errContains)
				}
			}
		})
	}
}

func TestEventCategoryReferences(t *testing.T) {
	validator := &RefValidator{}

	// Create a test category
	testCategory := catalog.Category{
		LocalID: "user_actions",
		Name:    "User Actions",
	}

	testCases := []struct {
		name          string
		events        []catalog.Event
		categories    []catalog.Category
		expectedErrs  int
		errorContains []string
	}{
		{
			name: "valid category reference in event",
			events: []catalog.Event{
				{
					LocalID:     "user_signup",
					Name:        "User Signup",
					Type:        "track",
					Description: "User signed up for the app",
					CategoryRef: stringPtr("#category:user_actions"),
				},
			},
			categories: []catalog.Category{
				testCategory,
			},
			expectedErrs: 0,
		},
		{
			name: "event without category reference should pass validation",
			events: []catalog.Event{
				{
					LocalID:     "user_signup",
					Name:        "User Signup",
					Type:        "track",
					Description: "User signed up for the app",
					CategoryRef: nil,
				},
			},
			categories:   nil,
			expectedErrs: 0,
		},
		{
			name: "invalid category reference format in event",
			events: []catalog.Event{
				{
					LocalID:     "user_signup",
					Name:        "User Signup",
					Type:        "track",
					Description: "User signed up for the app",
					CategoryRef: stringPtr("#category:"), // Missing category ID
				},
			},
			categories: []catalog.Category{
				testCategory,
			},
			expectedErrs:  1,
			errorContains: []string{"the category field value is invalid. It should always be a reference and must follow the format '#category:<id>'"},
		},
		{
			name: "reference to non-existent category in event",
			events: []catalog.Event{
				{
					LocalID:     "user_signup",
					Name:        "User Signup",
					Type:        "track",
					Description: "User signed up for the app",
					CategoryRef: stringPtr("#category:non_existent_category"),
				},
			},
			categories: []catalog.Category{
				testCategory,
			},
			expectedErrs:  1,
			errorContains: []string{"category reference '#category:non_existent_category' not found in catalog"},
		},
		{
			name: "completely malformed category reference",
			events: []catalog.Event{
				{
					LocalID:     "user_signup",
					Name:        "User Signup",
					Type:        "track",
					Description: "User signed up for the app",
					CategoryRef: stringPtr("user_actions"),
				},
			},
			categories: []catalog.Category{
				testCategory,
			},
			expectedErrs:  1,
			errorContains: []string{"the category field value is invalid. It should always be a reference and must follow the format '#category:<id>'"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dc := &catalog.DataCatalog{
				Events:     tc.events,
				Categories: tc.categories,
			}

			errs := validator.Validate(dc)

			assert.Len(t, errs, tc.expectedErrs)
			for i, errContains := range tc.errorContains {
				if i < len(errs) {
					assert.Contains(t, errs[i].error.Error(), errContains)
				}
			}
		})
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

func TestVariantsReferenceValidation(t *testing.T) {
	validator := &RefValidator{}

	// Create test properties for reference validation
	testProperties := []catalog.PropertyV1{
		{LocalID: "page_name", Name: "Page Name", Type: "string"},
		{LocalID: "search_term", Name: "Search Term", Type: "string"},
		{LocalID: "product_id", Name: "Product ID", Type: "string"},
		{LocalID: "user_id", Name: "User ID", Type: "string"},
	}

	testEvents := []catalog.Event{
		{LocalID: "test-event", Name: "Test Event", Type: "track"},
	}

	testCases := []struct {
		name          string
		trackingPlans []*catalog.TrackingPlanV1
		customTypes   []catalog.CustomTypeV1
		errors        []ValidationError
	}{
		{
			name: "valid variants references in tracking plan",
			trackingPlans: []*catalog.TrackingPlanV1{
				{
					LocalID: "test-tp",
					Name:    "Test Tracking Plan",
					Rules: []*catalog.TPRuleV1{
						{
							LocalID: "test-rule",
							Type:    "event_rule",
							Event: "#event:test-event",
							AdditionalProperties: false,
							Properties: []*catalog.TPRulePropertyV1{
								{
									Property: "#property:page_name",
									Required: true,
								},
							},
							Variants: catalog.Variants{
								{
									Type:          "discriminator",
									Discriminator: "#property:page_name",
									Cases: []catalog.VariantCase{
										{
											DisplayName: "Search Page",
											Match:       []any{"search", "search_bar"},
											Properties: []catalog.PropertyReference{
												{Ref: "#property:search_term", Required: true},
											},
										},
									},
									Default: []catalog.PropertyReference{
										{Ref: "#property:product_id", Required: true},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "valid variants references in custom type",
			customTypes: []catalog.CustomTypeV1{
				{
					LocalID:     "TestType",
					Name:        "TestType",
					Description: "Test custom type with variants",
					Type:        "object",
					Properties: []catalog.CustomTypePropertyV1{
						{
							Property: "#property:user_id",
							Required: true,
						},
					},
					Variants: catalog.VariantsV1{
						{
							Type:          "discriminator",
							Discriminator: "#property:user_id",
							Cases: []catalog.VariantCaseV1{
								{
									DisplayName: "Admin User",
									Match:       []any{"admin", "superuser"},
									Properties: []catalog.PropertyReferenceV1{
										{Property: "#property:product_id", Required: true},
									},
								},
							},
						},
					},
				},
			},
			errors: nil,
		},
		{
			name: "invalid discriminator and property reference in variant case",
			trackingPlans: []*catalog.TrackingPlanV1{
				{
					LocalID: "test-tp",
					Name:    "Test Tracking Plan",
					Rules: []*catalog.TPRuleV1{
						{
							LocalID: "test-rule",
							Type:    "event_rule",
							Event: "#event:test-event",
							AdditionalProperties: false,
							Variants: catalog.Variants{
								{
									Type:          "discriminator",
									Discriminator: "page_name",
									Cases: []catalog.VariantCase{
										{
											DisplayName: "Search Page",
											Match:       []any{"search"},
											Properties: []catalog.PropertyReference{
												{Ref: "#property:non_existent_property", Required: true},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			errors: []ValidationError{
				{
					error:     errors.New("property reference '#property:non_existent_property' not found in catalog"),
					Reference: "#tp:test-tp/event_rule/test-rule/variants[0]/cases[0]/properties[0]",
				},
				{
					error:     errors.New("discriminator reference has invalid format, should be #property:<id>"),
					Reference: "#tp:test-tp/event_rule/test-rule/variants[0]",
				},
			},
		},
		{
			name: "invalid property reference in variant default",
			trackingPlans: []*catalog.TrackingPlanV1{
				{
					LocalID: "test-tp",
					Name:    "Test Tracking Plan",
					Rules: []*catalog.TPRuleV1{
						{
							LocalID: "test-rule",
							Type:    "event_rule",
							Event: "#event:test-event",
							AdditionalProperties: false,
							Properties: []*catalog.TPRulePropertyV1{
								{
									Property: "#property:page_name",
									Required: true,
								},
							},
							Variants: catalog.Variants{
								{
									Type:          "discriminator",
									Discriminator: "#property:page_name",
									Cases: []catalog.VariantCase{
										{
											DisplayName: "Search Page",
											Match:       []any{"search"},
											Properties: []catalog.PropertyReference{
												{Ref: "#property:search_term", Required: true},
											},
										},
									},
									Default: []catalog.PropertyReference{
										{Ref: "#property:non_existent_property", Required: true},
									},
								},
							},
						},
					},
				},
			},
			errors: []ValidationError{
				{
					error:     fmt.Errorf("default property reference '#property:non_existent_property' not found in catalog"),
					Reference: "#tp:test-tp/event_rule/test-rule/variants[0]/default/properties[0]",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dc := &catalog.DataCatalog{
				Properties:    testProperties,
				TrackingPlans: tc.trackingPlans,
				CustomTypes:   tc.customTypes,
				Events:        testEvents,
			}

			errs := validator.Validate(dc)
			assert.Len(t, errs, len(tc.errors), "Expected %d validation errors, got %d", len(tc.errors), len(errs))

			if len(tc.errors) > 0 {
				for _, actual := range errs {
					found := false

					for _, expected := range tc.errors {
						if actual.Error() == expected.Error() && actual.Reference == expected.Reference {
							found = true
							break
						}
					}

					if !found {
						assert.Failf(
							t,
							"variants_reference_validation_failures",
							"Expected to find error: %s with reference: %s in expected",
							actual.Error(),
							actual.Reference,
						)
					}
				}
			}
		})
	}
}

func TestRecursiveReferenceValidation(t *testing.T) {
	validator := &RefValidator{}

	// Create test properties for nested reference validation
	testProperties := []catalog.PropertyV1{
		{LocalID: "user_profile", Name: "User Profile", Type: "object"},
		{LocalID: "profile_name", Name: "Profile Name", Type: "string"},
		{LocalID: "profile_settings", Name: "Profile Settings", Type: "object"},
		{LocalID: "theme_preference", Name: "Theme Preference", Type: "string"},
		{LocalID: "notification_prefs", Name: "Notification Preferences", Type: "object"},
		{LocalID: "email_enabled", Name: "Email Enabled", Type: "boolean"},
	}

	testEvents := []catalog.Event{
		{LocalID: "user_signup", Name: "User Signup", Type: "track"},
	}

	testCases := []struct {
		name          string
		trackingPlans []*catalog.TrackingPlanV1
		expectedErrs  int
		errorContains []string
	}{
		{
			name: "valid nested property references (2 levels)",
			trackingPlans: []*catalog.TrackingPlanV1{
				{
					LocalID: "test-tp",
					Name:    "Test Tracking Plan",
					Rules: []*catalog.TPRuleV1{
						{
							LocalID: "test-rule",
							Type:    "event_rule",
							Event: "#event:user_signup",
							AdditionalProperties: false,
							Properties: []*catalog.TPRulePropertyV1{
								{
									Property: "#property:user_profile",
									Required: true,
									Properties: []*catalog.TPRulePropertyV1{
										{
											Property: "#property:profile_name",
											Required: true,
										},
									},
								},
							},
						},
					},
				},
			},
			expectedErrs: 0,
		},
		{
			name: "valid nested property references (3 levels deep)",
			trackingPlans: []*catalog.TrackingPlanV1{
				{
					LocalID: "test-tp",
					Name:    "Test Tracking Plan",
					Rules: []*catalog.TPRuleV1{
						{
							LocalID: "test-rule",
							Type:    "event_rule",
							Event: "#/events/test-group/user_signup",
							Properties: []*catalog.TPRulePropertyV1{
								{
									Property: "#property:user_profile",
									Required: true,
									Properties: []*catalog.TPRulePropertyV1{
										{
											Property: "#property:profile_settings",
											Required: true,
											Properties: []*catalog.TPRulePropertyV1{
												{
													Property: "#property:theme_preference",
													Required: true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expectedErrs: 0,
		},
		{
			name: "invalid nested property reference (non-existent property)",
			trackingPlans: []*catalog.TrackingPlanV1{
				{
					LocalID: "test-tp",
					Name:    "Test Tracking Plan",
					Rules: []*catalog.TPRuleV1{	
						{
							LocalID: "test-rule",
							Type:    "event_rule",
							Event: "#/events/test-group/user_signup",
							Properties: []*catalog.TPRulePropertyV1{
								{
									Property: "#property:user_profile",
									Required: true,
									Properties: []*catalog.TPRulePropertyV1{
										{
											Property: "#property:nonexistent_property",
											Required: true,
										},
									},
								},
							},
						},
					},
				},
			},
			expectedErrs:  1,
			errorContains: []string{"property reference '#property:nonexistent_property' in rule '#tp:test-tp/rules/test-rule' not found in catalog"},
		},
		{
			name: "invalid nested property reference format",
			trackingPlans: []*catalog.TrackingPlanV1{
				{
					LocalID: "test-tp",
					Name:    "Test Tracking Plan",
					Rules: []*catalog.TPRuleV1{
						{
							LocalID: "test-rule",
							Type:    "event_rule",
							Event: "#event:user_signup",
							AdditionalProperties: false,
							Properties: []*catalog.TPRulePropertyV1{
								{
									Property: "#property:user_profile",
									Required: true,
									Properties: []*catalog.TPRulePropertyV1{
										{
											Property: "#property:", // Invalid format - missing property ID
											Required: true,
										},
									},
								},
							},
						},
					},
				},
			},
			expectedErrs:  1,
			errorContains: []string{"property reference '#property:' has invalid format in rule '#tp:test-tp/rules/test-rule'. Should be '#property:<id>'"},
		},
		{
			name: "multiple nested reference errors (different levels)",
			trackingPlans: []*catalog.TrackingPlanV1{
				{
					LocalID: "test-tp",
					Name:    "Test Tracking Plan",
					Rules: []*catalog.TPRuleV1{
						{
							LocalID: "test-rule",
							Type:    "event_rule",
							Event: "#event:user_signup",
							AdditionalProperties: false,
							Properties: []*catalog.TPRulePropertyV1{
								{
									Property: "#property:user_profile",
									Required: true,
									Properties: []*catalog.TPRulePropertyV1{
										{
											Property: "#property:invalid_property1", // Non-existent at level 2
											Required: true,
										},
										{
											Property: "#property:profile_settings",
											Required: true,
											Properties: []*catalog.TPRulePropertyV1{
												{
													Property: "#property:invalid_property2", // Non-existent at level 3
													Required: true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expectedErrs: 2,
			errorContains: []string{
				"property reference '#property:invalid_property1' in rule '#tp:test-tp/rules/test-rule' not found in catalog",
				"property reference '#property:invalid_property2' in rule '#tp:test-tp/rules/test-rule' not found in catalog",
			},
		},
		{
			name: "mixed valid and invalid nested references",
			trackingPlans: []*catalog.TrackingPlanV1{
				{
					LocalID: "test-tp",
					Name:    "Test Tracking Plan",
					Rules: []*catalog.TPRuleV1{
						{
							LocalID: "test-rule",
							Type:    "event_rule",
							Event: "#/events/test-group/user_signup",
							AdditionalProperties: false,
							Properties: []*catalog.TPRulePropertyV1{
								{
									Property: "#property:user_profile",
									Required: true,
									Properties: []*catalog.TPRulePropertyV1{
										{
											Property: "#property:profile_name", // Valid
											Required: true,
										},
										{
											Property: "#/properties/invalid-format", // Invalid format
											Required: true,
										},
										{
											Property: "#property:notification_prefs",
											Required: true,
											Properties: []*catalog.TPRulePropertyV1{
												{
													Property: "#property:email_enabled", // Valid
													Required: true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expectedErrs: 1,
			errorContains: []string{
				"property reference '#/properties/invalid-format' has invalid format in rule '#tp:test-tp/rules/test-rule'. Should be '#property:<id>'",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dc := &catalog.DataCatalog{
				Properties:    testProperties,
				Events:        testEvents,
				TrackingPlans: tc.trackingPlans,
			}

			errs := validator.Validate(dc)

			assert.Len(t, errs, tc.expectedErrs, "Expected %d validation errors, got %d", tc.expectedErrs, len(errs))

			if tc.expectedErrs > 0 {
				for _, expectedError := range tc.errorContains {
					found := false
					for _, actualError := range errs {
						if strings.Contains(actualError.Error(), expectedError) {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected to find error containing: %s\nActual errors: %v", expectedError, errs)
				}
			}
		})
	}
}
