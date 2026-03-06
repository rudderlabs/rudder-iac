package property

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPropertySemanticValidRule_MatchPatterns(t *testing.T) {
	t.Parallel()

	rule := NewPropertySemanticValidRule()

	expectedPatterns := append(
		prules.LegacyVersionPatterns(localcatalog.KindProperties),
		vrules.MatchKindVersion(localcatalog.KindProperties, specs.SpecVersionV1),
	)
	assert.Equal(t, expectedPatterns, rule.AppliesTo())
}

func TestPropertySemanticValidV1_ReferenceType(t *testing.T) {
	t.Parallel()

	t.Run("custom type ref found in type", func(t *testing.T) {
		t.Parallel()

		graph := funcs.GraphWith("Address", "custom-type")
		spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{LocalID: "address", Name: "Address", Type: "#custom-type:Address"},
			},
		}

		results := validatePropertySemanticV1(localcatalog.KindProperties, specs.SpecVersionV1, nil, spec, graph)
		assert.Empty(t, results)
	})

	t.Run("custom type ref not found in type", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{LocalID: "address", Name: "Address", Type: "#custom-type:MissingAddress"},
			},
		}

		results := validatePropertySemanticV1(localcatalog.KindProperties, specs.SpecVersionV1, nil, spec, graph)
		require.Len(t, results, 1)
		assert.Equal(t, "/properties/0/type", results[0].Reference)
		assert.Contains(t, results[0].Message, "referenced custom-type 'MissingAddress' not found")
	})

	t.Run("custom type ref found in item_type", func(t *testing.T) {
		t.Parallel()

		graph := funcs.GraphWith("Address", "custom-type")
		spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{
					LocalID:  "addresses",
					Name:     "Addresses",
					Type:     "array",
					ItemType: "#custom-type:Address",
				},
			},
		}

		results := validatePropertySemanticV1(localcatalog.KindProperties, specs.SpecVersionV1, nil, spec, graph)
		assert.Empty(t, results)
	})

	t.Run("custom type ref not found in item_type", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{
					LocalID:  "addresses",
					Name:     "Addresses",
					Type:     "array",
					ItemType: "#custom-type:MissingAddress",
				},
			},
		}

		results := validatePropertySemanticV1(localcatalog.KindProperties, specs.SpecVersionV1, nil, spec, graph)
		require.Len(t, results, 1)
		assert.Equal(t, "/properties/0/item_type", results[0].Reference)
		assert.Contains(t, results[0].Message, "referenced custom-type 'MissingAddress' not found")
	})

	t.Run("invalid custom ref format is reported by semantic validation", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{
					LocalID:  "addresses",
					Name:     "Addresses",
					Type:     "#custom-types:Address",
					ItemType: "#custom-types:Address",
				},
			},
		}

		results := validatePropertySemanticV1(localcatalog.KindProperties, specs.SpecVersionV1, nil, spec, graph)
		require.Len(t, results, 2)
		assert.Equal(t, "/properties/0/type", results[0].Reference)
		assert.Equal(t, "/properties/0/item_type", results[1].Reference)
		assert.Equal(t, "must be of pattern #custom-type:<id>", results[0].Message)
		assert.Equal(t, "must be of pattern #custom-type:<id>", results[1].Message)
	})
}

