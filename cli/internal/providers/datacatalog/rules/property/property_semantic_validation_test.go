package property

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// propertyResource creates a property resource with name, type, and optional config
// in its data, matching the shape produced by PropertyArgs.ToResourceData().
func propertyResource(id, name string, typ any, config ...map[string]any) *resources.Resource {
	data := resources.ResourceData{
		"name": name,
		"type": typ,
	}
	if len(config) > 0 {
		data["config"] = config[0]
	}
	return resources.NewResource(id, "property", data, nil)
}

func TestPropertySemanticValid_ReferenceType(t *testing.T) {
	t.Parallel()

	t.Run("custom type ref found in Type", func(t *testing.T) {
		t.Parallel()

		graph := funcs.GraphWith("Address", "custom-type")

		spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{LocalID: "address", Name: "Address", Type: "#custom-type:Address"},
			},
		}

		results := validatePropertySemantic(localcatalog.KindProperties, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "custom type exists in graph — no errors expected")
	})

	t.Run("custom type ref not found in Type", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()

		spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{LocalID: "address", Name: "Address", Type: "#custom-type:NonexistentType"},
			},
		}

		results := validatePropertySemantic(localcatalog.KindProperties, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/properties/0/type", results[0].Reference)
		assert.Contains(t, results[0].Message, "referenced custom-type 'NonexistentType' not found")
	})

	t.Run("primitive types skipped", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()

		spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{LocalID: "name", Name: "Name", Type: "string"},
				{LocalID: "age", Name: "Age", Type: "number"},
				{LocalID: "active", Name: "Active", Type: "boolean"},
				{LocalID: "tags", Name: "Tags", Type: "array"},
				{LocalID: "meta", Name: "Meta", Type: "object"},
			},
		}

		results := validatePropertySemantic(localcatalog.KindProperties, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "primitive types should not trigger ref lookup")
	})

	t.Run("mixed primitives and refs in Type", func(t *testing.T) {
		t.Parallel()

		graph := funcs.GraphWith("Address", "custom-type")

		spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{LocalID: "name", Name: "Name", Type: "string"},
				{LocalID: "address", Name: "Address", Type: "#custom-type:Address"},
				{LocalID: "payment", Name: "Payment", Type: "#custom-type:MissingPaymentType"},
			},
		}

		results := validatePropertySemantic(localcatalog.KindProperties, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/properties/2/type", results[0].Reference)
		assert.Contains(t, results[0].Message, "referenced custom-type 'MissingPaymentType' not found")
	})

	t.Run("empty type skipped", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()

		spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{LocalID: "name", Name: "Name", Type: ""},
			},
		}

		results := validatePropertySemantic(localcatalog.KindProperties, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "empty type should be skipped")
	})

	t.Run("itemTypes custom type ref found", func(t *testing.T) {
		t.Parallel()

		graph := funcs.GraphWith("Address", "custom-type")

		spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{
					LocalID: "tags",
					Name:    "Tags",
					Type:    "array",
					Config:  map[string]interface{}{"itemTypes": []any{"#custom-type:Address"}},
				},
			},
		}

		results := validatePropertySemantic(localcatalog.KindProperties, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "itemTypes custom type ref exists in graph — no error")
	})

	t.Run("itemTypes custom type ref not found", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()

		spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{
					LocalID: "tags",
					Name:    "Tags",
					Type:    "array",
					Config:  map[string]interface{}{"itemTypes": []any{"#custom-type:Missing"}},
				},
			},
		}

		results := validatePropertySemantic(localcatalog.KindProperties, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/properties/0/propConfig/itemTypes/0", results[0].Reference)
		assert.Contains(t, results[0].Message, "referenced custom-type 'Missing' not found")
	})

	t.Run("itemTypes mixed primitives and custom type refs", func(t *testing.T) {
		t.Parallel()

		graph := funcs.GraphWith("Address", "custom-type")

		spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{
					LocalID: "tags",
					Name:    "Tags",
					Type:    "array",
					Config:  map[string]interface{}{"itemTypes": []any{"string", "#custom-type:Address", "#custom-type:Missing"}},
				},
			},
		}

		results := validatePropertySemantic(localcatalog.KindProperties, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/properties/0/propConfig/itemTypes/2", results[0].Reference)
		assert.Contains(t, results[0].Message, "referenced custom-type 'Missing' not found")
	})

	t.Run("no itemTypes in config — no error", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()

		spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{
					LocalID: "age",
					Name:    "Age",
					Type:    "number",
					Config:  map[string]interface{}{"minimum": 0},
				},
			},
		}

		results := validatePropertySemantic(localcatalog.KindProperties, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "config without itemTypes should not trigger ref check")
	})
}

