package funcs

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// propertyResourceWithType creates a property resource with a specific type in its data.
func propertyResourceWithType(id, name, typ string) *resources.Resource {
	data := resources.ResourceData{"name": name, "type": typ}
	return resources.NewResource(id, "property", data, nil)
}

// propertyResourceWithCustomType creates a property resource with a PropertyRef type.
func propertyResourceWithCustomType(id, name string, ref resources.PropertyRef) *resources.Resource {
	data := resources.ResourceData{"name": name, "type": ref}
	return resources.NewResource(id, "property", data, nil)
}

// customTypeResourceWithType creates a custom-type resource with a specific type.
func customTypeResourceWithType(id, name, typ string) *resources.Resource {
	data := resources.ResourceData{"name": name, "type": typ}
	return resources.NewResource(id, "custom-type", data, nil)
}

func TestValidateVariantDiscriminators_TypeCheck(t *testing.T) {
	t.Parallel()

	t.Run("valid discriminator type — string", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("signup_method", "Signup Method", "string"))

		variants := localcatalog.Variants{
			{
				Type:          "discriminator",
				Discriminator: "#property:signup_method",
				Cases: []localcatalog.VariantCase{
					{DisplayName: "Email", Properties: []localcatalog.PropertyReference{{Ref: "#property:signup_method"}}},
				},
			},
		}

		results := ValidateVariantDiscriminators(variants, []string{"#property:signup_method"}, "/types/0", graph)
		assert.Empty(t, results, "string type should be valid for discriminator")
	})

	t.Run("valid discriminator type — integer", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("level", "Level", "integer"))

		variants := localcatalog.Variants{
			{
				Type:          "discriminator",
				Discriminator: "#property:level",
				Cases: []localcatalog.VariantCase{
					{DisplayName: "Basic", Properties: []localcatalog.PropertyReference{{Ref: "#property:level"}}},
				},
			},
		}

		results := ValidateVariantDiscriminators(variants, []string{"#property:level"}, "/types/0", graph)
		assert.Empty(t, results, "integer type should be valid for discriminator")
	})

	t.Run("valid discriminator type — boolean", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("is_active", "Is Active", "boolean"))

		variants := localcatalog.Variants{
			{
				Type:          "discriminator",
				Discriminator: "#property:is_active",
				Cases: []localcatalog.VariantCase{
					{DisplayName: "Active", Properties: []localcatalog.PropertyReference{{Ref: "#property:is_active"}}},
				},
			},
		}

		results := ValidateVariantDiscriminators(variants, []string{"#property:is_active"}, "/types/0", graph)
		assert.Empty(t, results, "boolean type should be valid for discriminator")
	})

	t.Run("invalid discriminator type — object", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("address", "Address", "object"))

		variants := localcatalog.Variants{
			{
				Type:          "discriminator",
				Discriminator: "#property:address",
				Cases: []localcatalog.VariantCase{
					{DisplayName: "Home", Properties: []localcatalog.PropertyReference{{Ref: "#property:address"}}},
				},
			},
		}

		results := ValidateVariantDiscriminators(variants, []string{"#property:address"}, "/types/0", graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/types/0/variants/0/discriminator", results[0].Reference)
		assert.Contains(t, results[0].Message, "invalid")
		assert.Contains(t, results[0].Message, "must be one of")
	})

	t.Run("invalid discriminator type — array", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("tags", "Tags", "array"))

		variants := localcatalog.Variants{
			{
				Type:          "discriminator",
				Discriminator: "#property:tags",
				Cases: []localcatalog.VariantCase{
					{DisplayName: "Tagged", Properties: []localcatalog.PropertyReference{{Ref: "#property:tags"}}},
				},
			},
		}

		results := ValidateVariantDiscriminators(variants, []string{"#property:tags"}, "/types/0", graph)

		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "invalid")
	})

	t.Run("custom type ref — valid underlying type", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithCustomType(
			"payment_method", "Payment Method",
			resources.PropertyRef{URN: "custom-type:PaymentType", Property: "name"},
		))
		graph.AddResource(customTypeResourceWithType("PaymentType", "PaymentType", "string"))

		variants := localcatalog.Variants{
			{
				Type:          "discriminator",
				Discriminator: "#property:payment_method",
				Cases: []localcatalog.VariantCase{
					{DisplayName: "Credit", Properties: []localcatalog.PropertyReference{{Ref: "#property:payment_method"}}},
				},
			},
		}

		results := ValidateVariantDiscriminators(variants, []string{"#property:payment_method"}, "/types/0", graph)
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

		variants := localcatalog.Variants{
			{
				Type:          "discriminator",
				Discriminator: "#property:address_field",
				Cases: []localcatalog.VariantCase{
					{DisplayName: "Home", Properties: []localcatalog.PropertyReference{{Ref: "#property:address_field"}}},
				},
			},
		}

		results := ValidateVariantDiscriminators(variants, []string{"#property:address_field"}, "/types/0", graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/types/0/variants/0/discriminator", results[0].Reference)
		assert.Contains(t, results[0].Message, "custom type")
		assert.Contains(t, results[0].Message, "object")
		assert.Contains(t, results[0].Message, "must be one of")
	})

	t.Run("custom type ref — custom type not in graph", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithCustomType(
			"payment_method", "Payment Method",
			resources.PropertyRef{URN: "custom-type:MissingType", Property: "name"},
		))
		// No custom type resource added to graph

		variants := localcatalog.Variants{
			{
				Type:          "discriminator",
				Discriminator: "#property:payment_method",
				Cases: []localcatalog.VariantCase{
					{DisplayName: "Credit", Properties: []localcatalog.PropertyReference{{Ref: "#property:payment_method"}}},
				},
			},
		}

		results := ValidateVariantDiscriminators(variants, []string{"#property:payment_method"}, "/types/0", graph)
		assert.Empty(t, results, "custom type not in graph should be skipped gracefully")
	})

	t.Run("discriminator not in graph — skip type check", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()

		variants := localcatalog.Variants{
			{
				Type:          "discriminator",
				Discriminator: "#property:missing_prop",
				Cases: []localcatalog.VariantCase{
					{DisplayName: "Case", Properties: []localcatalog.PropertyReference{{Ref: "#property:missing_prop"}}},
				},
			},
		}

		// Ownership error expected, but no type check error (ref resolution handled by ValidateReferences)
		results := ValidateVariantDiscriminators(variants, []string{"#property:missing_prop"}, "/types/0", graph)
		assert.Empty(t, results, "missing ref should be skipped for type check — reported by ValidateReferences")
	})
}

