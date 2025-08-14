package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVariants_Diff(t *testing.T) {
	t.Parallel()

	t.Run("identical variants", func(t *testing.T) {
		t.Parallel()

		description := "Test description"
		discriminator := map[string]any{"$ref": "event_type"}
		propID := map[string]any{"$ref": "user_id"}

		variants1 := Variants{
			Variant{
				Type:          "discriminator",
				Discriminator: discriminator,
				Cases: []VariantCase{
					{
						DisplayName: "Login",
						Match:       []any{"login"},
						Description: description,
						Properties: []PropertyReference{
							{ID: propID, Required: true},
						},
					},
				},
				Default: []PropertyReference{
					{ID: "timestamp", Required: false},
				},
			},
		}

		variants2 := Variants{
			Variant{
				Type:          "discriminator",
				Discriminator: discriminator,
				Cases: []VariantCase{
					{
						DisplayName: "Login",
						Match:       []any{"login"},
						Description: description,
						Properties: []PropertyReference{
							{ID: propID, Required: true},
						},
					},
				},
				Default: []PropertyReference{
					{ID: "timestamp", Required: false},
				},
			},
		}

		assert.False(t, variants1.Diff(variants2))
	})

	t.Run("different lengths", func(t *testing.T) {
		t.Parallel()

		variants1 := Variants{
			Variant{Type: "discriminator", Discriminator: "test"},
		}
		variants2 := Variants{
			Variant{Type: "discriminator", Discriminator: "test"},
			Variant{Type: "discriminator", Discriminator: "test2"},
		}

		assert.True(t, variants1.Diff(variants2))
	})

	t.Run("nil slices", func(t *testing.T) {
		t.Parallel()

		var (
			variants1 Variants = nil
			variants2 Variants = nil
			variants3 Variants = Variants{}
		)

		// nil vs nil should be false
		assert.False(t, variants1.Diff(variants2))

		// nil vs empty should be true
		assert.True(t, variants1.Diff(variants3))
	})

	t.Run("different variant types", func(t *testing.T) {
		t.Parallel()

		variants1 := Variants{
			Variant{Type: "discriminator", Discriminator: "test"},
		}
		variants2 := Variants{
			Variant{Type: "enum", Discriminator: "test"},
		}

		assert.True(t, variants1.Diff(variants2))
	})

	t.Run("different discriminators", func(t *testing.T) {
		t.Parallel()

		variants1 := Variants{
			Variant{Type: "discriminator", Discriminator: "test1"},
		}
		variants2 := Variants{
			Variant{Type: "discriminator", Discriminator: "test2"},
		}

		assert.True(t, variants1.Diff(variants2))
	})

	t.Run("different discriminator types", func(t *testing.T) {
		t.Parallel()
		variants1 := Variants{
			Variant{Type: "discriminator", Discriminator: "string_value"},
		}
		variants2 := Variants{
			Variant{Type: "discriminator", Discriminator: map[string]any{"$ref": "property"}},
		}

		assert.True(t, variants1.Diff(variants2))
	})

	t.Run("different cases", func(t *testing.T) {
		t.Parallel()
		variants1 := Variants{
			Variant{
				Type:          "discriminator",
				Discriminator: "test",
				Cases: []VariantCase{
					{DisplayName: "Case1", Match: []any{"value1"}},
				},
			},
		}
		variants2 := Variants{
			Variant{
				Type:          "discriminator",
				Discriminator: "test",
				Cases: []VariantCase{
					{DisplayName: "Case2", Match: []any{"value1"}},
				},
			},
		}

		assert.True(t, variants1.Diff(variants2))
	})

	t.Run("different case lengths", func(t *testing.T) {
		t.Parallel()
		variants1 := Variants{
			Variant{
				Type:          "discriminator",
				Discriminator: "test",
				Cases: []VariantCase{
					{DisplayName: "Case1", Match: []any{"value1"}},
				},
			},
		}
		variants2 := Variants{
			Variant{
				Type:          "discriminator",
				Discriminator: "test",
				Cases: []VariantCase{
					{DisplayName: "Case1", Match: []any{"value1"}},
					{DisplayName: "Case2", Match: []any{"value2"}},
				},
			},
		}

		assert.True(t, variants1.Diff(variants2))
	})

	t.Run("different match values", func(t *testing.T) {
		t.Parallel()
		variants1 := Variants{
			Variant{
				Type:          "discriminator",
				Discriminator: "test",
				Cases: []VariantCase{
					{DisplayName: "Case1", Match: []any{"value1", "value2"}},
				},
			},
		}
		variants2 := Variants{
			Variant{
				Type:          "discriminator",
				Discriminator: "test",
				Cases: []VariantCase{
					{DisplayName: "Case1", Match: []any{"value1", "value3"}},
				},
			},
		}

		assert.True(t, variants1.Diff(variants2))
	})

	t.Run("different descriptions", func(t *testing.T) {
		t.Parallel()

		desc1 := "Description 1"
		desc2 := "Description 2"
		variants1 := Variants{
			Variant{
				Type:          "discriminator",
				Discriminator: "test",
				Cases: []VariantCase{
					{DisplayName: "Case1", Description: desc1},
				},
			},
		}
		variants2 := Variants{
			Variant{
				Type:          "discriminator",
				Discriminator: "test",
				Cases: []VariantCase{
					{DisplayName: "Case1", Description: desc2},
				},
			},
		}

		assert.True(t, variants1.Diff(variants2))
	})

	t.Run("nil vs non nil description", func(t *testing.T) {
		t.Parallel()

		desc := "Description"
		variants1 := Variants{
			Variant{
				Type:          "discriminator",
				Discriminator: "test",
				Cases: []VariantCase{
					{DisplayName: "Case1", Description: ""},
				},
			},
		}
		variants2 := Variants{
			Variant{
				Type:          "discriminator",
				Discriminator: "test",
				Cases: []VariantCase{
					{DisplayName: "Case1", Description: desc},
				},
			},
		}

		assert.True(t, variants1.Diff(variants2))
	})

	t.Run("different properties", func(t *testing.T) {
		t.Parallel()

		variants1 := Variants{
			Variant{
				Type:          "discriminator",
				Discriminator: "test",
				Cases: []VariantCase{
					{
						DisplayName: "Case1",
						Properties: []PropertyReference{
							{ID: "prop1", Required: true},
						},
					},
				},
			},
		}
		variants2 := Variants{
			Variant{
				Type:          "discriminator",
				Discriminator: "test",
				Cases: []VariantCase{
					{
						DisplayName: "Case1",
						Properties: []PropertyReference{
							{ID: "prop1", Required: false},
						},
					},
				},
			},
		}

		assert.True(t, variants1.Diff(variants2))
	})

	t.Run("different property IDs", func(t *testing.T) {
		t.Parallel()

		variants1 := Variants{
			Variant{
				Type:          "discriminator",
				Discriminator: "test",
				Cases: []VariantCase{
					{
						DisplayName: "Case1",
						Properties: []PropertyReference{
							{ID: map[string]any{"$ref": "prop1"}, Required: true},
						},
					},
				},
			},
		}
		variants2 := Variants{
			Variant{
				Type:          "discriminator",
				Discriminator: "test",
				Cases: []VariantCase{
					{
						DisplayName: "Case1",
						Properties: []PropertyReference{
							{ID: map[string]any{"$ref": "prop2"}, Required: true},
						},
					},
				},
			},
		}

		assert.True(t, variants1.Diff(variants2))
	})

	t.Run("different default properties", func(t *testing.T) {
		t.Parallel()

		variants1 := Variants{
			Variant{
				Type:          "discriminator",
				Discriminator: "test",
				Default: []PropertyReference{
					{ID: "default1", Required: true},
				},
			},
		}
		variants2 := Variants{
			Variant{
				Type:          "discriminator",
				Discriminator: "test",
				Default: []PropertyReference{
					{ID: "default2", Required: true},
				},
			},
		}

		assert.True(t, variants1.Diff(variants2))
	})

	t.Run("empty vs nil slices", func(t *testing.T) {
		t.Parallel()

		variants1 := Variants{
			Variant{
				Type:          "discriminator",
				Discriminator: "test",
				Cases:         []VariantCase{},
				Default:       []PropertyReference{},
			},
		}
		variants2 := Variants{
			Variant{
				Type:          "discriminator",
				Discriminator: "test",
				Cases:         nil,
				Default:       nil,
			},
		}

		assert.False(t, variants1.Diff(variants2))
	})
}

