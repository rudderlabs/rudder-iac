package unflatten

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnflatten(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "SimpleKeys",
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
			name: "DottedKeys",
			input: map[string]interface{}{
				"user.name":    "John",
				"user.age":     30,
				"user.email":   "john@example.com",
				"context.ip":   "192.168.1.1",
				"context.city": "New York",
			},
			expected: map[string]interface{}{
				"user": map[string]interface{}{
					"name":  "John",
					"age":   30,
					"email": "john@example.com",
				},
				"context": map[string]interface{}{
					"ip":   "192.168.1.1",
					"city": "New York",
				},
			},
		},
		{
			name: "ArrayIndexes",
			input: map[string]interface{}{
				"items.0.name":  "item1",
				"items.0.price": 10.99,
				"items.1.name":  "item2",
				"items.1.price": 15.99,
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
						"name":  "item2",
						"price": 15.99,
					},
				},
				"tags": []interface{}{
					"tag1",
					"tag2",
				},
			},
		},
		{
			name: "DeepNesting",
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
			name: "MixedStructure",
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
		{
			name:     "EmptyInput",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
		{
			name: "SingleKey",
			input: map[string]interface{}{
				"key": "value",
			},
			expected: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name: "ArrayWithGaps",
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
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			result := UnflattenSchema(c.input)
			assert.Equal(t, c.expected, result)
		})
	}
}

func TestUnflatten_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("NilInput", func(t *testing.T) {
		t.Parallel()

		result := UnflattenSchema(nil)
		assert.Equal(t, map[string]interface{}{}, result)
	})

	t.Run("ComplexArrayIndexes", func(t *testing.T) {
		t.Parallel()

		input := map[string]interface{}{
			"matrix.0.0": "a",
			"matrix.0.1": "b",
			"matrix.1.0": "c",
			"matrix.1.1": "d",
		}

		result := UnflattenSchema(input)

		expected := map[string]interface{}{
			"matrix": []interface{}{
				[]interface{}{"a", "b"},
				[]interface{}{"c", "d"},
			},
		}

		assert.Equal(t, expected, result)
	})

	t.Run("MixedKeyTypes", func(t *testing.T) {
		t.Parallel()

		input := map[string]interface{}{
			"data":   "direct",
			"data.0": "array_item",
			"data.a": "object_prop",
		}

		result := UnflattenSchema(input)

		// The behavior depends on implementation, but it should handle gracefully
		assert.NotNil(t, result)
		assert.Contains(t, result, "data")
	})

	t.Run("StringArrayIndexes", func(t *testing.T) {
		t.Parallel()

		input := map[string]interface{}{
			"items.first.name":  "item1",
			"items.second.name": "item2",
		}

		result := UnflattenSchema(input)

		expected := map[string]interface{}{
			"items": map[string]interface{}{
				"first": map[string]interface{}{
					"name": "item1",
				},
				"second": map[string]interface{}{
					"name": "item2",
				},
			},
		}

		assert.Equal(t, expected, result)
	})
}

