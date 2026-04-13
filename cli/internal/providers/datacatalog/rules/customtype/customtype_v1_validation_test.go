package customtype

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomTypeSpecSyntaxValidRule_V1ValidSpecs(t *testing.T) {
	t.Parallel()

	spec := localcatalog.CustomTypeSpecV1{
		Types: []localcatalog.CustomTypeV1{
			{
				LocalID:     "contact_info",
				Name:        "ContactInfo",
				Description: "Contact details for a profile",
				Type:        "object",
				Properties: []localcatalog.CustomTypePropertyV1{
					{Property: "#property:email", Required: true},
					{Property: "#property:signup_method", Required: true},
				},
				Variants: localcatalog.VariantsV1{
					{
						Type:          "discriminator",
						Discriminator: "#property:signup_method",
						Cases: []localcatalog.VariantCaseV1{
							{
								DisplayName: "Email signup",
								Match:       []any{"email"},
								Properties: []localcatalog.PropertyReferenceV1{
									{Property: "#property:email", Required: true},
								},
							},
						},
						Default: localcatalog.DefaultPropertiesV1{
							Properties: []localcatalog.PropertyReferenceV1{
								{Property: "#property:email"},
							},
						},
					},
				},
			},
		},
	}

	results := validateCustomTypeSpecV1(localcatalog.KindCustomTypes, specs.SpecVersionV1, nil, spec)
	assert.Empty(t, results)
}

func TestCustomTypeSpecSyntaxValidRule_V1InvalidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		spec         localcatalog.CustomTypeSpecV1
		expectedRefs []string
		expectedMsgs []string
	}{
		{
			name: "required custom type fields",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{{}},
			},
			expectedRefs: []string{"/types/0/id", "/types/0/name", "/types/0/type"},
			expectedMsgs: []string{"'id' is required", "'name' is required", "'type' is required"},
		},
		{
			name: "invalid property and variant references",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "contact_info",
						Name:    "ContactInfo",
						Type:    "object",
						Properties: []localcatalog.CustomTypePropertyV1{
							{Property: "invalid-property-ref"},
						},
						Variants: localcatalog.VariantsV1{
							{
								Type:          "wrong-type",
								Discriminator: "bad-discriminator",
								Cases: []localcatalog.VariantCaseV1{
									{
										DisplayName: "Case1",
										Match:       []any{"email"},
										Properties: []localcatalog.PropertyReferenceV1{
											{Property: "bad-case-property"},
										},
									},
								},
								Default: localcatalog.DefaultPropertiesV1{
									Properties: []localcatalog.PropertyReferenceV1{
										{Property: "bad-default-property"},
									},
								},
							},
						},
					},
				},
			},
			expectedRefs: []string{
				"/types/0/properties/0/property",
				"/types/0/variants/0/type",
				"/types/0/variants/0/discriminator",
				"/types/0/variants/0/cases/0/properties/0/property",
				"/types/0/variants/0/default/properties/0/property",
			},
			expectedMsgs: []string{
				"'property' is not valid: must be of pattern #property:<id>",
				"'type' must equal 'discriminator'",
				"'discriminator' is not valid: must be of pattern #property:<id>",
				"'property' is not valid: must be of pattern #property:<id>",
				"'property' is not valid: must be of pattern #property:<id>",
			},
		},
		{
			name: "variants only allowed for object types",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID: "status",
						Name:    "Status",
						Type:    "string",
						Variants: localcatalog.VariantsV1{
							{
								Type:          "discriminator",
								Discriminator: "#property:status_type",
								Cases: []localcatalog.VariantCaseV1{
									{
										DisplayName: "Case1",
										Match:       []any{"email"},
										Properties: []localcatalog.PropertyReferenceV1{
											{Property: "#property:status_value"},
										},
									},
								},
							},
						},
					},
				},
			},
			expectedRefs: []string{"/types/0/variants"},
			expectedMsgs: []string{"'variants' is not allowed unless 'type object'"},
		},
		{
			name: "both item_type and item_types set(not allowed)",
			spec: localcatalog.CustomTypeSpecV1{
				Types: []localcatalog.CustomTypeV1{
					{
						LocalID:   "ct1",
						Name:      "CT1",
						Type:      "array",
						ItemType:  "string",
						ItemTypes: []string{"number"},
					},
				},
			},
			expectedRefs: []string{"/types/0/item_type", "/types/0/item_types"},
			expectedMsgs: []string{"'item_type' and 'item_types' cannot be specified together", "'item_types' and 'item_type' cannot be specified together"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateCustomTypeSpecV1(localcatalog.KindCustomTypes, specs.SpecVersionV1, nil, tt.spec)
			assert.ElementsMatch(t, tt.expectedRefs, extractRefs(results))
			assert.ElementsMatch(t, tt.expectedMsgs, extractMsgs(results))
		})
	}
}