func TestPropertySemanticValid_Uniqueness(t *testing.T) {
	t.Parallel()

	t.Run("no duplicate — unique properties", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResource("email", "Email", "string"))
		graph.AddResource(propertyResource("age", "Age", "number"))

		spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{LocalID: "email", Name: "Email", Type: "string"},
			},
		}

		results := validatePropertySemantic(localcatalog.KindProperties, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "unique (name, type) pairs should not trigger errors")
	})

	t.Run("same name different type is not a duplicate", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResource("email_str", "Email", "string"))
		graph.AddResource(propertyResource("email_num", "Email", "number"))

		spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{LocalID: "email_str", Name: "Email", Type: "string"},
			},
		}

		results := validatePropertySemantic(localcatalog.KindProperties, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "same name but different type is not a duplicate")
	})

	t.Run("duplicate detected across graph", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResource("email_v1", "Email", "string"))
		graph.AddResource(propertyResource("email_v2", "Email", "string"))

		spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{LocalID: "email_v1", Name: "Email", Type: "string"},
			},
		}

		results := validatePropertySemantic(localcatalog.KindProperties, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/properties/0", results[0].Reference)
		assert.Contains(t, results[0].Message, "duplicate name")
	})

	t.Run("single in graph — no false positive", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResource("email", "Email", "string"))

		spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{LocalID: "email", Name: "Email", Type: "string"},
			},
		}

		results := validatePropertySemantic(localcatalog.KindProperties, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "single property in graph should not be flagged")
	})

	t.Run("custom type ref duplicate", func(t *testing.T) {
		t.Parallel()

		// In the real graph, custom type refs are stored as PropertyRef, not strings
		customTypeRef := resources.PropertyRef{URN: "custom-type:Addr", Property: "name"}
		duplicateCustomTypeRef := resources.PropertyRef{URN: "custom-type:Addr", Property: "name"}

		graph := funcs.GraphWith("Addr", "custom-type")
		graph.AddResource(propertyResource("addr_v1", "Address", customTypeRef))
		graph.AddResource(propertyResource("addr_v2", "Address", duplicateCustomTypeRef))

		spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{LocalID: "addr_v1", Name: "Address", Type: "#custom-type:Addr"},
			},
		}

		results := validatePropertySemantic(localcatalog.KindProperties, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/properties/0", results[0].Reference)
		assert.Contains(t, results[0].Message, "duplicate name")
	})

	t.Run("comma-separated type order ignored", func(t *testing.T) {
		t.Parallel()

		// Graph has "number,string" (sorted by PropertyArgs) and "string,number" (unsorted)
		// Both normalize to "number,string" so they are duplicates
		graph := resources.NewGraph()
		graph.AddResource(propertyResource("mixed_v1", "Mixed", "number,string"))
		graph.AddResource(propertyResource("mixed_v2", "Mixed", "string,number"))

		spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{LocalID: "mixed_v1", Name: "Mixed", Type: "string,number"},
			},
		}

		results := validatePropertySemantic(localcatalog.KindProperties, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/properties/0", results[0].Reference)
		assert.Contains(t, results[0].Message, "duplicate name")
	})

	t.Run("same name and type but different itemTypes — not a duplicate", func(t *testing.T) {
		t.Parallel()

		// Both are array type with name "Tags" but different itemTypes
		graph := resources.NewGraph()
		graph.AddResource(propertyResource("tags_str", "Tags", "array",
			map[string]any{"item_types": []any{"string"}},
		))
		graph.AddResource(propertyResource("tags_num", "Tags", "array",
			map[string]any{"item_types": []any{"number"}},
		))

		spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{LocalID: "tags_str", Name: "Tags", Type: "array",
					Config: map[string]any{"itemTypes": []any{"string"}}},
			},
		}

		results := validatePropertySemantic(localcatalog.KindProperties, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "same (name, type) but different itemTypes is not a duplicate")
	})

	t.Run("same name type and itemTypes — duplicate detected", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(propertyResource("tags_v1", "Tags", "array",
			map[string]any{"item_types": []any{"string", "number"}},
		))
		graph.AddResource(propertyResource("tags_v2", "Tags", "array",
			map[string]any{"item_types": []any{"string", "number"}},
		))

		spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{LocalID: "tags_v1", Name: "Tags", Type: "array",
					Config: map[string]any{"itemTypes": []any{"string", "number"}}},
			},
		}

		results := validatePropertySemantic(localcatalog.KindProperties, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/properties/0", results[0].Reference)
		assert.Contains(t, results[0].Message, "duplicate name")
	})

	t.Run("itemTypes order ignored in uniqueness check", func(t *testing.T) {
		t.Parallel()

		// Graph has sorted item_types, spec has unsorted — should still match
		graph := resources.NewGraph()
		graph.AddResource(propertyResource("tags_v1", "Tags", "array",
			map[string]any{"item_types": []any{"number", "string"}},
		))
		graph.AddResource(propertyResource("tags_v2", "Tags", "array",
			map[string]any{"item_types": []any{"string", "number"}},
		))

		spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{LocalID: "tags_v1", Name: "Tags", Type: "array",
					Config: map[string]any{"itemTypes": []any{"string", "number"}}},
			},
		}

		results := validatePropertySemantic(localcatalog.KindProperties, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "duplicate name")
	})

	t.Run("itemTypes with custom type ref — duplicate detected", func(t *testing.T) {
		t.Parallel()

		// Graph stores custom type item_types as PropertyRef
		var (
			customTypeRef          = resources.PropertyRef{URN: "custom-type:Addr", Property: "name"}
			duplicateCustomTypeRef = resources.PropertyRef{URN: "custom-type:Addr", Property: "name"}
		)

		graph := funcs.GraphWith("Addr", "custom-type")
		graph.AddResource(propertyResource("tags_v1", "Tags", "array",
			map[string]any{"item_types": []any{customTypeRef}},
		))
		graph.AddResource(propertyResource("tags_v2", "Tags", "array",
			map[string]any{"item_types": []any{duplicateCustomTypeRef}},
		))

		// Spec uses string format for the ref
		spec := localcatalog.PropertySpec{
			Properties: []localcatalog.Property{
				{LocalID: "tags_v1", Name: "Tags", Type: "array",
					Config: map[string]any{"itemTypes": []any{"#custom-type:Addr"}}},
			},
		}

		results := validatePropertySemantic(localcatalog.KindProperties, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "duplicate name")
	})
}
