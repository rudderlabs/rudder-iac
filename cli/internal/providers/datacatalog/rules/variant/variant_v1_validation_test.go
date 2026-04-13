package variant

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateVariantSemanticV1_TypeCheck(t *testing.T) {
	t.Parallel()

	t.Run("valid discriminator type — string", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("signup_method", "Signup Method", "string"))

		variants := localcatalog.VariantsV1{
			{
				Type:          "discriminator",
				Discriminator: "#property:signup_method",
				Cases: []localcatalog.VariantCaseV1{
					{
						DisplayName: "Email",
						Match:       []any{"email"},
						Properties:  []localcatalog.PropertyReferenceV1{{Property: "#property:signup_method"}},
					},
				},
			},
		}

		results := ValidateVariantSemanticV1(variants, []string{"#property:signup_method"}, "/types/0", graph)
		assert.Empty(t, results, "string type should be valid for discriminator")
	})

	t.Run("valid discriminator type — integer", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("level", "Level", "integer"))

		variants := localcatalog.VariantsV1{
			{
				Type:          "discriminator",
				Discriminator: "#property:level",
				Cases: []localcatalog.VariantCaseV1{
					{
						DisplayName: "Basic",
						Match:       []any{1},
						Properties:  []localcatalog.PropertyReferenceV1{{Property: "#property:level"}},
					},
				},
			},
		}

		results := ValidateVariantSemanticV1(variants, []string{"#property:level"}, "/types/0", graph)
		assert.Empty(t, results, "integer type should be valid for discriminator")
	})

	t.Run("valid discriminator type — boolean", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("is_active", "Is Active", "boolean"))

		variants := localcatalog.VariantsV1{
			{
				Type:          "discriminator",
				Discriminator: "#property:is_active",
				Cases: []localcatalog.VariantCaseV1{
					{
						DisplayName: "Active",
						Match:       []any{true},
						Properties:  []localcatalog.PropertyReferenceV1{{Property: "#property:is_active"}},
					},
				},
			},
		}

		results := ValidateVariantSemanticV1(variants, []string{"#property:is_active"}, "/types/0", graph)
		assert.Empty(t, results, "boolean type should be valid for discriminator")
	})

	t.Run("invalid discriminator type — object", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("address", "Address", "object"))

		variants := localcatalog.VariantsV1{
			{
				Type:          "discriminator",
				Discriminator: "#property:address",
				Cases: []localcatalog.VariantCaseV1{
					{
						DisplayName: "Home",
						Match:       []any{"home"},
						Properties:  []localcatalog.PropertyReferenceV1{{Property: "#property:address"}},
					},
				},
			},
		}

		results := ValidateVariantSemanticV1(variants, []string{"#property:address"}, "/types/0", graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/types/0/variants/0/discriminator", results[0].Reference)
		assert.Contains(t, results[0].Message, "discriminator property type 'object' must contain one of: string, integer, boolean")
	})

	t.Run("invalid discriminator type — array", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("tags", "Tags", "array"))

		variants := localcatalog.VariantsV1{
			{
				Type:          "discriminator",
				Discriminator: "#property:tags",
				Cases: []localcatalog.VariantCaseV1{
					{
						DisplayName: "Tagged",
						Match:       []any{"tag"},
						Properties:  []localcatalog.PropertyReferenceV1{{Property: "#property:tags"}},
					},
				},
			},
		}

		results := ValidateVariantSemanticV1(variants, []string{"#property:tags"}, "/types/0", graph)

		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "discriminator property type 'array' must contain one of: string, integer, boolean")
	})

	t.Run("custom type ref — valid underlying type", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithCustomType(
			"payment_method", "Payment Method",
			resources.PropertyRef{URN: "custom-type:PaymentType", Property: "name"},
		))
		graph.AddResource(customTypeResourceWithType("PaymentType", "PaymentType", "string"))

		variants := localcatalog.VariantsV1{
			{
				Type:          "discriminator",
				Discriminator: "#property:payment_method",
				Cases: []localcatalog.VariantCaseV1{
					{
						DisplayName: "Credit",
						Match:       []any{"credit"},
						Properties:  []localcatalog.PropertyReferenceV1{{Property: "#property:payment_method"}},
					},
				},
			},
		}

		results := ValidateVariantSemanticV1(variants, []string{"#property:payment_method"}, "/types/0", graph)
		assert.Empty(t, results, "custom type ref with valid underlying type should be allowed")
	})

	t.Run("custom type ref — invalid underlying type", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithCustomType(
			"address_field", "Address Field",
			resources.PropertyRef{URN: "custom-type:Address", Property: "name"},
		))
		graph.AddResource(customTypeResourceWithType("Address", "Address", "object"))

		variants := localcatalog.VariantsV1{
			{
				Type:          "discriminator",
				Discriminator: "#property:address_field",
				Cases: []localcatalog.VariantCaseV1{
					{
						DisplayName: "Home",
						Match:       []any{"home"},
						Properties:  []localcatalog.PropertyReferenceV1{{Property: "#property:address_field"}},
					},
				},
			},
		}

		results := ValidateVariantSemanticV1(variants, []string{"#property:address_field"}, "/types/0", graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/types/0/variants/0/discriminator", results[0].Reference)
		assert.Contains(t, results[0].Message, "custom type")
		assert.Contains(t, results[0].Message, "object")
		assert.Contains(t, results[0].Message, "must be one of")
	})

	t.Run("discriminator not in graph — skip type check", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()

		variants := localcatalog.VariantsV1{
			{
				Type:          "discriminator",
				Discriminator: "#property:missing_prop",
				Cases: []localcatalog.VariantCaseV1{
					{
						DisplayName: "Case",
						Match:       []any{"x"},
						Properties:  []localcatalog.PropertyReferenceV1{{Property: "#property:missing_prop"}},
					},
				},
			},
		}

		results := ValidateVariantSemanticV1(variants, []string{"#property:missing_prop"}, "/types/0", graph)
		assert.Empty(t, results, "missing ref should be skipped for type check — reported by ValidateReferences")
	})
}