func TestPropertySemanticValidV1_Uniqueness(t *testing.T) {
	t.Parallel()

	t.Run("unique combination does not fail", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResource("email", "Email", "string"))
		graph.AddResource(propertyResource("age", "Age", "number"))

		spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{LocalID: "email", Name: "Email", Type: "string"},
			},
		}

		results := validatePropertySemanticV1(localcatalog.KindProperties, specs.SpecVersionV1, nil, spec, graph)
		assert.Empty(t, results)
	})

	t.Run("duplicate name and single type", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResource("email_v1", "Email", "string"))
		graph.AddResource(propertyResource("email_v2", "Email", "string"))

		spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{LocalID: "email_v1", Name: "Email", Type: "string"},
			},
		}

		results := validatePropertySemanticV1(localcatalog.KindProperties, specs.SpecVersionV1, nil, spec, graph)
		require.Len(t, results, 1)
		assert.Equal(t, "/properties/0", results[0].Reference)
		assert.Contains(t, results[0].Message, "duplicate name 'Email'")
	})

	t.Run("types array order is ignored", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResource("mixed_v1", "Mixed", "number,string"))
		graph.AddResource(propertyResource("mixed_v2", "Mixed", "string,number"))

		spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{LocalID: "mixed_v1", Name: "Mixed", Types: []string{"string", "number"}},
			},
		}

		results := validatePropertySemanticV1(localcatalog.KindProperties, specs.SpecVersionV1, nil, spec, graph)
		require.Len(t, results, 1)
		assert.Equal(t, "/properties/0", results[0].Reference)
		assert.Contains(t, results[0].Message, "duplicate name 'Mixed'")
	})

	t.Run("item_types array order is ignored", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResource("tags_v1", "Tags", "array", map[string]any{"item_types": []any{"number", "string"}}))
		graph.AddResource(propertyResource("tags_v2", "Tags", "array", map[string]any{"item_types": []any{"string", "number"}}))

		spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{
					LocalID:   "tags_v1",
					Name:      "Tags",
					Type:      "array",
					ItemTypes: []string{"string", "number"},
				},
			},
		}

		results := validatePropertySemanticV1(localcatalog.KindProperties, specs.SpecVersionV1, nil, spec, graph)
		require.Len(t, results, 1)
		assert.Equal(t, "/properties/0", results[0].Reference)
		assert.Contains(t, results[0].Message, "duplicate name 'Tags'")
	})

	t.Run("same name and type with different item type is not duplicate", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResource("tags_str", "Tags", "array", map[string]any{"item_types": []any{"string"}}))
		graph.AddResource(propertyResource("tags_num", "Tags", "array", map[string]any{"item_types": []any{"number"}}))

		spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{
					LocalID:  "tags_str",
					Name:     "Tags",
					Type:     "array",
					ItemType: "string",
				},
			},
		}

		results := validatePropertySemanticV1(localcatalog.KindProperties, specs.SpecVersionV1, nil, spec, graph)
		assert.Empty(t, results)
	})
}

