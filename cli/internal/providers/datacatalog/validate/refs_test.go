package validate

import (
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