func TestCustomTypeSemanticValid_V1(t *testing.T) {
	t.Parallel()

	t.Run("valid references and discriminator", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("email", "Email", "string"))
		graph.AddResource(propertyResourceWithType("signup_method", "Signup Method", "string"))
		graph.AddResource(customTypeResource("contact_info", "ContactInfo"))

		spec := localcatalog.CustomTypeSpecV1{
			Types: []localcatalog.CustomTypeV1{
				{
					LocalID: "contact_info",
					Name:    "ContactInfo",
					Type:    "object",
					Properties: []localcatalog.CustomTypePropertyV1{
						{Property: "#property:email"},
						{Property: "#property:signup_method"},
					},
					Variants: localcatalog.VariantsV1{
						{
							Type:          "discriminator",
							Discriminator: "#property:signup_method",
							Cases: []localcatalog.VariantCaseV1{
								{
									DisplayName: "Email",
									Match:       []any{"email"},
									Properties: []localcatalog.PropertyReferenceV1{
										{Property: "#property:email"},
									},
								},
							},
							Default: localcatalog.DefaultPropertiesV1{
								Properties: []localcatalog.PropertyReferenceV1{
									{Property: "#property:email"},
								},
							},
						},
					},
				},
			},
		}

		results := validateCustomTypeSemanticV1(localcatalog.KindCustomTypes, specs.SpecVersionV1, nil, spec, graph)
		assert.Empty(t, results)
	})

	t.Run("missing refs, invalid discriminator, and duplicate name", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("email", "Email", "string"))
		graph.AddResource(propertyResourceWithType("address", "Address", "object"))
		graph.AddResource(customTypeResource("contact_info_v1", "ContactInfo"))
		graph.AddResource(customTypeResource("contact_info_v2", "ContactInfo"))

		spec := localcatalog.CustomTypeSpecV1{
			Types: []localcatalog.CustomTypeV1{
				{
					LocalID: "contact_info_v1",
					Name:    "ContactInfo",
					Type:    "object",
					Properties: []localcatalog.CustomTypePropertyV1{
						{Property: "#property:email"},
					},
					Variants: localcatalog.VariantsV1{
						{
							Type:          "discriminator",
							Discriminator: "#property:address",
							Cases: []localcatalog.VariantCaseV1{
								{
									DisplayName: "Home",
									Match:       []any{"home"},
									Properties: []localcatalog.PropertyReferenceV1{
										{Property: "#property:missing_case_property"},
									},
								},
							},
							Default: localcatalog.DefaultPropertiesV1{
								Properties: []localcatalog.PropertyReferenceV1{
									{Property: "#property:missing_default_property"},
								},
							},
						},
					},
				},
			},
		}

		results := validateCustomTypeSemanticV1(localcatalog.KindCustomTypes, specs.SpecVersionV1, nil, spec, graph)
		require.Len(t, results, 5)
		assert.ElementsMatch(t, []string{
			"/types/0/variants/0/discriminator",
			"/types/0/variants/0/cases/0/properties/0/property",
			"/types/0/variants/0/default/properties/0/property",
			"/types/0/variants/0/discriminator",
			"/types/0/name",
		}, extractRefs(results))
		assert.ElementsMatch(t, []string{
			"referenced property 'missing_case_property' not found in resource graph",
			"referenced property 'missing_default_property' not found in resource graph",
			"discriminator must reference a property defined in the parent's own properties",
			"discriminator property type 'object' must contain one of: string, integer, boolean",
			"duplicate name 'ContactInfo' within kind 'custom-types'",
		}, extractMsgs(results))
	})
}

