package model

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCategoryForExport(t *testing.T) {
	t.Run("creates map with externalID set as id", func(t *testing.T) {
		upstream := &catalog.Category{
			Name: "User Actions",
		}

		mockRes := &mockResolver{}
		category := &ImportableCategoryV1{}
		result, err := category.ForExport("user_actions", upstream, mockRes)

		require.Nil(t, err)
		assert.Equal(t, map[string]any{
			"id":   "user_actions",
			"name": "User Actions",
		}, result)
	})
}

func TestCategoryForExportV0(t *testing.T) {
	t.Run("creates map with externalID set as id", func(t *testing.T) {
		upstream := &catalog.Category{
			Name: "User Actions",
		}

		mockRes := &mockResolver{}
		category := &ImportableCategory{}
		result, err := category.ForExport("user_actions", upstream, mockRes)

		require.Nil(t, err)
		assert.Equal(t, map[string]any{
			"id":   "user_actions",
			"name": "User Actions",
		}, result)
	})
}
