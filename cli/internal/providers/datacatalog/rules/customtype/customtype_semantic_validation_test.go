package customtype

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
)

// customTypeResource creates a custom-type resource with name in its data.
func customTypeResource(id, name string) *resources.Resource {
	data := resources.ResourceData{"name": name}
	return resources.NewResource(id, "custom-type", data, nil)
}

// propertyResourceWithType creates a property resource with a specific type.
func propertyResourceWithType(id, name, typ string) *resources.Resource {
	data := resources.ResourceData{"name": name, "type": typ}
	return resources.NewResource(id, "property", data, nil)
}

func TestCustomTypeSemanticValid_ReferenceResolution(t *testing.T) {
	t.Parallel()

	t.Run("all refs found", func(t *testing.T) {
		t.Parallel()

		graph := funcs.GraphWith(
			"email", "property",
			"phone", "property",
			"signup_method", "property",
		)

		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID: "contact_info",
					Name:    "Contact Info",
					Type:    "object",
					Properties: []localcatalog.CustomTypeProperty{
						{Ref: "#property:email"},
						{Ref: "#property:phone"},
					},
					Variants: localcatalog.Variants{
						{
							Type:          "discriminator",
							Discriminator: "#property:signup_method",
							Cases: []localcatalog.VariantCase{
								{
									DisplayName: "Email",
									Properties: []localcatalog.PropertyReference{
										{Ref: "#property:email"},
									},
								},
							},
						},
					},
				},
			},
		}

		results := funcs.ValidateReferences(spec, graph)
		assert.Empty(t, results, "all refs exist in graph — no errors expected")
	})

	t.Run("property ref not found", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()

		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID: "contact_info",
					Name:    "Contact Info",
					Type:    "object",
					Properties: []localcatalog.CustomTypeProperty{
						{Ref: "#property:missing_prop"},
					},
				},
			},
		}

		results := funcs.ValidateReferences(spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/types/0/properties/0/$ref", results[0].Reference)
		assert.Contains(t, results[0].Message, "referenced property 'missing_prop' not found")
	})

	t.Run("variant discriminator not found", func(t *testing.T) {
		t.Parallel()

		graph := funcs.GraphWith("email", "property")

		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID: "contact_info",
					Name:    "Contact Info",
					Type:    "object",
					Properties: []localcatalog.CustomTypeProperty{
						{Ref: "#property:email"},
					},
					Variants: localcatalog.Variants{
						{
							Type:          "discriminator",
							Discriminator: "#property:missing_discriminator",
							Cases: []localcatalog.VariantCase{
								{
									DisplayName: "Case 1",
									Properties: []localcatalog.PropertyReference{
										{Ref: "#property:email"},
									},
								},
							},
						},
					},
				},
			},
		}

		results := funcs.ValidateReferences(spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/types/0/variants/0/discriminator", results[0].Reference)
		assert.Contains(t, results[0].Message, "referenced property 'missing_discriminator' not found")
	})

	t.Run("variant case property not found", func(t *testing.T) {
		t.Parallel()

		graph := funcs.GraphWith("signup_method", "property")

		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID: "contact_info",
					Name:    "Contact Info",
					Type:    "object",
					Variants: localcatalog.Variants{
						{
							Type:          "discriminator",
							Discriminator: "#property:signup_method",
							Cases: []localcatalog.VariantCase{
								{
									DisplayName: "Email",
									Properties: []localcatalog.PropertyReference{
										{Ref: "#property:missing_case_prop"},
									},
								},
							},
						},
					},
				},
			},
		}

		results := funcs.ValidateReferences(spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/types/0/variants/0/cases/0/properties/0/$ref", results[0].Reference)
		assert.Contains(t, results[0].Message, "referenced property 'missing_case_prop' not found")
	})

	t.Run("variant default property not found", func(t *testing.T) {
		t.Parallel()

		graph := funcs.GraphWith("signup_method", "property", "email", "property")

		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID: "contact_info",
					Name:    "Contact Info",
					Type:    "object",
					Variants: localcatalog.Variants{
						{
							Type:          "discriminator",
							Discriminator: "#property:signup_method",
							Cases: []localcatalog.VariantCase{
								{
									DisplayName: "Email",
									Properties: []localcatalog.PropertyReference{
										{Ref: "#property:email"},
									},
								},
							},
							Default: []localcatalog.PropertyReference{
								{Ref: "#property:missing_default_prop"},
							},
						},
					},
				},
			},
		}

		results := funcs.ValidateReferences(spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/types/0/variants/0/default/0/$ref", results[0].Reference)
		assert.Contains(t, results[0].Message, "referenced property 'missing_default_prop' not found")
	})

	t.Run("no variants — object type", func(t *testing.T) {
		t.Parallel()

		graph := funcs.GraphWith("email", "property")

		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID: "simple_type",
					Name:    "Simple Type",
					Type:    "object",
					Properties: []localcatalog.CustomTypeProperty{
						{Ref: "#property:email"},
					},
				},
			},
		}

		results := funcs.ValidateReferences(spec, graph)
		assert.Empty(t, results)
	})
}

