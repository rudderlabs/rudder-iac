package unflatten

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnflattenComprehensive(t *testing.T) {
	t.Parallel()

	cases := []struct {
		category string
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
		check    func(t *testing.T, result map[string]interface{})
	}{
		// Basic Cases
		{
			category: "Basic",
			name:     "SimpleKeys",
			input: map[string]interface{}{
				"name":  "John",
				"age":   30,
				"email": "john@example.com",
			},
			expected: map[string]interface{}{
				"name":  "John",
				"age":   30,
				"email": "john@example.com",
			},
		},
		{
			category: "Basic",
			name:     "DottedKeys",
			input: map[string]interface{}{
				"user.name":    "John",
				"user.age":     30,
				"context.ip":   "192.168.1.1",
				"context.city": "New York",
			},
			expected: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "John",
					"age":  30,
				},
				"context": map[string]interface{}{
					"ip":   "192.168.1.1",
					"city": "New York",
				},
			},
		},
		{
			category: "Basic",
			name:     "EmptyInput",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
		{
			category: "Basic",
			name:     "SingleKey",
			input: map[string]interface{}{
				"key": "value",
			},
			expected: map[string]interface{}{
				"key": "value",
			},
		},

		// Array Cases
		{
			category: "Arrays",
			name:     "ArrayIndexes",
			input: map[string]interface{}{
				"items.0.name":  "item1",
				"items.0.price": 10.99,
				"items.1.name":  "item2",
				"tags.0":        "tag1",
				"tags.1":        "tag2",
			},
			expected: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{
						"name":  "item1",
						"price": 10.99,
					},
					map[string]interface{}{
						"name": "item2",
					},
				},
				"tags": []interface{}{
					"tag1",
					"tag2",
				},
			},
		},
		{
			category: "Arrays",
			name:     "ArrayWithGaps",
			input: map[string]interface{}{
				"arr.0": "first",
				"arr.2": "third",
				"arr.5": "sixth",
			},
			expected: map[string]interface{}{
				"arr": []interface{}{
					"first",
					nil,
					"third",
					nil,
					nil,
					"sixth",
				},
			},
		},
		{
			category: "Arrays",
			name:     "ComplexArrayIndexes",
			input: map[string]interface{}{
				"matrix.0.0": "a",
				"matrix.0.1": "b",
				"matrix.1.0": "c",
				"matrix.1.1": "d",
			},
			expected: map[string]interface{}{
				"matrix": []interface{}{
					[]interface{}{"a", "b"},
					[]interface{}{"c", "d"},
				},
			},
		},

		// Deep Nesting Cases
		{
			category: "Nesting",
			name:     "DeepNesting",
			input: map[string]interface{}{
				"user.profile.personal.name":      "John",
				"user.profile.personal.age":       30,
				"user.profile.professional.title": "Developer",
				"user.profile.professional.years": 5,
				"user.settings.theme":             "dark",
				"user.settings.notifications":     true,
			},
			expected: map[string]interface{}{
				"user": map[string]interface{}{
					"profile": map[string]interface{}{
						"personal": map[string]interface{}{
							"name": "John",
							"age":  30,
						},
						"professional": map[string]interface{}{
							"title": "Developer",
							"years": 5,
						},
					},
					"settings": map[string]interface{}{
						"theme":         "dark",
						"notifications": true,
					},
				},
			},
		},
		{
			category: "Nesting",
			name:     "MixedStructure",
			input: map[string]interface{}{
				"event":                "product_viewed",
				"userId":               "123",
				"properties.name":      "iPhone",
				"properties.price":     999.99,
				"properties.tags.0":    "electronics",
				"properties.tags.1":    "phone",
				"context.app.name":     "MyApp",
				"context.app.version":  "1.0.0",
				"context.library.name": "analytics-js",
			},
			expected: map[string]interface{}{
				"event":  "product_viewed",
				"userId": "123",
				"properties": map[string]interface{}{
					"name":  "iPhone",
					"price": 999.99,
					"tags": []interface{}{
						"electronics",
						"phone",
					},
				},
				"context": map[string]interface{}{
					"app": map[string]interface{}{
						"name":    "MyApp",
						"version": "1.0.0",
					},
					"library": map[string]interface{}{
						"name": "analytics-js",
					},
				},
			},
		},

		// Edge Cases
		{
			category: "EdgeCases",
			name:     "NilInput",
			input:    nil,
			expected: map[string]interface{}{},
		},
		{
			category: "EdgeCases",
			name:     "StringArrayIndexes",
			input: map[string]interface{}{
				"items.first.name":  "item1",
				"items.second.name": "item2",
			},
			expected: map[string]interface{}{
				"items": map[string]interface{}{
					"first": map[string]interface{}{
						"name": "item1",
					},
					"second": map[string]interface{}{
						"name": "item2",
					},
				},
			},
		},

		// Real World Scenarios
		{
			category: "RealWorld",
			name:     "RudderStackEventSchema",
			input: map[string]interface{}{
				"anonymousId":              "string",
				"channel":                  "string",
				"context.app.name":         "string",
				"context.app.version":      "string",
				"context.device.type":      "string",
				"context.library.name":     "string",
				"context.traits.email":     "string",
				"context.traits.firstName": "string",
				"event":                    "string",
				"properties.category":      "string",
				"properties.product_id":    "string",
				"userId":                   "string",
			},
			check: func(t *testing.T, result map[string]interface{}) {
				assert.Contains(t, result, "context")
				context := result["context"].(map[string]interface{})
				assert.Equal(t, "string", context["app"].(map[string]interface{})["name"])
				assert.Equal(t, "string", context["traits"].(map[string]interface{})["email"])
				assert.Equal(t, "string", result["properties"].(map[string]interface{})["category"])
			},
		},
		{
			category: "RealWorld",
			name:     "DeepNestedPath",
			input: map[string]interface{}{
				"a.b.c.d.e.f.g.h.i.j": "deep_value",
				"a.b.c.d.e.f.g.h.x":   "another_value",
			},
			check: func(t *testing.T, result map[string]interface{}) {
				h := result["a"].(map[string]interface{})["b"].(map[string]interface{})["c"].(map[string]interface{})["d"].(map[string]interface{})["e"].(map[string]interface{})["f"].(map[string]interface{})["g"].(map[string]interface{})["h"].(map[string]interface{})
				assert.Equal(t, "deep_value", h["i"].(map[string]interface{})["j"])
				assert.Equal(t, "another_value", h["x"])
			},
		},
		{
			category: "RealWorld",
			name:     "MixedArraysAndObjects",
			input: map[string]interface{}{
				"data.0.items.0.properties.name": "item1",
				"data.0.items.1.properties.name": "item2",
				"data.1.items.0.properties.name": "item3",
				"data.1.metadata.processed":      true,
			},
			check: func(t *testing.T, result map[string]interface{}) {
				data := result["data"].([]interface{})
				require.Len(t, data, 2)
				firstItem := data[0].(map[string]interface{})["items"].([]interface{})[0].(map[string]interface{})["properties"].(map[string]interface{})
				assert.Equal(t, "item1", firstItem["name"])
			},
		},
		{
			category: "RealWorld",
			name:     "LargeArrayIndexes",
			input: map[string]interface{}{
				"items.100.id":   "large_index",
				"items.1000.id":  "very_large_index",
				"items.10000.id": "extremely_large_index",
			},
			check: func(t *testing.T, result map[string]interface{}) {
				items := result["items"].([]interface{})
				require.True(t, len(items) > 10000)
				assert.Equal(t, "large_index", items[100].(map[string]interface{})["id"])
				assert.Equal(t, "extremely_large_index", items[10000].(map[string]interface{})["id"])
			},
		},

		// Special Cases
		{
			category: "SpecialCases",
			name:     "ArrayGaps",
			input: map[string]interface{}{
				"items.0.name": "first",
				"items.2.name": "third",
				"items.3.name": "fourth",
			},
			check: func(t *testing.T, result map[string]interface{}) {
				items := result["items"].([]interface{})
				require.Len(t, items, 4)
				assert.Nil(t, items[1])
				assert.Equal(t, "first", items[0].(map[string]interface{})["name"])
				assert.Equal(t, "third", items[2].(map[string]interface{})["name"])
			},
		},
		{
			category: "SpecialCases",
			name:     "MixedKeyTypes",
			input: map[string]interface{}{
				"data":   "direct",
				"data.0": "array_item",
				"data.a": "object_prop",
			},
			check: func(t *testing.T, result map[string]interface{}) {
				assert.NotNil(t, result)
				assert.Contains(t, result, "data")
			},
		},
	}

	for _, c := range cases {
		t.Run(c.category+"/"+c.name, func(t *testing.T) {
			t.Parallel()
			result := UnflattenSchema(c.input)

			if c.check != nil {
				c.check(t, result)
			} else {
				assert.Equal(t, c.expected, result)
			}
		})
	}
}

func BenchmarkUnflatten(b *testing.B) {
	input := map[string]interface{}{
		"event":                      "string",
		"userId":                     "string",
		"context.app.name":           "string",
		"context.device.type":        "string",
		"properties.product_id":      "string",
		"properties.categories.0":    "string",
		"properties.categories.1":    "string",
		"properties.custom_data.key": "string",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		UnflattenSchema(input)
	}
}

func BenchmarkUnflattenLargeArray(b *testing.B) {
	input := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		input[fmt.Sprintf("items.%d.id", i)] = fmt.Sprintf("item_%d", i)
		input[fmt.Sprintf("items.%d.value", i)] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		UnflattenSchema(input)
	}
}