func TestPropertySemanticValid_UniquenessAcrossV0AndV1(t *testing.T) {
	t.Parallel()

	t.Run("simple property (string)", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResource("email_v0", "Email", "string"))
		graph.AddResource(propertyResource("email_v1", "Email", "string"))

		v0Spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{LocalID: "email_v0", Name: "Email", Type: "string"},
			},
		}
		v1Spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{LocalID: "email_v1", Name: "Email", Type: "string"},
			},
		}

		v0Results := validatePropertySemantic(localcatalog.KindProperties, specs.SpecVersionV0_1, nil, v0Spec, graph)
		v1Results := validatePropertySemanticV1(localcatalog.KindProperties, specs.SpecVersionV1, nil, v1Spec, graph)

		require.Len(t, v0Results, 1, "v0 validator should report duplicate")
		assert.Contains(t, v0Results[0].Message, "duplicate name, type and itemTypes")

		require.Len(t, v1Results, 1, "v1 validator should report duplicate")
		assert.Contains(t, v1Results[0].Message, "duplicate name, type and itemTypes")
	})

	t.Run("array property with item types (same order)", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResource("tags_v0", "Tags", "array", map[string]any{"item_types": []any{"string", "number"}}))
		graph.AddResource(propertyResource("tags_v1", "Tags", "array", map[string]any{"item_types": []any{"string", "number"}}))

		v0Spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{LocalID: "tags_v0", Name: "Tags", Type: "array", Config: map[string]any{"itemTypes": []any{"string", "number"}}},
			},
		}

		v1Spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{LocalID: "tags_v1", Name: "Tags", Type: "array", ItemTypes: []string{"string", "number"}},
			},
		}

		v0Results := validatePropertySemantic(
			localcatalog.KindProperties,
			specs.SpecVersionV0_1,
			nil,
			v0Spec,
			graph,
		)

		v1Results := validatePropertySemanticV1(
			localcatalog.KindProperties,
			specs.SpecVersionV1,
			nil,
			v1Spec,
			graph,
		)

		require.Len(t, v0Results, 1)
		assert.Contains(t, v0Results[0].Message, "duplicate name, type and itemTypes")

		require.Len(t, v1Results, 1)
		assert.Contains(t, v1Results[0].Message, "duplicate name, type and itemTypes")
	})

	t.Run("multiple types: v0 comma-separated vs v1 types array (different order)", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResource("mixed_v0", "Mixed", "number,string"))
		graph.AddResource(propertyResource("mixed_v1", "Mixed", "string,number"))

		v0Spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{LocalID: "mixed_v0", Name: "Mixed", Type: "string,number"},
			},
		}
		v1Spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{LocalID: "mixed_v1", Name: "Mixed", Types: []string{"number", "string"}},
			},
		}

		v0Results := validatePropertySemantic(
			localcatalog.KindProperties,
			specs.SpecVersionV0_1,
			nil,
			v0Spec,
			graph,
		)

		v1Results := validatePropertySemanticV1(
			localcatalog.KindProperties,
			specs.SpecVersionV1,
			nil,
			v1Spec,
			graph,
		)

		require.Len(t, v0Results, 1)
		assert.Contains(t, v0Results[0].Message, "duplicate name, type and itemTypes")

		require.Len(t, v1Results, 1)
		assert.Contains(t, v1Results[0].Message, "duplicate name, type and itemTypes")
	})

	t.Run("Type: v0 custom-type reference matches v1 Type reference", func(t *testing.T) {
		t.Parallel()

		addrRef := resources.PropertyRef{URN: "custom-type:Address", Property: "name"}
		graph := funcs.GraphWith("Address", "custom-type")
		graph.AddResource(propertyResource("addr_v0", "Address", addrRef))
		graph.AddResource(propertyResource("addr_v1", "Address", addrRef))

		v0Spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{LocalID: "addr_v0", Name: "Address", Type: "#custom-type:Address"},
			},
		}
		v1Spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{LocalID: "addr_v1", Name: "Address", Type: "#custom-type:Address"},
			},
		}

		v0Results := validatePropertySemantic(localcatalog.KindProperties, specs.SpecVersionV0_1, nil, v0Spec, graph)
		v1Results := validatePropertySemanticV1(localcatalog.KindProperties, specs.SpecVersionV1, nil, v1Spec, graph)

		require.Len(t, v0Results, 1)
		assert.Contains(t, v0Results[0].Message, "duplicate name, type and itemTypes")

		require.Len(t, v1Results, 1)
		assert.Contains(t, v1Results[0].Message, "duplicate name, type and itemTypes")
	})

	t.Run("item types: v0 array vs v1 array with different order", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResource("tags_v0", "Tags", "array", map[string]any{"item_types": []any{"number", "string"}}))
		graph.AddResource(propertyResource("tags_v1", "Tags", "array", map[string]any{"item_types": []any{"string", "number"}}))

		v0Spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{LocalID: "tags_v0", Name: "Tags", Type: "array", Config: map[string]any{"itemTypes": []any{"string", "number"}}},
			},
		}
		v1Spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{LocalID: "tags_v1", Name: "Tags", Type: "array", ItemTypes: []string{"number", "string"}},
			},
		}

		v0Results := validatePropertySemantic(localcatalog.KindProperties, specs.SpecVersionV0_1, nil, v0Spec, graph)
		v1Results := validatePropertySemanticV1(localcatalog.KindProperties, specs.SpecVersionV1, nil, v1Spec, graph)

		require.Len(t, v0Results, 1)
		assert.Contains(t, v0Results[0].Message, "duplicate name, type and itemTypes")

		require.Len(t, v1Results, 1)
		assert.Contains(t, v1Results[0].Message, "duplicate name, type and itemTypes")
	})

	t.Run("item types: v0 custom-type reference in itemTypes matches v1 ItemType reference", func(t *testing.T) {
		t.Parallel()

		addrRef := resources.PropertyRef{URN: "custom-type:Address", Property: "name"}
		graph := funcs.GraphWith("Address", "custom-type")
		graph.AddResource(propertyResource("addr_v0", "Addresses", "array", map[string]any{"item_types": []any{addrRef}}))
		graph.AddResource(propertyResource("addr_v1", "Addresses", "array", map[string]any{"item_types": []any{addrRef}}))

		v0Spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{LocalID: "addr_v0", Name: "Addresses", Type: "array", Config: map[string]any{"itemTypes": []any{"#custom-type:Address"}}},
			},
		}
		v1Spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{LocalID: "addr_v1", Name: "Addresses", Type: "array", ItemType: "#custom-type:Address"},
			},
		}

		v0Results := validatePropertySemantic(localcatalog.KindProperties, specs.SpecVersionV0_1, nil, v0Spec, graph)
		v1Results := validatePropertySemanticV1(localcatalog.KindProperties, specs.SpecVersionV1, nil, v1Spec, graph)

		require.Len(t, v0Results, 1)
		assert.Contains(t, v0Results[0].Message, "duplicate name, type and itemTypes")

		require.Len(t, v1Results, 1)
		assert.Contains(t, v1Results[0].Message, "duplicate name, type and itemTypes")
	})
}