func TestCustomTypeSemanticValid_Uniqueness(t *testing.T) {
	t.Parallel()

	t.Run("no duplicate — unique custom types", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(customTypeResource("ContactInfo", "ContactInfo"))
		graph.AddResource(customTypeResource("PaymentType", "PaymentType"))

		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{LocalID: "ContactInfo", Name: "ContactInfo", Type: "object"},
			},
		}

		results := validateCustomTypeSemantic(localcatalog.KindCustomTypes, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "unique custom type names should not trigger errors")
	})

	t.Run("duplicate detected", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(customTypeResource("ContactInfo_v1", "ContactInfo"))
		graph.AddResource(customTypeResource("ContactInfo_v2", "ContactInfo"))

		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{LocalID: "ContactInfo_v1", Name: "ContactInfo", Type: "object"},
			},
		}

		results := validateCustomTypeSemantic(localcatalog.KindCustomTypes, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/types/0/name", results[0].Reference)
		assert.Contains(t, results[0].Message, "not unique across the catalog")
	})

	t.Run("single in graph — no false positive", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(customTypeResource("ContactInfo", "ContactInfo"))

		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{LocalID: "ContactInfo", Name: "ContactInfo", Type: "object"},
			},
		}

		results := validateCustomTypeSemantic(localcatalog.KindCustomTypes, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "single custom type should not be flagged")
	})

	t.Run("mixed unique and duplicate — only duplicate flagged", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(customTypeResource("ContactInfo_v1", "ContactInfo"))
		graph.AddResource(customTypeResource("ContactInfo_v2", "ContactInfo"))
		graph.AddResource(customTypeResource("PaymentType", "PaymentType"))

		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{LocalID: "ContactInfo_v1", Name: "ContactInfo", Type: "object"},
				{LocalID: "PaymentType", Name: "PaymentType", Type: "string"},
			},
		}

		results := validateCustomTypeSemantic(localcatalog.KindCustomTypes, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/types/0/name", results[0].Reference)
		assert.Contains(t, results[0].Message, "custom type with name 'ContactInfo' is not unique")
	})
}

func TestCustomTypeSemanticValid_ConfigItemTypes(t *testing.T) {
	t.Parallel()

	t.Run("custom type ref found in itemTypes", func(t *testing.T) {
		t.Parallel()

		graph := funcs.GraphWith("Address", "custom-type")

		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID: "AddressList",
					Name:    "AddressList",
					Type:    "array",
					Config:  map[string]any{"itemTypes": []any{"#custom-type:Address"}},
				},
			},
		}

		results := validateCustomTypeSemantic(localcatalog.KindCustomTypes, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "custom type ref exists in graph — no error")
	})

	t.Run("custom type ref not found", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()

		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID: "AddressList",
					Name:    "AddressList",
					Type:    "array",
					Config:  map[string]any{"itemTypes": []any{"#custom-type:Missing"}},
				},
			},
		}

		results := validateCustomTypeSemantic(localcatalog.KindCustomTypes, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/types/0/config/itemTypes/0", results[0].Reference)
		assert.Contains(t, results[0].Message, "referenced custom-type 'Missing' not found")
	})

	t.Run("primitive types skipped", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()

		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID: "StringList",
					Name:    "StringList",
					Type:    "array",
					Config:  map[string]any{"itemTypes": []any{"string"}},
				},
			},
		}

		results := validateCustomTypeSemantic(localcatalog.KindCustomTypes, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "primitive types should not trigger ref lookup")
	})

	t.Run("mixed primitives and custom type refs", func(t *testing.T) {
		t.Parallel()

		graph := funcs.GraphWith("Address", "custom-type")

		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID: "MixedList",
					Name:    "MixedList",
					Type:    "array",
					Config:  map[string]any{"itemTypes": []any{"string", "#custom-type:Address", "#custom-type:Missing"}},
				},
			},
		}

		results := validateCustomTypeSemantic(localcatalog.KindCustomTypes, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/types/0/config/itemTypes/2", results[0].Reference)
		assert.Contains(t, results[0].Message, "referenced custom-type 'Missing' not found")
	})

	t.Run("no itemTypes in config", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()

		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID: "StringType",
					Name:    "StringType",
					Type:    "string",
					Config:  map[string]any{"min_length": 1},
				},
			},
		}

		results := validateCustomTypeSemantic(localcatalog.KindCustomTypes, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "config without itemTypes should not trigger ref check")
	})

	t.Run("nil config", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()

		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID: "SimpleType",
					Name:    "SimpleType",
					Type:    "string",
				},
			},
		}

		results := validateCustomTypeSemantic(localcatalog.KindCustomTypes, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "nil config should not trigger ref check")
	})
}