func TestValidateVariantSemanticV1_Ownership(t *testing.T) {
	t.Parallel()

	t.Run("discriminator exists in own properties", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("signup_method", "Signup Method", "string"))

		variants := localcatalog.VariantsV1{
			{
				Type:          "discriminator",
				Discriminator: "#property:signup_method",
				Cases: []localcatalog.VariantCaseV1{
					{
						DisplayName: "Email",
						Match:       []any{"email"},
						Properties:  []localcatalog.PropertyReferenceV1{{Property: "#property:signup_method"}},
					},
				},
			},
		}

		ownRefs := []string{"#property:email", "#property:signup_method"}
		results := ValidateVariantSemanticV1(variants, ownRefs, "/types/0", graph)
		assert.Empty(t, results, "discriminator found in own properties — no error")
	})

	t.Run("discriminator not in own properties", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("signup_method", "Signup Method", "string"))

		variants := localcatalog.VariantsV1{
			{
				Type:          "discriminator",
				Discriminator: "#property:signup_method",
				Cases: []localcatalog.VariantCaseV1{
					{
						DisplayName: "Email",
						Match:       []any{"email"},
						Properties:  []localcatalog.PropertyReferenceV1{{Property: "#property:signup_method"}},
					},
				},
			},
		}

		ownRefs := []string{"#property:email", "#property:phone"}
		results := ValidateVariantSemanticV1(variants, ownRefs, "/types/0", graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/types/0/variants/0/discriminator", results[0].Reference)
		assert.Contains(t, results[0].Message, "must reference a property defined in the parent's own properties")
	})

	t.Run("empty own properties", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("signup_method", "Signup Method", "string"))

		variants := localcatalog.VariantsV1{
			{
				Type:          "discriminator",
				Discriminator: "#property:signup_method",
				Cases: []localcatalog.VariantCaseV1{
					{
						DisplayName: "Email",
						Match:       []any{"email"},
						Properties:  []localcatalog.PropertyReferenceV1{{Property: "#property:signup_method"}},
					},
				},
			},
		}

		results := ValidateVariantSemanticV1(variants, nil, "/types/0", graph)

		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "must reference a property defined in the parent's own properties")
	})
}

func TestCheckDuplicatePropertyRefsV1(t *testing.T) {
	t.Parallel()

	t.Run("no duplicates", func(t *testing.T) {
		t.Parallel()

		props := []localcatalog.PropertyReferenceV1{
			{Property: "#property:email"},
			{Property: "#property:name"},
		}

		results := CheckDuplicatePropertyRefsV1(props, "/rules/0/variants/0/cases/0/properties")
		assert.Empty(t, results)
	})

	t.Run("duplicates reported at every occurrence", func(t *testing.T) {
		t.Parallel()

		props := []localcatalog.PropertyReferenceV1{
			{Property: "#property:email"},
			{Property: "#property:name"},
			{Property: "#property:email"},
		}

		results := CheckDuplicatePropertyRefsV1(props, "/rules/0/variants/0/cases/0/properties")
		require.Len(t, results, 2)
		assert.Equal(t, "/rules/0/variants/0/cases/0/properties/0/property", results[0].Reference)
		assert.Contains(t, results[0].Message, "duplicate property reference in tracking plan event rule")
		assert.Equal(t, "/rules/0/variants/0/cases/0/properties/2/property", results[1].Reference)
		assert.Contains(t, results[1].Message, "duplicate property reference in tracking plan event rule")
	})
}
