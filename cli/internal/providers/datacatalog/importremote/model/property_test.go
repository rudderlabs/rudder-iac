package model

import (
	"fmt"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockResolver struct {
	resolveFunc func(entityType string, remoteID string) (string, error)
}

func (m *mockResolver) ResolveToReference(entityType string, remoteID string) (string, error) {
	if m.resolveFunc != nil {
		return m.resolveFunc(entityType, remoteID)
	}
	return "", fmt.Errorf("resolver not configured")
}

func TestPropertyForExport(t *testing.T) {
	t.Run("creates map with externalID set as id", func(t *testing.T) {
		upstream := &catalog.Property{
			Name:        "User Email",
			Description: "Email address of the user",
			Type:        "string",
			Config: map[string]interface{}{
				"format":    "email",
				"minLength": float64(5),
				"maxLength": float64(255),
			},
		}

		mockRes := &mockResolver{}
		prop := &ImportablePropertyV1{}
		result, err := prop.ForExport("user_email", upstream, mockRes)

		require.Nil(t, err)
		assert.Equal(t, map[string]any{
			"id":          "user_email",
			"name":        "User Email",
			"description": "Email address of the user",
			"type":        "string",
			"config": map[string]interface{}{
				"format":     "email",
				"min_length": float64(5),
				"max_length": float64(255),
			},
		}, result)
	})

	t.Run("omits optional fields when empty", func(t *testing.T) {
		upstream := &catalog.Property{
			Name:        "Simple Property",
			Type:        "boolean",
			Description: "",  // optional field empty
			Config:      nil, // optional field empty
		}

		mockRes := &mockResolver{}
		prop := &ImportablePropertyV1{}
		result, err := prop.ForExport("simple_prop", upstream, mockRes)

		require.Nil(t, err)
		assert.Equal(t, map[string]any{
			"id":   "simple_prop",
			"name": "Simple Property",
			"type": "boolean",
		}, result)
	})

	t.Run("resolves custom type to reference", func(t *testing.T) {
		customTypeID := "ct_abc123xyz"
		expectedRef := "#custom-type:ProductIDType"

		upstream := &catalog.Property{
			Name:        "Product ID",
			Description: "Custom product identifier",
			Type:        customTypeID,
			Config: map[string]interface{}{
				"pattern": "^PROD-[0-9]{7}$",
			},
			DefinitionId: customTypeID,
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				assert.Equal(t, types.CustomTypeResourceType, entityType)
				assert.Equal(t, customTypeID, remoteID)
				return expectedRef, nil
			},
		}

		prop := &ImportablePropertyV1{}
		result, err := prop.ForExport("product_id", upstream, mockRes)

		require.Nil(t, err)
		assert.Equal(t, map[string]any{
			"id":          "product_id",
			"name":        "Product ID",
			"description": "Custom product identifier",
			"type":        expectedRef,
		}, result)
	})

	t.Run("errors when resolver fails", func(t *testing.T) {
		upstream := &catalog.Property{
			Name:         "Failing Property",
			Type:         "ct_invalid_type",
			DefinitionId: "ct_invalid_type",
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				return "", fmt.Errorf("resource not found")
			},
		}

		prop := &ImportablePropertyV1{}
		result, err := prop.ForExport("failing_prop", upstream, mockRes)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "resource not found")
	})

	t.Run("splits multi-type string into types array", func(t *testing.T) {
		upstream := &catalog.Property{
			Name:        "Nullable String",
			Description: "A string that can be null",
			Type:        "string,null",
			Config: map[string]interface{}{
				"minLength": float64(1),
			},
		}

		mockRes := &mockResolver{}
		prop := &ImportablePropertyV1{}
		result, err := prop.ForExport("nullable_string", upstream, mockRes)

		require.Nil(t, err)
		assert.Equal(t, map[string]any{
			"id":          "nullable_string",
			"name":        "Nullable String",
			"description": "A string that can be null",
			"types":       []string{"string", "null"},
			"config": map[string]interface{}{
				"min_length": float64(1),
			},
		}, result)
	})

	t.Run("array with single primitive item type exports item_type not item_types in config", func(t *testing.T) {
		upstream := &catalog.Property{
			Name:   "String List",
			Type:   "array",
			Config: map[string]interface{}{"itemTypes": []interface{}{"string"}},
		}

		mockRes := &mockResolver{}
		prop := &ImportablePropertyV1{}
		result, err := prop.ForExport("string_list", upstream, mockRes)

		require.Nil(t, err)
		assert.Equal(t, "string", result["item_type"])
		assert.NotContains(t, result, "item_types")
		if config, ok := result["config"].(map[string]interface{}); ok {
			assert.NotContains(t, config, "item_types")
		}
	})

	t.Run("array with multiple primitive item types exports item_types at top level", func(t *testing.T) {
		upstream := &catalog.Property{
			Name:   "Multi Type List",
			Type:   "array",
			Config: map[string]interface{}{"itemTypes": []interface{}{"string", "number"}},
		}

		mockRes := &mockResolver{}
		prop := &ImportablePropertyV1{}
		result, err := prop.ForExport("multi_type_list", upstream, mockRes)

		require.Nil(t, err)
		assert.Equal(t, []string{"string", "number"}, result["item_types"])
		assert.NotContains(t, result, "item_type")
		if config, ok := result["config"].(map[string]interface{}); ok {
			assert.NotContains(t, config, "item_types")
		}
	})
}

