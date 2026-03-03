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

	t.Run("invalid custom ref format is ignored by semantic validation", func(t *testing.T) {
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
		assert.Empty(t, results)
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