func TestCustomTypeSemanticValid_V1_ItemTypeRefs(t *testing.T) {
	t.Parallel()

	t.Run("custom type ref found in item_type", func(t *testing.T) {
		t.Parallel()

		graph := funcs.GraphWith("Address", "custom-type")

		spec := localcatalog.CustomTypeSpecV1{
			Types: []localcatalog.CustomTypeV1{
				{
					LocalID:  "address_list",
					Name:     "AddressList",
					Type:     "array",
					ItemType: "#custom-type:Address",
				},
			},
		}

		results := validateCustomTypeSemanticV1(localcatalog.KindCustomTypes, specs.SpecVersionV1, nil, spec, graph)
		assert.Empty(t, results, "custom type ref exists in graph - no error")
	})

	t.Run("custom type ref not found in item_type", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()

		spec := localcatalog.CustomTypeSpecV1{
			Types: []localcatalog.CustomTypeV1{
				{
					LocalID:  "address_list",
					Name:     "AddressList",
					Type:     "array",
					ItemType: "#custom-type:Missing",
				},
			},
		}

		results := validateCustomTypeSemanticV1(localcatalog.KindCustomTypes, specs.SpecVersionV1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/types/0/item_type", results[0].Reference)
		assert.Contains(t, results[0].Message, "referenced custom-type 'Missing' not found")
	})

	t.Run("primitive item_type skipped", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()

		spec := localcatalog.CustomTypeSpecV1{
			Types: []localcatalog.CustomTypeV1{
				{
					LocalID:  "string_list",
					Name:     "StringList",
					Type:     "array",
					ItemType: "string",
				},
			},
		}

		results := validateCustomTypeSemanticV1(localcatalog.KindCustomTypes, specs.SpecVersionV1, nil, spec, graph)
		assert.Empty(t, results, "primitive item_type should not trigger ref lookup")
	})

	t.Run("no item_type set", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()

		spec := localcatalog.CustomTypeSpecV1{
			Types: []localcatalog.CustomTypeV1{
				{
					LocalID: "simple_type",
					Name:    "SimpleType",
					Type:    "string",
				},
			},
		}

		results := validateCustomTypeSemanticV1(localcatalog.KindCustomTypes, specs.SpecVersionV1, nil, spec, graph)
		assert.Empty(t, results, "no item_type should not trigger ref check")
	})

}

func TestCustomTypeSpecSyntaxValidRule_V1_ItemTypeValidation(t *testing.T) {
	t.Parallel()

	t.Run("valid item_type primitive", func(t *testing.T) {
		t.Parallel()

		spec := localcatalog.CustomTypeSpecV1{
			Types: []localcatalog.CustomTypeV1{
				{
					LocalID:  "tags",
					Name:     "Tags",
					Type:     "array",
					ItemType: "string",
				},
			},
		}

		results := validateCustomTypeSpecV1(localcatalog.KindCustomTypes, specs.SpecVersionV1, nil, spec)
		assert.Empty(t, results)
	})

	t.Run("valid item_type custom type ref", func(t *testing.T) {
		t.Parallel()

		spec := localcatalog.CustomTypeSpecV1{
			Types: []localcatalog.CustomTypeV1{
				{
					LocalID:  "address_list",
					Name:     "AddressList",
					Type:     "array",
					ItemType: "#custom-type:Address",
				},
			},
		}

		results := validateCustomTypeSpecV1(localcatalog.KindCustomTypes, specs.SpecVersionV1, nil, spec)
		assert.Empty(t, results)
	})

	t.Run("invalid item_type value", func(t *testing.T) {
		t.Parallel()

		spec := localcatalog.CustomTypeSpecV1{
			Types: []localcatalog.CustomTypeV1{
				{
					LocalID:  "bad_list",
					Name:     "BadList",
					Type:     "array",
					ItemType: "invalid_type",
				},
			},
		}

		results := validateCustomTypeSpecV1(localcatalog.KindCustomTypes, specs.SpecVersionV1, nil, spec)

		require.Len(t, results, 1)
		assert.Equal(t, "/types/0/item_type", results[0].Reference)
		assert.Contains(t, results[0].Message, "'item_type' is invalid")
	})

	t.Run("duplicate item_types values", func(t *testing.T) {
		t.Parallel()

		spec := localcatalog.CustomTypeSpecV1{
			Types: []localcatalog.CustomTypeV1{
				{
					LocalID:   "dup_list",
					Name:      "DupList",
					Type:      "array",
					ItemTypes: []string{"string", "string"},
				},
			},
		}

		results := validateCustomTypeSpecV1(localcatalog.KindCustomTypes, specs.SpecVersionV1, nil, spec)

		require.Len(t, results, 1)
		assert.Equal(t, "/types/0/item_types", results[0].Reference)
		assert.Contains(t, results[0].Message, "'item_types' is invalid: must be unique")
	})

	t.Run("mixed primitives and custom type refs", func(t *testing.T) {
		t.Parallel()

		spec := localcatalog.CustomTypeSpecV1{
			Types: []localcatalog.CustomTypeV1{
				{
					LocalID:   "mixed_list",
					Name:      "MixedList",
					Type:      "array",
					ItemTypes: []string{"#custom-type:Address", "string"},
				},
			},
		}

		results := validateCustomTypeSpecV1(localcatalog.KindCustomTypes, specs.SpecVersionV1, nil, spec)

		require.Len(t, results, 1)
		assert.Equal(t, "/types/0/item_types/0", results[0].Reference)
		assert.Contains(t, results[0].Message, "must be one of [string number integer boolean null array object]")
	})

	t.Run("custom type ref in item_types", func(t *testing.T) {
		t.Parallel()

		spec := localcatalog.CustomTypeSpecV1{
			Types: []localcatalog.CustomTypeV1{
				{
					LocalID:   "mixed_list",
					Name:      "MixedList",
					Type:      "array",
					ItemTypes: []string{"#custom-type:Address"},
				},
			},
		}

		results := validateCustomTypeSpecV1(localcatalog.KindCustomTypes, specs.SpecVersionV1, nil, spec)

		require.Len(t, results, 1)
		assert.Equal(t, "/types/0/item_types/0", results[0].Reference)
		assert.Contains(t, results[0].Message, "must be one of [string number integer boolean null array object]")
	})
}

