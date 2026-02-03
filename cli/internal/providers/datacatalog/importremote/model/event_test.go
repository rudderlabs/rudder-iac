package model

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventForExport(t *testing.T) {
	t.Run("creates map with externalID and basic fields", func(t *testing.T) {
		upstream := &catalog.Event{
			Name:        "Page Viewed",
			Description: "User viewed a page",
			EventType:   "track",
		}

		mockRes := &mockResolver{}
		event := &ImportableEvent{}
		result, err := event.ForExport("page_viewed", upstream, mockRes)

		require.Nil(t, err)
		assert.Equal(t, map[string]any{
			"id":          "page_viewed",
			"name":        "Page Viewed",
			"description": "User viewed a page",
			"event_type":  "track",
		}, result)
	})

	t.Run("resolves category reference when categoryId is set", func(t *testing.T) {
		categoryId := "cat-123"
		categoryRef := "#category:ecommerce_category"
		upstream := &catalog.Event{
			Name:        "Product Purchased",
			Description: "User purchased a product",
			EventType:   "track",
			CategoryId:  &categoryId,
		}

		mockRes := &mockResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				if entityType == types.CategoryResourceType && remoteID == "cat-123" {
					return "#category:ecommerce_category", nil
				}
				return "", nil
			},
		}

		event := &ImportableEvent{}
		result, err := event.ForExport("product_purchased", upstream, mockRes)

		require.Nil(t, err)
		assert.Equal(t, map[string]any{
			"id":          "product_purchased",
			"name":        "Product Purchased",
			"description": "User purchased a product",
			"event_type":  "track",
			"category":    &categoryRef,
		}, result)
	})
}
