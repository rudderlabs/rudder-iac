package model

import (
	"fmt"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomTypeForExport(t *testing.T) {
	t.Run("creates map with externalID set as id", func(t *testing.T) {
		upstream := &catalog.CustomType{
			Name:        "Address",
			Description: "User address information",
			Type:        "object",
			Properties: []catalog.CustomTypeProperty{
				{ID: "prop_street_123", Required: true},
				{ID: "prop_city_456", Required: true},
				{ID: "prop_zip_789", Required: false},
			},
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				assert.Equal(t, types.PropertyResourceType, entityType)
				switch remoteID {
				case "prop_street_123":
					return "#property:street", nil
				case "prop_city_456":
					return "#property:city", nil
				case "prop_zip_789":
					return "#property:zip_code", nil
				default:
					return "", fmt.Errorf("unknown property: %s", remoteID)
				}
			},
		}

		ct := &ImportableCustomType{}
		result, err := ct.ForExport("address_type", upstream, mockRes)

		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"id":          "address_type",
			"name":        "Address",
			"description": "User address information",
			"type":        "object",
			"properties": []any{
				map[string]any{
					"property": "#property:street",
					"required": true,
				},
				map[string]any{
					"property": "#property:city",
					"required": true,
				},
				map[string]any{
					"property": "#property:zip_code",
					"required": false,
				},
			},
		}, result)
	})

	t.Run("omits optional fields when empty", func(t *testing.T) {
		upstream := &catalog.CustomType{
			Name:        "Simple CustomType",
			Type:        "object",
			Description: "",
			Config:      map[string]any{},
			Properties:  []catalog.CustomTypeProperty{},
		}

		mockRes := &mockResolver{}
		ct := &ImportableCustomType{}
		result, err := ct.ForExport("simple_ct", upstream, mockRes)

		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"id":   "simple_ct",
			"name": "Simple CustomType",
			"type": "object",
		}, result)
	})

	t.Run("resolves itemTypes custom type references in config", func(t *testing.T) {
		customTypeID := "ct_product_123"
		expectedRef := "#custom-type:Product"

		upstream := &catalog.CustomType{
			Name:        "Product List",
			Description: "Array of products",
			Type:        "array",
			Config: map[string]any{
				"minItems": float64(1),
				"maxItems": float64(5),
			},
			ItemDefinitions: []any{
				map[string]any{
					"id":   customTypeID,
					"type": "object",
					"name": "Product",
				},
			},
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				assert.Equal(t, types.CustomTypeResourceType, entityType)
				assert.Equal(t, customTypeID, remoteID)
				return expectedRef, nil
			},
		}

		ct := &ImportableCustomType{}
		result, err := ct.ForExport("product_list", upstream, mockRes)

		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"id":          "product_list",
			"name":        "Product List",
			"description": "Array of products",
			"type":        "array",
			"config": map[string]any{
				"min_items":  float64(1),
				"max_items":  float64(5),
				"item_types": []any{expectedRef},
			},
		}, result)
	})

	t.Run("resolves variant with discriminator and cases", func(t *testing.T) {
		upstream := &catalog.CustomType{
			Name:        "PageObject",
			Description: "Page object for variants",
			Type:        "object",
			Properties: []catalog.CustomTypeProperty{
				{ID: "prop_page", Required: true},
				{ID: "prop_search_term", Required: false},
				{ID: "prop_product_id", Required: false},
			},
			Variants: []catalog.Variant{
				{
					Type:          "discriminator",
					Discriminator: "prop_page",
					Cases: []catalog.VariantCase{
						{
							DisplayName: "search_page",
							Match:       []any{"search", "search_bar"},
							Description: "applies when a product part of search results",
							Properties: []catalog.PropertyReference{
								{ID: "prop_search_term", Required: true},
							},
						},
						{
							DisplayName: "product_page",
							Match:       []any{"product"},
							Description: "applies on a product page",
							Properties: []catalog.PropertyReference{
								{ID: "prop_page", Required: false},
							},
						},
					},
					Default: []catalog.PropertyReference{
						{ID: "prop_page", Required: true},
					},
				},
			},
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				assert.Equal(t, types.PropertyResourceType, entityType)
				refMap := map[string]string{
					"prop_page":        "#property:page",
					"prop_search_term": "#property:search_term",
					"prop_product_id":  "#property:original_product_id",
				}
				if ref, ok := refMap[remoteID]; ok {
					return ref, nil
				}
				return "", fmt.Errorf("unknown property: %s", remoteID)
			},
		}

		ct := &ImportableCustomType{}
		result, err := ct.ForExport("page-object", upstream, mockRes)

		require.NoError(t, err)

		expected := map[string]any{
			"id":          "page-object",
			"name":        "PageObject",
			"description": "Page object for variants",
			"type":        "object",
			"properties": []any{
				map[string]any{
					"property": "#property:page",
					"required": true,
				},
				map[string]any{
					"property": "#property:search_term",
					"required": false,
				},
				map[string]any{
					"property": "#property:original_product_id",
					"required": false,
				},
			},
			"variants": []any{
				map[string]any{
					"type":          "discriminator",
					"discriminator": "#property:page",
					"cases": []any{
						map[string]any{
							"display_name": "search_page",
							"match":        []any{"search", "search_bar"},
							"description":  "applies when a product part of search results",
							"properties": []any{
								map[string]any{
									"property": "#property:search_term",
									"required": true,
								},
							},
						},
						map[string]any{
							"display_name": "product_page",
							"match":        []any{"product"},
							"description":  "applies on a product page",
							"properties": []any{
								map[string]any{
									"property": "#property:page",
									"required": false,
								},
							},
						},
					},
					"default": map[string]any{
						"properties": []any{
							map[string]any{
								"property": "#property:page",
								"required": true,
							},
						},
					},
				},
			},
		}

		assert.Equal(t, expected, result)
	})

	t.Run("errors when property resolver fails", func(t *testing.T) {
		upstream := &catalog.CustomType{
			Name: "Error CustomType",
			Type: "object",
			Properties: []catalog.CustomTypeProperty{
				{ID: "prop_invalid_123", Required: true},
			},
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				return "", fmt.Errorf("property not found")
			},
		}

		ct := &ImportableCustomType{}
		result, err := ct.ForExport("error_ct", upstream, mockRes)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "property not found")
	})

	t.Run("errors when property resolver returns empty reference", func(t *testing.T) {
		upstream := &catalog.CustomType{
			Name: "Empty Ref CustomType",
			Type: "object",
			Properties: []catalog.CustomTypeProperty{
				{ID: "prop_empty", Required: true},
			},
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				return "", nil
			},
		}

		ct := &ImportableCustomType{}
		result, err := ct.ForExport("empty_ref_ct", upstream, mockRes)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "resolved reference is empty for property")
	})

	t.Run("errors when custom type resolver for itemTypes fails", func(t *testing.T) {
		upstream := &catalog.CustomType{
			Name: "Item Error",
			Type: "array",
			ItemDefinitions: []any{
				map[string]any{
					"id": "ct_invalid_123",
				},
			},
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				return "", fmt.Errorf("custom type not found")
			},
		}

		ct := &ImportableCustomType{}
		result, err := ct.ForExport("item_error", upstream, mockRes)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "custom type not found")
	})

	t.Run("errors when custom type resolver for itemTypes returns empty reference", func(t *testing.T) {
		upstream := &catalog.CustomType{
			Name: "Item Empty Ref",
			Type: "array",
			ItemDefinitions: []any{
				map[string]any{
					"id": "ct_empty_123",
				},
			},
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				return "", nil
			},
		}

		ct := &ImportableCustomType{}
		result, err := ct.ForExport("item_empty_ref", upstream, mockRes)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "resolved reference is empty for item_types")
	})

	t.Run("errors when variant discriminator resolver fails", func(t *testing.T) {
		upstream := &catalog.CustomType{
			Name: "Variant Error",
			Type: "object",
			Variants: []catalog.Variant{
				{
					Type:          "discriminator",
					Discriminator: "prop_invalid_discriminator",
					Cases:         []catalog.VariantCase{},
				},
			},
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				return "", fmt.Errorf("discriminator property not found")
			},
		}

		ct := &ImportableCustomType{}
		result, err := ct.ForExport("variant_error", upstream, mockRes)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "discriminator property not found")
	})

	t.Run("errors when variant discriminator resolver returns empty reference", func(t *testing.T) {
		upstream := &catalog.CustomType{
			Name: "Variant Empty Discriminator",
			Type: "object",
			Variants: []catalog.Variant{
				{
					Type:          "discriminator",
					Discriminator: "prop_discriminator_123",
					Cases:         []catalog.VariantCase{},
				},
			},
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				return "", nil
			},
		}

		ct := &ImportableCustomType{}
		result, err := ct.ForExport("variant_empty_disc", upstream, mockRes)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "resolved reference is empty for discriminator")
	})

	t.Run("errors when variant case property resolver fails", func(t *testing.T) {
		upstream := &catalog.CustomType{
			Name: "Variant Case Error",
			Type: "object",
			Variants: []catalog.Variant{
				{
					Type:          "discriminator",
					Discriminator: "prop_discriminator",
					Cases: []catalog.VariantCase{
						{
							DisplayName: "Case One",
							Match:       []any{"one"},
							Properties: []catalog.PropertyReference{
								{ID: "prop_invalid_456", Required: true},
							},
						},
					},
				},
			},
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				if remoteID == "prop_discriminator" {
					return "#property:discriminator", nil
				}
				return "", fmt.Errorf("case property not found")
			},
		}

		ct := &ImportableCustomType{}
		result, err := ct.ForExport("variant_case_error", upstream, mockRes)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "case property not found")
	})

	t.Run("errors when variant default property resolver fails", func(t *testing.T) {
		upstream := &catalog.CustomType{
			Name: "Variant Default Error",
			Type: "object",
			Variants: []catalog.Variant{
				{
					Type:          "discriminator",
					Discriminator: "prop_discriminator",
					Cases:         []catalog.VariantCase{},
					Default: []catalog.PropertyReference{
						{ID: "prop_invalid", Required: false},
					},
				},
			},
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				if remoteID == "prop_invalid" {
					return "", fmt.Errorf("default property not found")
				}
				return "#property:discriminator", nil
			},
		}

		ct := &ImportableCustomType{}
		result, err := ct.ForExport("variant_default_error", upstream, mockRes)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "default property not found")
	})
}