func TestVariants_ResourceData(t *testing.T) {

	t.Run("ToResourceData()", func(t *testing.T) {
		t.Parallel()

		t.Run("nil variants generates empty map", func(t *testing.T) {
			t.Parallel()
			var variants Variants
			assert.Equal(t, []map[string]any{}, variants.ToResourceData())
		})

		t.Run("variants with empty cases and default generates correct resource data", func(t *testing.T) {
			t.Parallel()

			var variants Variants
			variants = append(variants, Variant{
				Type:          "discriminator",
				Discriminator: "test",
			})

			assert.Equal(t, []map[string]any{
				{
					"type":          "discriminator",
					"discriminator": "test",
					"cases":         []map[string]any{}, // empty cases
					"default":       []map[string]any{}, // empty default
				},
			}, variants.ToResourceData())
		})

		t.Run("variants with valid cases and default generates correct resource data", func(t *testing.T) {
			t.Parallel()

			var variants Variants
			variants = append(variants, Variant{
				Type:          "discriminator",
				Discriminator: "test",
				Cases: []VariantCase{
					{
						DisplayName: "Case1",
						Match:       []any{"value1"},
						Description: "Description 1",
						Properties: []PropertyReference{
							{ID: "upstream-prop1", Required: true},
							{ID: "upstream-prop2", Required: false},
						},
					},
					{
						DisplayName: "Case2",
						Match:       []any{"value1, value2"},
						Description: "Description 2",
						Properties: []PropertyReference{
							{ID: "upstream-prop3", Required: true},
							{ID: "upstream-prop4", Required: false},
						},
					},
				},
				Default: []PropertyReference{
					{ID: "default1", Required: true},
				},
			})
			assert.Equal(t, []map[string]any{
				{
					"type":          "discriminator",
					"discriminator": "test",
					"cases": []map[string]any{
						{
							"display_name": "Case1",
							"match":        []any{"value1"},
							"description":  "Description 1",
							"properties": []map[string]any{
								{
									"id":       "upstream-prop1",
									"required": true,
								},
								{
									"id":       "upstream-prop2",
									"required": false,
								},
							},
						},
						{
							"display_name": "Case2",
							"match":        []any{"value1, value2"},
							"description":  "Description 2",
							"properties": []map[string]any{
								{
									"id":       "upstream-prop3",
									"required": true,
								},
								{
									"id":       "upstream-prop4",
									"required": false,
								},
							},
						},
					},
					"default": []map[string]any{
						{
							"id":       "default1",
							"required": true,
						},
					},
				},
			}, variants.ToResourceData())

		})

	})

	t.Run("FromResourceData()", func(t *testing.T) {
		t.Parallel()

		t.Run("empty resource data generates empty variants", func(t *testing.T) {
			t.Parallel()

			var (
				actualVariants   Variants
				expectedVariants Variants
			)

			actualVariants.FromResourceData([]map[string]any{})
			assert.Equal(t, expectedVariants, actualVariants)
		})

		t.Run("resource data with valid cases and default generates correct variants", func(t *testing.T) {
			t.Parallel()

			var (
				actualVariants   Variants
				expectedVariants Variants = []Variant{
					{
						Type:          "discriminator",
						Discriminator: "test",
						Cases: []VariantCase{
							{
								DisplayName: "Case1",
								Match:       []any{"value1"},
								Description: "Description 1",
								Properties: []PropertyReference{
									{ID: "upstream-prop1", Required: true},
								},
							},
						},
						Default: []PropertyReference{
							{ID: "default1", Required: true},
						},
					},
				}
			)

			actualVariants.FromResourceData([]map[string]any{
				{
					"type":          "discriminator",
					"discriminator": "test",
					"cases": []map[string]any{
						{
							"display_name": "Case1",
							"match":        []any{"value1"},
							"description":  "Description 1",
							"properties": []map[string]any{
								{
									"id":       "upstream-prop1",
									"required": true,
								},
							},
						},
					},
					"default": []map[string]any{
						{
							"id":       "default1",
							"required": true,
						},
					},
				},
			})

			assert.Equal(t, expectedVariants, actualVariants)
		})
	})
}
