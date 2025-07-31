package validate

import (
	"fmt"
	"strings"
	"testing"

	catalog "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/stretchr/testify/assert"
)

func TestPropertyItemTypesCustomTypeReferences(t *testing.T) {
	validator := &RefValidator{}

	// Create a test custom type
	testCustomType := catalog.CustomType{
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
		properties    map[catalog.EntityGroup][]catalog.Property
		customTypes   map[catalog.EntityGroup][]catalog.CustomType
		expectedErrs  int
		errorContains []string
	}{
		{
			name: "valid custom type reference in property itemTypes",
			properties: map[catalog.EntityGroup][]catalog.Property{
				"test-group": {
					{
						LocalID:     "emailList",
						Name:        "Email List",
						Description: "List of user emails",
						Type:        "array",
						Config: map[string]interface{}{
							"itemTypes": []interface{}{"#/custom-types/email-types/EmailType"},
						},
					},
				},
			},
			customTypes: map[catalog.EntityGroup][]catalog.CustomType{
				"email-types": {testCustomType},
			},
			expectedErrs: 0,
		},
		{
			name: "invalid custom type reference format in property itemTypes",
			properties: map[catalog.EntityGroup][]catalog.Property{
				"test-group": {
					{
						LocalID:     "emailList",
						Name:        "Email List",
						Description: "List of user emails",
						Type:        "array",
						Config: map[string]interface{}{
							"itemTypes": []interface{}{"#/custom-types/email-types"}, // Missing type ID
						},
					},
				},
			},
			customTypes: map[catalog.EntityGroup][]catalog.CustomType{
				"email-types": {testCustomType},
			},
			expectedErrs:  1,
			errorContains: []string{"custom type reference in itemTypes at idx: 0 has invalid format"},
		},
		{
			name: "reference to non-existent custom type in property itemTypes",
			properties: map[catalog.EntityGroup][]catalog.Property{
				"test-group": {
					{
						LocalID:     "emailList",
						Name:        "Email List",
						Description: "List of user emails",
						Type:        "array",
						Config: map[string]interface{}{
							"itemTypes": []interface{}{"#/custom-types/email-types/NonExistentType"},
						},
					},
				},
			},
			customTypes: map[catalog.EntityGroup][]catalog.CustomType{
				"email-types": {testCustomType},
			},
			expectedErrs:  1,
			errorContains: []string{"custom type reference '#/custom-types/email-types/NonExistentType' in itemTypes at idx: 0 not found in catalog"},
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
		events        map[catalog.EntityGroup][]catalog.Event
		categories    map[catalog.EntityGroup][]catalog.Category
		expectedErrs  int
		errorContains []string
	}{
		{
			name: "valid category reference in event",
			events: map[catalog.EntityGroup][]catalog.Event{
				"app-events": {
					{
						LocalID:     "user_signup",
						Name:        "User Signup",
						Type:        "track",
						Description: "User signed up for the app",
						CategoryRef: stringPtr("#/categories/app-categories/user_actions"),
					},
				},
			},
			categories: map[catalog.EntityGroup][]catalog.Category{
				"app-categories": {testCategory},
			},
			expectedErrs: 0,
		},
		{
			name: "event without category reference should pass validation",
			events: map[catalog.EntityGroup][]catalog.Event{
				"app-events": {
					{
						LocalID:     "user_signup",
						Name:        "User Signup",
						Type:        "track",
						Description: "User signed up for the app",
						CategoryRef: nil,
					},
				},
			},
			categories:   map[catalog.EntityGroup][]catalog.Category{},
			expectedErrs: 0,
		},
		{
			name: "invalid category reference format in event",
			events: map[catalog.EntityGroup][]catalog.Event{
				"app-events": {
					{
						LocalID:     "user_signup",
						Name:        "User Signup",
						Type:        "track",
						Description: "User signed up for the app",
						CategoryRef: stringPtr("#/categories/app-categories"), // Missing category ID
					},
				},
			},
			categories: map[catalog.EntityGroup][]catalog.Category{
				"app-categories": {testCategory},
			},
			expectedErrs:  1,
			errorContains: []string{"the category field value is invalid. It should always be a reference and must follow the format '#/categories/<group>/<id>'"},
		},
		{
			name: "reference to non-existent category in event",
			events: map[catalog.EntityGroup][]catalog.Event{
				"app-events": {
					{
						LocalID:     "user_signup",
						Name:        "User Signup",
						Type:        "track",
						Description: "User signed up for the app",
						CategoryRef: stringPtr("#/categories/app-categories/non_existent_category"),
					},
				},
			},
			categories: map[catalog.EntityGroup][]catalog.Category{
				"app-categories": {testCategory},
			},
			expectedErrs:  1,
			errorContains: []string{"category reference '#/categories/app-categories/non_existent_category' not found in catalog"},
		},
		{
			name: "completely malformed category reference",
			events: map[catalog.EntityGroup][]catalog.Event{
				"app-events": {
					{
						LocalID:     "user_signup",
						Name:        "User Signup",
						Type:        "track",
						Description: "User signed up for the app",
						CategoryRef: stringPtr("user_actions"),
					},
				},
			},
			categories: map[catalog.EntityGroup][]catalog.Category{
				"app-categories": {testCategory},
			},
			expectedErrs:  1,
			errorContains: []string{"the category field value is invalid. It should always be a reference and must follow the format '#/categories/<group>/<id>'"},
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
	testProperties := map[catalog.EntityGroup][]catalog.Property{
		"test-group": {
			{LocalID: "page_name", Name: "Page Name", Type: "string"},
			{LocalID: "search_term", Name: "Search Term", Type: "string"},
			{LocalID: "product_id", Name: "Product ID", Type: "string"},
			{LocalID: "user_id", Name: "User ID", Type: "string"},
		},
	}

	testEvents := map[catalog.EntityGroup][]catalog.Event{
		"test-group": {
			{LocalID: "test-event", Name: "Test Event", Type: "track"},
		},
	}

	testCases := []struct {
		name          string
		trackingPlans map[catalog.EntityGroup]*catalog.TrackingPlan
		customTypes   map[catalog.EntityGroup][]catalog.CustomType
		errors        []ValidationError
	}{
		{
			name: "valid variants references in tracking plan",
			trackingPlans: map[catalog.EntityGroup]*catalog.TrackingPlan{
				"test-group": {
					LocalID: "test-tp",
					Name:    "Test Tracking Plan",
					Rules: []*catalog.TPRule{
						{
							LocalID: "test-rule",
							Type:    "event_rule",
							Event: &catalog.TPRuleEvent{
								Ref: "#/events/test-group/test-event",
							},
							Variants: catalog.Variants{
								{
									Type:          "discriminator",
									Discriminator: "#/properties/test-group/page_name",
									Cases: []catalog.VariantCase{
										{
											DisplayName: "Search Page",
											Match:       []any{"search", "search_bar"},
											Properties: []catalog.PropertyReference{
												{Ref: "#/properties/test-group/search_term", Required: true},
											},
										},
									},
									Default: []catalog.PropertyReference{
										{Ref: "#/properties/test-group/product_id", Required: true},
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
			customTypes: map[catalog.EntityGroup][]catalog.CustomType{
				"test-group": {
					{
						LocalID:     "TestType",
						Name:        "TestType",
						Description: "Test custom type with variants",
						Type:        "object",
						Variants: catalog.Variants{
							{
								Type:          "discriminator",
								Discriminator: "#/properties/test-group/user_id",
								Cases: []catalog.VariantCase{
									{
										DisplayName: "Admin User",
										Match:       []any{"admin", "superuser"},
										Properties: []catalog.PropertyReference{
											{Ref: "#/properties/test-group/product_id", Required: true},
										},
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
			name: "invalid property reference in variant case",
			trackingPlans: map[catalog.EntityGroup]*catalog.TrackingPlan{
				"test-group": {
					LocalID: "test-tp",
					Name:    "Test Tracking Plan",
					Rules: []*catalog.TPRule{
						{
							LocalID: "test-rule",
							Type:    "event_rule",
							Event: &catalog.TPRuleEvent{
								Ref: "#/events/test-group/test-event",
							},
							Variants: catalog.Variants{
								{
									Type:          "discriminator",
									Discriminator: "page_name",
									Cases: []catalog.VariantCase{
										{
											DisplayName: "Search Page",
											Match:       []any{"search"},
											Properties: []catalog.PropertyReference{
												{Ref: "#/properties/test-group/non_existent_property", Required: true},
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
					error:     fmt.Errorf("property reference '#/properties/test-group/non_existent_property' not found in catalog"),
					Reference: "#/tp/test-group/test-tp/rules/test-rule/variants[0]/cases[0]/properties[0]",
				},
			},
		},
		{
			name: "invalid property reference in variant default",
			trackingPlans: map[catalog.EntityGroup]*catalog.TrackingPlan{
				"test-group": {
					LocalID: "test-tp",
					Name:    "Test Tracking Plan",
					Rules: []*catalog.TPRule{
						{
							LocalID: "test-rule",
							Type:    "event_rule",
							Event: &catalog.TPRuleEvent{
								Ref: "#/events/test-group/test-event",
							},
							Variants: catalog.Variants{
								{
									Type:          "discriminator",
									Discriminator: "#/properties/test-group/page_name",
									Cases: []catalog.VariantCase{
										{
											DisplayName: "Search Page",
											Match:       []any{"search"},
											Properties: []catalog.PropertyReference{
												{Ref: "#/properties/test-group/search_term", Required: true},
											},
										},
									},
									Default: []catalog.PropertyReference{
										{Ref: "#/properties/test-group/non_existent_property", Required: true},
									},
								},
							},
						},
					},
				},
			},
			errors: []ValidationError{
				{
					error:     fmt.Errorf("default property reference '#/properties/test-group/non_existent_property' not found in catalog"),
					Reference: "#/tp/test-group/test-tp/rules/test-rule/variants[0]/default/properties[0]",
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
						assert.Failf(t, "variants_reference_validation_failures", "Expected to find error: %v with reference: %s in expected", actual, actual.Reference)
					}
				}
			}
		})
	}
}

func TestRecursiveReferenceValidation(t *testing.T) {
	validator := &RefValidator{}

	// Create test properties for nested reference validation
	testProperties := map[catalog.EntityGroup][]catalog.Property{
		"test-group": {
			{LocalID: "user_profile", Name: "User Profile", Type: "object"},
			{LocalID: "profile_name", Name: "Profile Name", Type: "string"},
			{LocalID: "profile_settings", Name: "Profile Settings", Type: "object"},
			{LocalID: "theme_preference", Name: "Theme Preference", Type: "string"},
			{LocalID: "notification_prefs", Name: "Notification Preferences", Type: "object"},
			{LocalID: "email_enabled", Name: "Email Enabled", Type: "boolean"},
		},
	}

	testEvents := map[catalog.EntityGroup][]catalog.Event{
		"test-group": {
			{LocalID: "user_signup", Name: "User Signup", Type: "track"},
		},
	}

	testCases := []struct {
		name          string
		trackingPlans map[catalog.EntityGroup]*catalog.TrackingPlan
		expectedErrs  int
		errorContains []string
	}{
		{
			name: "valid nested property references (2 levels)",
			trackingPlans: map[catalog.EntityGroup]*catalog.TrackingPlan{
				"test-group": {
					LocalID: "test-tp",
					Name:    "Test Tracking Plan",
					Rules: []*catalog.TPRule{
						{
							LocalID: "test-rule",
							Type:    "event_rule",
							Event: &catalog.TPRuleEvent{
								Ref: "#/events/test-group/user_signup",
							},
							Properties: []*catalog.TPRuleProperty{
								{
									Ref:      "#/properties/test-group/user_profile",
									Required: true,
									Properties: []*catalog.TPRuleProperty{
										{
											Ref:      "#/properties/test-group/profile_name",
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
			trackingPlans: map[catalog.EntityGroup]*catalog.TrackingPlan{
				"test-group": {
					LocalID: "test-tp",
					Name:    "Test Tracking Plan",
					Rules: []*catalog.TPRule{
						{
							LocalID: "test-rule",
							Type:    "event_rule",
							Event: &catalog.TPRuleEvent{
								Ref: "#/events/test-group/user_signup",
							},
							Properties: []*catalog.TPRuleProperty{
								{
									Ref:      "#/properties/test-group/user_profile",
									Required: true,
									Properties: []*catalog.TPRuleProperty{
										{
											Ref:      "#/properties/test-group/profile_settings",
											Required: true,
											Properties: []*catalog.TPRuleProperty{
												{
													Ref:      "#/properties/test-group/theme_preference",
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
			trackingPlans: map[catalog.EntityGroup]*catalog.TrackingPlan{
				"test-group": {
					LocalID: "test-tp",
					Name:    "Test Tracking Plan",
					Rules: []*catalog.TPRule{
						{
							LocalID: "test-rule",
							Type:    "event_rule",
							Event: &catalog.TPRuleEvent{
								Ref: "#/events/test-group/user_signup",
							},
							Properties: []*catalog.TPRuleProperty{
								{
									Ref:      "#/properties/test-group/user_profile",
									Required: true,
									Properties: []*catalog.TPRuleProperty{
										{
											Ref:      "#/properties/test-group/nonexistent_property",
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
			errorContains: []string{"property reference '#/properties/test-group/nonexistent_property' in rule '#/tp/test-group/test-tp/rules/test-rule' not found in catalog"},
		},
		{
			name: "invalid nested property reference format",
			trackingPlans: map[catalog.EntityGroup]*catalog.TrackingPlan{
				"test-group": {
					LocalID: "test-tp",
					Name:    "Test Tracking Plan",
					Rules: []*catalog.TPRule{
						{
							LocalID: "test-rule",
							Type:    "event_rule",
							Event: &catalog.TPRuleEvent{
								Ref: "#/events/test-group/user_signup",
							},
							Properties: []*catalog.TPRuleProperty{
								{
									Ref:      "#/properties/test-group/user_profile",
									Required: true,
									Properties: []*catalog.TPRuleProperty{
										{
											Ref:      "#/properties/test-group", // Invalid format - missing property ID
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
			errorContains: []string{"property reference '#/properties/test-group' has invalid format in rule '#/tp/test-group/test-tp/rules/test-rule'. Should be '#/properties/<group>/<id>'"},
		},
		{
			name: "multiple nested reference errors (different levels)",
			trackingPlans: map[catalog.EntityGroup]*catalog.TrackingPlan{
				"test-group": {
					LocalID: "test-tp",
					Name:    "Test Tracking Plan",
					Rules: []*catalog.TPRule{
						{
							LocalID: "test-rule",
							Type:    "event_rule",
							Event: &catalog.TPRuleEvent{
								Ref: "#/events/test-group/user_signup",
							},
							Properties: []*catalog.TPRuleProperty{
								{
									Ref:      "#/properties/test-group/user_profile",
									Required: true,
									Properties: []*catalog.TPRuleProperty{
										{
											Ref:      "#/properties/test-group/invalid_property1", // Non-existent at level 2
											Required: true,
										},
										{
											Ref:      "#/properties/test-group/profile_settings",
											Required: true,
											Properties: []*catalog.TPRuleProperty{
												{
													Ref:      "#/properties/test-group/invalid_property2", // Non-existent at level 3
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
				"property reference '#/properties/test-group/invalid_property1' in rule '#/tp/test-group/test-tp/rules/test-rule' not found in catalog",
				"property reference '#/properties/test-group/invalid_property2' in rule '#/tp/test-group/test-tp/rules/test-rule' not found in catalog",
			},
		},
		{
			name: "mixed valid and invalid nested references",
			trackingPlans: map[catalog.EntityGroup]*catalog.TrackingPlan{
				"test-group": {
					LocalID: "test-tp",
					Name:    "Test Tracking Plan",
					Rules: []*catalog.TPRule{
						{
							LocalID: "test-rule",
							Type:    "event_rule",
							Event: &catalog.TPRuleEvent{
								Ref: "#/events/test-group/user_signup",
							},
							Properties: []*catalog.TPRuleProperty{
								{
									Ref:      "#/properties/test-group/user_profile",
									Required: true,
									Properties: []*catalog.TPRuleProperty{
										{
											Ref:      "#/properties/test-group/profile_name", // Valid
											Required: true,
										},
										{
											Ref:      "#/properties/invalid-format", // Invalid format
											Required: true,
										},
										{
											Ref:      "#/properties/test-group/notification_prefs",
											Required: true,
											Properties: []*catalog.TPRuleProperty{
												{
													Ref:      "#/properties/test-group/email_enabled", // Valid
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
				"property reference '#/properties/invalid-format' has invalid format in rule '#/tp/test-group/test-tp/rules/test-rule'. Should be '#/properties/<group>/<id>'",
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