func TestUnflatten_RealWorldScenarios(t *testing.T) {
	t.Parallel()

	t.Run("RudderStackEventSchema", func(t *testing.T) {
		t.Parallel()

		input := map[string]interface{}{
			"anonymousId":                  "string",
			"channel":                      "string",
			"context.app.name":             "string",
			"context.app.version":          "string",
			"context.campaign.content":     "string",
			"context.campaign.medium":      "string",
			"context.campaign.name":        "string",
			"context.campaign.source":      "string",
			"context.campaign.term":        "string",
			"context.device.advertisingId": "string",
			"context.device.id":            "string",
			"context.device.manufacturer":  "string",
			"context.device.model":         "string",
			"context.device.name":          "string",
			"context.device.type":          "string",
			"context.ip":                   "string",
			"context.library.name":         "string",
			"context.library.version":      "string",
			"context.locale":               "string",
			"context.network.carrier":      "string",
			"context.network.cellular":     "boolean",
			"context.network.wifi":         "boolean",
			"context.os.name":              "string",
			"context.os.version":           "string",
			"context.page.path":            "string",
			"context.page.referrer":        "string",
			"context.page.search":          "string",
			"context.page.title":           "string",
			"context.page.url":             "string",
			"context.screen.density":       "number",
			"context.screen.height":        "number",
			"context.screen.width":         "number",
			"context.timezone":             "string",
			"context.traits.email":         "string",
			"context.traits.firstName":     "string",
			"context.traits.lastName":      "string",
			"context.userAgent":            "string",
			"event":                        "string",
			"integrations.All":             "boolean",
			"messageId":                    "string",
			"originalTimestamp":            "string",
			"properties.category":          "string",
			"properties.product_id":        "string",
			"properties.sku":               "string",
			"receivedAt":                   "string",
			"sentAt":                       "string",
			"timestamp":                    "string",
			"type":                         "string",
			"userId":                       "string",
		}

		result := UnflattenSchema(input)

		// Verify the structure is properly unflattened
		assert.Contains(t, result, "context")
		contextMap, ok := result["context"].(map[string]interface{})
		assert.True(t, ok)

		assert.Contains(t, contextMap, "app")
		appMap, ok := contextMap["app"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "string", appMap["name"])
		assert.Equal(t, "string", appMap["version"])

		assert.Contains(t, contextMap, "traits")
		traitsMap, ok := contextMap["traits"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "string", traitsMap["email"])
		assert.Equal(t, "string", traitsMap["firstName"])
		assert.Equal(t, "string", traitsMap["lastName"])

		assert.Contains(t, result, "properties")
		propertiesMap, ok := result["properties"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "string", propertiesMap["category"])
		assert.Equal(t, "string", propertiesMap["product_id"])
		assert.Equal(t, "string", propertiesMap["sku"])
	})
}

func BenchmarkUnflatten(b *testing.B) {
	input := map[string]interface{}{
		"event":                      "string",
		"userId":                     "string",
		"anonymousId":                "string",
		"context.app.name":           "string",
		"context.app.version":        "string",
		"context.device.type":        "string",
		"context.library.name":       "string",
		"context.library.version":    "string",
		"context.ip":                 "string",
		"context.userAgent":          "string",
		"properties.product_id":      "string",
		"properties.product_name":    "string",
		"properties.price":           "number",
		"properties.currency":        "string",
		"properties.quantity":        "number",
		"properties.discount":        "number",
		"properties.categories.0":    "string",
		"properties.categories.1":    "string",
		"properties.custom_data.key": "string",
		"timestamp":                  "timestamp",
		"sentAt":                     "timestamp",
		"messageId":                  "string",
		"type":                       "string",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		UnflattenSchema(input)
	}
}

func TestUnflattenSchema_ArrayGaps(t *testing.T) {
	t.Parallel()

	// Test arrays with gaps
	input := map[string]interface{}{
		"items.0.name": "first",
		"items.2.name": "third", // gap at index 1
		"items.3.name": "fourth",
	}

	result := UnflattenSchema(input)

	items, ok := result["items"].([]interface{})
	require.True(t, ok)
	require.Len(t, items, 4) // Should have 4 elements including nil at index 1

	// Check that index 1 is nil (gap)
	assert.Nil(t, items[1])

	// Check other values
	firstItem := items[0].(map[string]interface{})
	assert.Equal(t, "first", firstItem["name"])

	thirdItem := items[2].(map[string]interface{})
	assert.Equal(t, "third", thirdItem["name"])

	fourthItem := items[3].(map[string]interface{})
	assert.Equal(t, "fourth", fourthItem["name"])
}

func TestUnflattenSchema_DeepNesting(t *testing.T) {
	t.Parallel()

	input := map[string]interface{}{
		"a.b.c.d.e.f.g.h.i.j": "deep_value",
		"a.b.c.d.e.f.g.h.x":   "another_value",
	}

	result := UnflattenSchema(input)

	// Navigate to the deep value
	a := result["a"].(map[string]interface{})
	b := a["b"].(map[string]interface{})
	c := b["c"].(map[string]interface{})
	d := c["d"].(map[string]interface{})
	e := d["e"].(map[string]interface{})
	f := e["f"].(map[string]interface{})
	g := f["g"].(map[string]interface{})
	h := g["h"].(map[string]interface{})

	assert.Equal(t, "deep_value", h["i"].(map[string]interface{})["j"])
	assert.Equal(t, "another_value", h["x"])
}

