package customtype

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
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
					{Property: "#properties:email", Required: true},
					{Property: "#properties:signup_method", Required: true},
				},
				Variants: localcatalog.VariantsV1{
					{
						Type:          "discriminator",
						Discriminator: "#properties:signup_method",
						Cases: []localcatalog.VariantCaseV1{
							{
								DisplayName: "Email signup",
								Match:       []any{"email"},
								Properties: []localcatalog.PropertyReferenceV1{
									{Property: "#properties:email", Required: true},
								},
							},
						},
						Default: localcatalog.DefaultPropertiesV1{
							Properties: []localcatalog.PropertyReferenceV1{
								{Property: "#properties:email"},
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
				"'property' is not valid: must be of pattern #properties:<id>",
				"'type' must equal 'discriminator'",
				"'discriminator' is not valid: must be of pattern #properties:<id>",
				"'property' is not valid: must be of pattern #properties:<id>",
				"'property' is not valid: must be of pattern #properties:<id>",
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
								Discriminator: "#properties:status_type",
								Cases: []localcatalog.VariantCaseV1{
									{
										DisplayName: "Case1",
										Match:       []any{"email"},
										Properties: []localcatalog.PropertyReferenceV1{
											{Property: "#properties:status_value"},
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
						{Property: "#properties:email"},
						{Property: "#properties:signup_method"},
					},
					Variants: localcatalog.VariantsV1{
						{
							Type:          "discriminator",
							Discriminator: "#properties:signup_method",
							Cases: []localcatalog.VariantCaseV1{
								{
									DisplayName: "Email",
									Match:       []any{"email"},
									Properties: []localcatalog.PropertyReferenceV1{
										{Property: "#properties:email"},
									},
								},
							},
							Default: localcatalog.DefaultPropertiesV1{
								Properties: []localcatalog.PropertyReferenceV1{
									{Property: "#properties:email"},
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
						{Property: "#properties:email"},
					},
					Variants: localcatalog.VariantsV1{
						{
							Type:          "discriminator",
							Discriminator: "#properties:address",
							Cases: []localcatalog.VariantCaseV1{
								{
									DisplayName: "Home",
									Match:       []any{"home"},
									Properties: []localcatalog.PropertyReferenceV1{
										{Property: "#properties:missing_case_property"},
									},
								},
							},
							Default: localcatalog.DefaultPropertiesV1{
								Properties: []localcatalog.PropertyReferenceV1{
									{Property: "#properties:missing_default_property"},
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

func TestCustomTypeConfigValidRule_V1(t *testing.T) {
	t.Parallel()

	t.Run("object config is rejected", func(t *testing.T) {
		t.Parallel()

		spec := localcatalog.CustomTypeSpecV1{
			Types: []localcatalog.CustomTypeV1{
				{
					LocalID: "contact_info",
					Name:    "ContactInfo",
					Type:    "object",
					Config:  map[string]any{"anything": "value"},
				},
			},
		}

		results := validateCustomTypeConfigV1(localcatalog.KindCustomTypes, specs.SpecVersionV1, nil, spec)
		require.Len(t, results, 1)
		assert.Equal(t, "/types/0/config", results[0].Reference)
		assert.Equal(t, "config is not allowed for the specified type(s)", results[0].Message)
	})

	t.Run("snake case config stays out of scope", func(t *testing.T) {
		t.Parallel()

		spec := localcatalog.CustomTypeSpecV1{
			Types: []localcatalog.CustomTypeV1{
				{
					LocalID: "status",
					Name:    "Status",
					Type:    "string",
					Config: map[string]any{
						"min_length": 1,
						"max_length": 32,
					},
				},
			},
		}

		results := validateCustomTypeConfigV1(localcatalog.KindCustomTypes, specs.SpecVersionV1, nil, spec)
		assert.Empty(t, results)
	})
}