func TestPropertyForExportV0(t *testing.T) {
	t.Run("creates map with externalID set as id", func(t *testing.T) {
		upstream := &catalog.Property{
			Name:        "User Email",
			Description: "Email address of the user",
			Type:        "string",
			Config: map[string]interface{}{
				"format":    "email",
				"minLength": float64(5),
				"maxLength": float64(255),
			},
		}

		mockRes := &mockResolver{}
		prop := &ImportableProperty{}
		result, err := prop.ForExport("user_email", upstream, mockRes)

		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"id":          "user_email",
			"name":        "User Email",
			"description": "Email address of the user",
			"type":        "string",
			"propConfig": map[string]interface{}{
				"format":    "email",
				"minLength": float64(5),
				"maxLength": float64(255),
			},
		}, result)
	})

	t.Run("omits optional fields when empty", func(t *testing.T) {
		upstream := &catalog.Property{
			Name:        "Simple Property",
			Type:        "boolean",
			Description: "",
			Config:      nil,
		}

		mockRes := &mockResolver{}
		prop := &ImportableProperty{}
		result, err := prop.ForExport("simple_prop", upstream, mockRes)

		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"id":   "simple_prop",
			"name": "Simple Property",
			"type": "boolean",
		}, result)
	})

	t.Run("resolves custom type to reference", func(t *testing.T) {
		customTypeID := "ct_abc123xyz"
		expectedRef := "#/custom-types/default/ProductIDType"

		upstream := &catalog.Property{
			Name:        "Product ID",
			Description: "Custom product identifier",
			Type:        customTypeID,
			Config: map[string]interface{}{
				"pattern": "^PROD-[0-9]{7}$",
			},
			DefinitionId: customTypeID,
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				assert.Equal(t, types.CustomTypeResourceType, entityType)
				assert.Equal(t, customTypeID, remoteID)
				return expectedRef, nil
			},
		}

		prop := &ImportableProperty{}
		result, err := prop.ForExport("product_id", upstream, mockRes)

		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"id":          "product_id",
			"name":        "Product ID",
			"description": "Custom product identifier",
			"type":        expectedRef,
		}, result)
	})

	t.Run("errors when resolver fails", func(t *testing.T) {
		upstream := &catalog.Property{
			Name:         "Failing Property",
			Type:         "ct_invalid_type",
			DefinitionId: "ct_invalid_type",
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				return "", fmt.Errorf("resource not found")
			},
		}

		prop := &ImportableProperty{}
		result, err := prop.ForExport("failing_prop", upstream, mockRes)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "resource not found")
	})

	t.Run("keeps multi-type as single type string in v0", func(t *testing.T) {
		upstream := &catalog.Property{
			Name:        "Nullable String",
			Description: "A string that can be null",
			Type:        "string,null",
			Config: map[string]interface{}{
				"minLength": float64(1),
			},
		}

		mockRes := &mockResolver{}
		prop := &ImportableProperty{}
		result, err := prop.ForExport("nullable_string", upstream, mockRes)

		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"id":          "nullable_string",
			"name":        "Nullable String",
			"description": "A string that can be null",
			"type":        "string,null",
			"propConfig": map[string]interface{}{
				"minLength": float64(1),
			},
		}, result)
	})
}

func TestIsCustomType(t *testing.T) {
	assert.True(t, isCustomType(&catalog.Property{DefinitionId: "someuuid"}))
	assert.False(t, isCustomType(&catalog.Property{DefinitionId: ""}))
}