func TestUnflattenSchema_MixedArraysAndObjects(t *testing.T) {
	t.Parallel()

	input := map[string]interface{}{
		"data.0.items.0.properties.name": "item1",
		"data.0.items.1.properties.name": "item2",
		"data.1.items.0.properties.name": "item3",
		"data.1.metadata.processed":      true,
		"data.1.metadata.timestamp":      "2023-01-01",
	}

	result := UnflattenSchema(input)

	data := result["data"].([]interface{})
	require.Len(t, data, 2)

	// First data item
	firstData := data[0].(map[string]interface{})
	firstItems := firstData["items"].([]interface{})
	require.Len(t, firstItems, 2)

	firstItem := firstItems[0].(map[string]interface{})["properties"].(map[string]interface{})
	assert.Equal(t, "item1", firstItem["name"])

	secondItem := firstItems[1].(map[string]interface{})["properties"].(map[string]interface{})
	assert.Equal(t, "item2", secondItem["name"])

	// Second data item
	secondData := data[1].(map[string]interface{})
	secondItems := secondData["items"].([]interface{})
	require.Len(t, secondItems, 1)

	thirdItem := secondItems[0].(map[string]interface{})["properties"].(map[string]interface{})
	assert.Equal(t, "item3", thirdItem["name"])

	metadata := secondData["metadata"].(map[string]interface{})
	assert.Equal(t, true, metadata["processed"])
	assert.Equal(t, "2023-01-01", metadata["timestamp"])
}

func TestUnflattenSchema_LargeArrayIndexes(t *testing.T) {
	t.Parallel()

	input := map[string]interface{}{
		"items.100.name":  "item_100",
		"items.1000.name": "item_1000",
	}

	result := UnflattenSchema(input)

	items := result["items"].([]interface{})
	require.True(t, len(items) >= 1001) // Should have at least 1001 elements

	// Check that item at index 100 exists
	item100 := items[100].(map[string]interface{})
	assert.Equal(t, "item_100", item100["name"])

	// Check that item at index 1000 exists
	item1000 := items[1000].(map[string]interface{})
	assert.Equal(t, "item_1000", item1000["name"])

	// Check that gaps exist (nil values)
	for i := 0; i < 100; i++ {
		assert.Nil(t, items[i])
	}
}

func TestUnflattenSchema_RudderStackRealWorld(t *testing.T) {
	t.Parallel()

	// Real-world RudderStack schema example
	input := map[string]interface{}{
		"event":                           "Product Viewed",
		"userId":                          "user123",
		"anonymousId":                     "anon456",
		"context.app.name":                "My Mobile App",
		"context.app.version":             "1.2.3",
		"context.device.type":             "mobile",
		"context.device.model":            "iPhone 12",
		"context.library.name":            "rudder-sdk-ios",
		"context.library.version":         "1.0.0",
		"context.ip":                      "192.168.1.1",
		"context.userAgent":               "Mozilla/5.0...",
		"context.traits.email":            "user@example.com",
		"context.traits.firstName":        "John",
		"context.traits.lastName":         "Doe",
		"properties.product_id":           "prod_123",
		"properties.product_name":         "Premium Widget",
		"properties.price":                99.99,
		"properties.currency":             "USD",
		"properties.quantity":             2,
		"properties.discount":             10.00,
		"properties.categories.0":         "Electronics",
		"properties.categories.1":         "Widgets",
		"properties.custom_data.source":   "mobile_app",
		"properties.custom_data.campaign": "spring_sale",
		"timestamp":                       "2023-05-15T10:30:00Z",
		"sentAt":                          "2023-05-15T10:30:01Z",
		"messageId":                       "msg_789",
		"type":                            "track",
		"integrations.Google Analytics":   true,
		"integrations.Facebook Pixel":     false,
	}

	result := UnflattenSchema(input)

	// Verify basic fields
	assert.Equal(t, "Product Viewed", result["event"])
	assert.Equal(t, "user123", result["userId"])
	assert.Equal(t, "anon456", result["anonymousId"])

	// Verify context structure
	context := result["context"].(map[string]interface{})
	app := context["app"].(map[string]interface{})
	assert.Equal(t, "My Mobile App", app["name"])
	assert.Equal(t, "1.2.3", app["version"])

	device := context["device"].(map[string]interface{})
	assert.Equal(t, "mobile", device["type"])
	assert.Equal(t, "iPhone 12", device["model"])

	traits := context["traits"].(map[string]interface{})
	assert.Equal(t, "user@example.com", traits["email"])
	assert.Equal(t, "John", traits["firstName"])

	// Verify properties structure
	properties := result["properties"].(map[string]interface{})
	assert.Equal(t, "prod_123", properties["product_id"])
	assert.Equal(t, 99.99, properties["price"])

	categories := properties["categories"].([]interface{})
	assert.Len(t, categories, 2)
	assert.Equal(t, "Electronics", categories[0])
	assert.Equal(t, "Widgets", categories[1])

	customData := properties["custom_data"].(map[string]interface{})
	assert.Equal(t, "mobile_app", customData["source"])
	assert.Equal(t, "spring_sale", customData["campaign"])

	// Verify integrations
	integrations := result["integrations"].(map[string]interface{})
	assert.Equal(t, true, integrations["Google Analytics"])
	assert.Equal(t, false, integrations["Facebook Pixel"])
}

