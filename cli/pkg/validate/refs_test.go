package validate

import (
	"testing"

	catalog "github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
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
