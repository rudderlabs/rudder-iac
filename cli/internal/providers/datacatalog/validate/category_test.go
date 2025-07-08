package validate

import (
	"testing"

	catalog "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/stretchr/testify/assert"
)

func TestCategoryValidator_Validate(t *testing.T) {
	validator := &CategoryValidator{}

	testCases := []struct {
		name           string
		categories     map[catalog.EntityGroup][]catalog.Category
		expectedErrors int
		errorContains  []string
	}{
		{
			name: "valid categories",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "marketing",
						Name:    "Marketing Team",
					},
					{
						LocalID: "engineering",
						Name:    "Engineering_Team",
					},
					{
						LocalID: "sales",
						Name:    "Sales, Support - Customer Success",
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "category with missing ID",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "",
						Name:    "Marketing Team",
					},
				},
			},
			expectedErrors: 1,
			errorContains:  []string{"id and name fields on category are mandatory"},
		},
		{
			name: "category with missing name",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "marketing",
						Name:    "",
					},
				},
			},
			expectedErrors: 1,
			errorContains:  []string{"id and name fields on category are mandatory"},
		},
		{
			name: "category with missing both ID and name",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "",
						Name:    "",
					},
				},
			},
			expectedErrors: 1,
			errorContains:  []string{"id and name fields on category are mandatory"},
		},
		{
			name: "category with leading whitespace in name",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "marketing",
						Name:    " Marketing Team",
					},
				},
			},
			expectedErrors: 1,
			errorContains:  []string{"category name cannot have leading or trailing whitespace characters"},
		},
		{
			name: "category with trailing whitespace in name",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "marketing",
						Name:    "Marketing Team ",
					},
				},
			},
			expectedErrors: 1,
			errorContains:  []string{"category name cannot have leading or trailing whitespace characters"},
		},
		{
			name: "category with both leading and trailing whitespace in name",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "marketing",
						Name:    " Marketing Team ",
					},
				},
			},
			expectedErrors: 1,
			errorContains:  []string{"category name cannot have leading or trailing whitespace characters"},
		},
		{
			name: "category name too short (less than 3 characters)",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "marketing",
						Name:    "Ma",
					},
				},
			},
			expectedErrors: 1,
			errorContains:  []string{"category name must start with a letter"},
		},
		{
			name: "category name too long (more than 65 characters)",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "marketing",
						Name:    "This is a very long category name that exceeds the maximum allowed length of sixty-five characters and should fail validation",
					},
				},
			},
			expectedErrors: 1,
			errorContains:  []string{"category name must start with a letter"},
		},
		{
			name: "category name starting with number (invalid)",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "marketing",
						Name:    "123 Marketing",
					},
				},
			},
			expectedErrors: 1,
			errorContains:  []string{"category name must start with a letter"},
		},
		{
			name: "category name starting with special character (invalid)",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "marketing",
						Name:    "@Marketing",
					},
				},
			},
			expectedErrors: 1,
			errorContains:  []string{"category name must start with a letter"},
		},
		{
			name: "category name with valid special characters",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "category1",
						Name:    "Marketing Team, Sales - Support",
					},
					{
						LocalID: "category2",
						Name:    "Engineering-Product.Development",
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "category name with invalid special characters",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "marketing",
						Name:    "Marketing#Team",
					},
				},
			},
			expectedErrors: 1,
			errorContains:  []string{"category name must start with a letter"},
		},
		{
			name: "duplicate category names",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "marketing1",
						Name:    "Marketing Team",
					},
					{
						LocalID: "marketing2",
						Name:    "Marketing Team",
					},
				},
			},
			expectedErrors: 1,
			errorContains:  []string{"duplicate category name: Marketing Team"},
		},
		{
			name: "duplicate category IDs",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "marketing",
						Name:    "Marketing Team",
					},
					{
						LocalID: "marketing",
						Name:    "Sales Team",
					},
				},
			},
			expectedErrors: 1,
			errorContains:  []string{"duplicate category id: marketing"},
		},
		{
			name: "multiple validation errors",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "",
						Name:    "Invalid Name",
					},
					{
						LocalID: "valid",
						Name:    "123InvalidStart",
					},
					{
						LocalID: "duplicate",
						Name:    "Valid Name",
					},
					{
						LocalID: "duplicate",
						Name:    "Valid Name",
					},
				},
			},
			expectedErrors: 3,
			errorContains: []string{
				"id and name fields on category are mandatory",
				"category name must start with a letter",
				"duplicate category name: Valid Name",
			},
		},
		{
			name: "edge case: minimum valid length (3 characters)",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "min",
						Name:    "Min",
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "edge case: maximum valid length (65 characters)",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "max",
						Name:    "A very long category name that is exactly sixty five characters",
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "edge case: underscore starting name",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "underscore",
						Name:    "_Private Category",
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "edge case: uppercase starting name",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "uppercase",
						Name:    "MARKETING TEAM",
					},
				},
			},
			expectedErrors: 0,
		},
		{
			name: "edge case: lowercase starting name",
			categories: map[catalog.EntityGroup][]catalog.Category{
				"test-group": {
					{
						LocalID: "lowercase",
						Name:    "marketing team",
					},
				},
			},
			expectedErrors: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a minimal data catalog with just categories
			dc := &catalog.DataCatalog{
				Categories: tc.categories,
			}

			// Validate
			errors := validator.Validate(dc)

			// Check results
			if tc.expectedErrors == 0 {
				assert.Empty(t, errors, "Expected no validation errors")
			} else {
				assert.Len(t, errors, tc.expectedErrors, "Expected %d validation errors, got %d", tc.expectedErrors, len(errors))

				// Check that all expected error messages are present
				if len(tc.errorContains) > 0 {
					errorMessages := make([]string, len(errors))
					for i, err := range errors {
						errorMessages[i] = err.Error()
					}

					for _, expectedMsg := range tc.errorContains {
						found := false
						for _, errorMsg := range errorMessages {
							if contains(errorMsg, expectedMsg) {
								found = true
								break
							}
						}
						assert.True(t, found, "Expected to find error containing '%s' in errors: %v", expectedMsg, errorMessages)
					}
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > len(substr) && containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