func BenchmarkUnflattenLargeArray(b *testing.B) {
	// Create a large array with 1000 elements
	input := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		input[fmt.Sprintf("array.%d", i)] = fmt.Sprintf("value_%d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		UnflattenSchema(input)
	}
}

// TestUnflatten_SuccessfulNestedStructureCreation tests successful nested structure creation
// This covers lines 73-74 in unflatten.go (setNestedValue function creating nested structures)
func TestUnflatten_SuccessfulNestedStructureCreation(t *testing.T) {
	t.Parallel()

	t.Run("SuccessfulObjectCreation", func(t *testing.T) {
		t.Parallel()

		// Test successful creation of nested objects (lines 73-74)
		input := map[string]interface{}{
			"level1.level2.level3.field": "deep_value",
			"level1.level2.other_field":  "shallow_value",
			"level1.simple_field":        "simple_value",
		}

		result := UnflattenSchema(input)

		// Verify the nested structure was created successfully
		expected := map[string]interface{}{
			"level1": map[string]interface{}{
				"level2": map[string]interface{}{
					"level3": map[string]interface{}{
						"field": "deep_value",
					},
					"other_field": "shallow_value",
				},
				"simple_field": "simple_value",
			},
		}

		assert.Equal(t, expected, result)

		// Verify specific nested structure access
		level1, ok := result["level1"].(map[string]interface{})
		require.True(t, ok, "level1 should be a map")

		level2, ok := level1["level2"].(map[string]interface{})
		require.True(t, ok, "level2 should be a map")

		level3, ok := level2["level3"].(map[string]interface{})
		require.True(t, ok, "level3 should be a map")

		assert.Equal(t, "deep_value", level3["field"])
		assert.Equal(t, "shallow_value", level2["other_field"])
		assert.Equal(t, "simple_value", level1["simple_field"])
	})

	t.Run("SuccessfulArrayCreation", func(t *testing.T) {
		t.Parallel()

		// Test successful creation of arrays within nested objects
		input := map[string]interface{}{
			"data.items.0.name":         "item1",
			"data.items.0.properties.0": "prop1",
			"data.items.0.properties.1": "prop2",
			"data.items.1.name":         "item2",
			"data.items.1.properties.0": "prop3",
			"data.metadata.count":       2,
		}

		result := UnflattenSchema(input)

		// Verify nested structure with arrays
		data, ok := result["data"].(map[string]interface{})
		require.True(t, ok, "data should be a map")

		items, ok := data["items"].([]interface{})
		require.True(t, ok, "items should be an array")
		require.Len(t, items, 2)

		// Check first item
		item1, ok := items[0].(map[string]interface{})
		require.True(t, ok, "first item should be a map")
		assert.Equal(t, "item1", item1["name"])

		properties1, ok := item1["properties"].([]interface{})
		require.True(t, ok, "properties should be an array")
		assert.Equal(t, []interface{}{"prop1", "prop2"}, properties1)

		// Check second item
		item2, ok := items[1].(map[string]interface{})
		require.True(t, ok, "second item should be a map")
		assert.Equal(t, "item2", item2["name"])

		properties2, ok := item2["properties"].([]interface{})
		require.True(t, ok, "properties should be an array")
		assert.Equal(t, []interface{}{"prop3"}, properties2)

		// Check metadata
		metadata, ok := data["metadata"].(map[string]interface{})
		require.True(t, ok, "metadata should be a map")
		assert.Equal(t, 2, metadata["count"])
	})

	t.Run("SuccessfulMixedTypeCreation", func(t *testing.T) {
		t.Parallel()

		// Test creation of structures with mixed types
		input := map[string]interface{}{
			"user.id":                        123,
			"user.profile.name":              "John Doe",
			"user.profile.active":            true,
			"user.profile.score":             95.5,
			"user.tags.0":                    "developer",
			"user.tags.1":                    "golang",
			"user.preferences.theme":         "dark",
			"user.preferences.notifications": true,
			"user.metadata.created":          "2024-01-01",
			"user.metadata.updated":          "2024-01-15",
		}

		result := UnflattenSchema(input)

		// Verify all types and structures were created correctly
		user, ok := result["user"].(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, 123, user["id"])

		profile, ok := user["profile"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "John Doe", profile["name"])
		assert.Equal(t, true, profile["active"])
		assert.Equal(t, 95.5, profile["score"])

		tags, ok := user["tags"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, []interface{}{"developer", "golang"}, tags)

		preferences, ok := user["preferences"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "dark", preferences["theme"])
		assert.Equal(t, true, preferences["notifications"])

		metadata, ok := user["metadata"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "2024-01-01", metadata["created"])
		assert.Equal(t, "2024-01-15", metadata["updated"])
	})

	t.Run("SuccessfulComplexStructures", func(t *testing.T) {
		t.Parallel()

		// Test creation of complex nested structures that mirror real-world usage
		input := map[string]interface{}{
			"event":                         "product_purchased",
			"properties.product.id":         "prod_123",
			"properties.product.name":       "Premium Widget",
			"properties.product.category.0": "electronics",
			"properties.product.category.1": "widgets",
			"properties.order.id":           "order_456",
			"properties.order.total":        199.99,
			"properties.order.items.0.sku":  "SKU001",
			"properties.order.items.0.qty":  2,
			"properties.order.items.1.sku":  "SKU002",
			"properties.order.items.1.qty":  1,
			"context.user_agent":            "Mozilla/5.0...",
			"context.page.url":              "https://example.com/checkout",
			"context.page.title":            "Checkout - Example Store",
		}

		result := UnflattenSchema(input)

		// Verify the complex structure
		assert.Equal(t, "product_purchased", result["event"])

		properties, ok := result["properties"].(map[string]interface{})
		require.True(t, ok)

		product, ok := properties["product"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "prod_123", product["id"])
		assert.Equal(t, "Premium Widget", product["name"])

		category, ok := product["category"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, []interface{}{"electronics", "widgets"}, category)

		order, ok := properties["order"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "order_456", order["id"])
		assert.Equal(t, 199.99, order["total"])

		items, ok := order["items"].([]interface{})
		require.True(t, ok)
		require.Len(t, items, 2)

		item1, ok := items[0].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "SKU001", item1["sku"])
		assert.Equal(t, 2, item1["qty"])

		item2, ok := items[1].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "SKU002", item2["sku"])
		assert.Equal(t, 1, item2["qty"])

		context, ok := result["context"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "Mozilla/5.0...", context["user_agent"])

		page, ok := context["page"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "https://example.com/checkout", page["url"])
		assert.Equal(t, "Checkout - Example Store", page["title"])
	})

	t.Run("SuccessfulDynamicStructureBuilding", func(t *testing.T) {
		t.Parallel()

		// Test that structures are built progressively (testing the core setNestedValue logic)
		input := map[string]interface{}{
			"a.b.c": "value1",
			"a.b.d": "value2",
			"a.e.f": "value3",
			"g.h.0": "array_value1",
			"g.h.1": "array_value2",
			"g.i":   "simple_value",
		}

		result := UnflattenSchema(input)

		// Verify that each path was built correctly and structures are shared appropriately
		a, ok := result["a"].(map[string]interface{})
		require.True(t, ok)

		b, ok := a["b"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "value1", b["c"])
		assert.Equal(t, "value2", b["d"])

		e, ok := a["e"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "value3", e["f"])

		g, ok := result["g"].(map[string]interface{})
		require.True(t, ok)

		h, ok := g["h"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, []interface{}{"array_value1", "array_value2"}, h)

		assert.Equal(t, "simple_value", g["i"])
	})
}