func TestCustomTypeSemanticValid_VariantDiscriminator(t *testing.T) {
	t.Parallel()

	t.Run("valid discriminator type — string", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("email", "Email", "string"))
		graph.AddResource(propertyResourceWithType("signup_method", "Signup Method", "string"))

		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID: "contact_info",
					Name:    "ContactInfo",
					Type:    "object",
					Properties: []localcatalog.CustomTypeProperty{
						{Ref: "#property:email"},
						{Ref: "#property:signup_method"},
					},
					Variants: localcatalog.Variants{
						{
							Type:          "discriminator",
							Discriminator: "#property:signup_method",
							Cases: []localcatalog.VariantCase{
								{
									DisplayName: "Email",
									Properties:  []localcatalog.PropertyReference{{Ref: "#property:email"}},
								},
							},
						},
					},
				},
			},
		}

		results := validateCustomTypeSemantic(localcatalog.KindCustomTypes, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "string discriminator in own properties — no errors expected")
	})

	t.Run("valid discriminator type — boolean", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("email", "Email", "string"))
		graph.AddResource(propertyResourceWithType("is_active", "Is Active", "boolean"))

		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID: "contact_info",
					Name:    "ContactInfo",
					Type:    "object",
					Properties: []localcatalog.CustomTypeProperty{
						{Ref: "#property:email"},
						{Ref: "#property:is_active"},
					},
					Variants: localcatalog.Variants{
						{
							Type:          "discriminator",
							Discriminator: "#property:is_active",
							Cases: []localcatalog.VariantCase{
								{
									DisplayName: "Active",
									Properties:  []localcatalog.PropertyReference{{Ref: "#property:email"}},
								},
							},
						},
					},
				},
			},
		}

		results := validateCustomTypeSemantic(localcatalog.KindCustomTypes, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "boolean discriminator in own properties — no errors expected")
	})

	t.Run("invalid discriminator type — object", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("email", "Email", "string"))
		graph.AddResource(propertyResourceWithType("address", "Address", "object"))

		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID: "contact_info",
					Name:    "ContactInfo",
					Type:    "object",
					Properties: []localcatalog.CustomTypeProperty{
						{Ref: "#property:email"},
						{Ref: "#property:address"},
					},
					Variants: localcatalog.Variants{
						{
							Type:          "discriminator",
							Discriminator: "#property:address",
							Cases: []localcatalog.VariantCase{
								{
									DisplayName: "Home",
									Properties:  []localcatalog.PropertyReference{{Ref: "#property:email"}},
								},
							},
						},
					},
				},
			},
		}

		results := validateCustomTypeSemantic(localcatalog.KindCustomTypes, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/types/0/variants/0/discriminator", results[0].Reference)
		assert.Contains(t, results[0].Message, "invalid")
		assert.Contains(t, results[0].Message, "must be one of")
	})

	t.Run("custom type ref discriminator — allowed", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource("email", "property", resources.ResourceData{
			"name": "Email", "type": "string",
		}, nil))
		graph.AddResource(resources.NewResource("payment_method", "property", resources.ResourceData{
			"name": "Payment Method",
			"type": resources.PropertyRef{URN: "custom-type:PaymentType", Property: "name"},
		}, nil))

		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID: "order",
					Name:    "Order",
					Type:    "object",
					Properties: []localcatalog.CustomTypeProperty{
						{Ref: "#property:email"},
						{Ref: "#property:payment_method"},
					},
					Variants: localcatalog.Variants{
						{
							Type:          "discriminator",
							Discriminator: "#property:payment_method",
							Cases: []localcatalog.VariantCase{
								{
									DisplayName: "Credit",
									Properties:  []localcatalog.PropertyReference{{Ref: "#property:email"}},
								},
							},
						},
					},
				},
			},
		}

		results := validateCustomTypeSemantic(localcatalog.KindCustomTypes, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "custom type ref discriminator should be allowed")
	})

	t.Run("discriminator not in own properties", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("email", "Email", "string"))
		graph.AddResource(propertyResourceWithType("external_prop", "External Prop", "string"))

		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID: "contact_info",
					Name:    "ContactInfo",
					Type:    "object",
					Properties: []localcatalog.CustomTypeProperty{
						{Ref: "#property:email"},
					},
					Variants: localcatalog.Variants{
						{
							Type:          "discriminator",
							Discriminator: "#property:external_prop",
							Cases: []localcatalog.VariantCase{
								{
									DisplayName: "Case1",
									Properties:  []localcatalog.PropertyReference{{Ref: "#property:email"}},
								},
							},
						},
					},
				},
			},
		}

		results := validateCustomTypeSemantic(localcatalog.KindCustomTypes, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/types/0/variants/0/discriminator", results[0].Reference)
		assert.Contains(t, results[0].Message, "must reference a property defined in the parent's own properties")
	})

	t.Run("no variants — no variant errors", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("email", "Email", "string"))

		spec := localcatalog.CustomTypeSpec{
			Types: []localcatalog.CustomType{
				{
					LocalID: "simple_type",
					Name:    "SimpleType",
					Type:    "object",
					Properties: []localcatalog.CustomTypeProperty{
						{Ref: "#property:email"},
					},
				},
			},
		}

		results := validateCustomTypeSemantic(localcatalog.KindCustomTypes, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "no variants means no variant errors")
	})
}