func TestValidateVariantDiscriminators_Ownership(t *testing.T) {
	t.Parallel()

	t.Run("discriminator exists in own properties", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("signup_method", "Signup Method", "string"))

		variants := localcatalog.Variants{
			{
				Type:          "discriminator",
				Discriminator: "#property:signup_method",
				Cases: []localcatalog.VariantCase{
					{DisplayName: "Email", Properties: []localcatalog.PropertyReference{{Ref: "#property:signup_method"}}},
				},
			},
		}

		ownRefs := []string{"#property:email", "#property:signup_method"}
		results := ValidateVariantDiscriminators(variants, ownRefs, "/types/0", graph)
		assert.Empty(t, results, "discriminator found in own properties — no error")
	})

	t.Run("discriminator not in own properties", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("signup_method", "Signup Method", "string"))

		variants := localcatalog.Variants{
			{
				Type:          "discriminator",
				Discriminator: "#property:signup_method",
				Cases: []localcatalog.VariantCase{
					{DisplayName: "Email", Properties: []localcatalog.PropertyReference{{Ref: "#property:signup_method"}}},
				},
			},
		}

		// ownRefs does NOT contain signup_method
		ownRefs := []string{"#property:email", "#property:phone"}
		results := ValidateVariantDiscriminators(variants, ownRefs, "/types/0", graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/types/0/variants/0/discriminator", results[0].Reference)
		assert.Contains(t, results[0].Message, "must reference a property defined in the parent's own properties")
	})

	t.Run("empty own properties", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResourceWithType("signup_method", "Signup Method", "string"))

		variants := localcatalog.Variants{
			{
				Type:          "discriminator",
				Discriminator: "#property:signup_method",
				Cases: []localcatalog.VariantCase{
					{DisplayName: "Email", Properties: []localcatalog.PropertyReference{{Ref: "#property:signup_method"}}},
				},
			},
		}

		results := ValidateVariantDiscriminators(variants, nil, "/types/0", graph)

		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "must reference a property defined in the parent's own properties")
	})
}