func TestCustomTypeSemanticValid_V1_VariantDuplicateProperties(t *testing.T) {
	t.Parallel()

	t.Run("duplicate in variant case and default properties", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("email", "Email", "string"))
		graph.AddResource(propertyResourceWithType("name", "Name", "string"))
		graph.AddResource(propertyResourceWithType("method", "Method", "string"))
		graph.AddResource(customTypeResource("ct1", "CT1"))

		spec := localcatalog.CustomTypeSpecV1{
			Types: []localcatalog.CustomTypeV1{
				{
					LocalID: "ct1",
					Name:    "CT1",
					Type:    "object",
					Properties: []localcatalog.CustomTypePropertyV1{
						{Property: "#property:email"},
						{Property: "#property:method"},
					},
					Variants: localcatalog.VariantsV1{
						{
							Type:          "discriminator",
							Discriminator: "#property:method",
							Cases: []localcatalog.VariantCaseV1{
								{
									DisplayName: "Case 1",
									Match:       []any{"a"},
									Properties: []localcatalog.PropertyReferenceV1{
										{Property: "#property:email"},
										{Property: "#property:name"},
										{Property: "#property:email"},
									},
								},
							},
							Default: localcatalog.DefaultPropertiesV1{
								Properties: []localcatalog.PropertyReferenceV1{
									{Property: "#property:name"},
									{Property: "#property:name"},
								},
							},
						},
					},
				},
			},
		}

		results := validateCustomTypeSemanticV1(localcatalog.KindCustomTypes, specs.SpecVersionV1, nil, spec, graph)

		refs := extractRefs(results)
		msgs := extractMsgs(results)

		assert.Contains(t, refs, "/types/0/variants/0/cases/0/properties/0/property")
		assert.Contains(t, refs, "/types/0/variants/0/cases/0/properties/2/property")
		assert.Contains(t, refs, "/types/0/variants/0/default/properties/0/property")
		assert.Contains(t, refs, "/types/0/variants/0/default/properties/1/property")
		assert.Contains(t, msgs, "duplicate property reference in tracking plan event rule")
	})

	t.Run("no duplicates in variant properties", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("email", "Email", "string"))
		graph.AddResource(propertyResourceWithType("name", "Name", "string"))
		graph.AddResource(propertyResourceWithType("method", "Method", "string"))
		graph.AddResource(customTypeResource("ct1", "CT1"))

		spec := localcatalog.CustomTypeSpecV1{
			Types: []localcatalog.CustomTypeV1{
				{
					LocalID: "ct1",
					Name:    "CT1",
					Type:    "object",
					Properties: []localcatalog.CustomTypePropertyV1{
						{Property: "#property:email"},
						{Property: "#property:method"},
					},
					Variants: localcatalog.VariantsV1{
						{
							Type:          "discriminator",
							Discriminator: "#property:method",
							Cases: []localcatalog.VariantCaseV1{
								{
									DisplayName: "Case 1",
									Match:       []any{"a"},
									Properties: []localcatalog.PropertyReferenceV1{
										{Property: "#property:email"},
										{Property: "#property:name"},
									},
								},
							},
						},
					},
				},
			},
		}

		results := validateCustomTypeSemanticV1(localcatalog.KindCustomTypes, specs.SpecVersionV1, nil, spec, graph)
		for _, r := range results {
			assert.NotEqual(t, "duplicate property reference in tracking plan event rule", r.Message)
		}
	})
}
