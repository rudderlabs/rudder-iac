package model

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
)

type mockTPResolver struct {
	resolveFunc func(entityType string, remoteID string) (string, error)
}

func (m *mockTPResolver) ResolveToReference(entityType string, remoteID string) (string, error) {
	if m.resolveFunc != nil {
		return m.resolveFunc(entityType, remoteID)
	}
	return "", fmt.Errorf("resolver not configured")
}

type mockNamer struct {
	nameFunc func(scope namer.ScopeName) (string, error)
	loadFunc func([]namer.ScopeName) error
}

func (m *mockNamer) Name(scope namer.ScopeName) (string, error) {
	if m.nameFunc != nil {
		return m.nameFunc(scope)
	}
	return "", fmt.Errorf("namer not configured")
}

func (m *mockNamer) Load(scopes []namer.ScopeName) error {
	if m.loadFunc != nil {
		return m.loadFunc(scopes)
	}
	return nil
}

func TestTrackingPlanForExport(t *testing.T) {
	t.Run("creates tracking plan with event rules and properties", func(t *testing.T) {
		desc := "Main tracking plan for e-commerce"
		upstream := &catalog.TrackingPlanWithIdentifiers{
			Name:        "E-commerce Tracking",
			Description: &desc,
			Events: []*catalog.TrackingPlanEventPropertyIdentifiers{
				{
					ID:                   "evt_product_viewed",
					Name:                 "Product Viewed",
					AdditionalProperties: false,
					IdentitySection:      "properties",
					Properties: []*catalog.TrackingPlanEventProperty{
						{ID: "prop_product_id", Required: true},
						{ID: "prop_price", Required: false},
					},
				},
			},
		}

		mockRes := &mockTPResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				switch entityType {
				case state.EventResourceType:
					if remoteID == "evt_product_viewed" {
						return "#/events/ecommerce/product-viewed", nil
					}
				case state.PropertyResourceType:
					switch remoteID {
					case "prop_product_id":
						return "#/properties/products/product-id", nil
					case "prop_price":
						return "#/properties/products/price", nil
					}
				}
				return "", fmt.Errorf("unknown entity: %s/%s", entityType, remoteID)
			},
		}

		mockNamer := &mockNamer{
			nameFunc: func(scope namer.ScopeName) (string, error) {
				assert.Equal(t, TypeEventRule, scope.Scope)
				assert.Equal(t, "Product Viewed Rule", scope.Name)
				return "product-viewed-rule", nil
			},
		}

		tp := &ImportableTrackingPlan{}
		result, err := tp.ForExport("ecommerce-tracking", upstream, mockRes, mockNamer)

		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"id":           "ecommerce-tracking",
			"display_name": "E-commerce Tracking",
			"description":  "Main tracking plan for e-commerce",
			"rules": []any{
				map[string]any{
					"type": TypeEventRule,
					"id":   "product-viewed-rule",
					"event": map[string]any{
						"$ref":             "#/events/ecommerce/product-viewed",
						"allow_unplanned":  false,
						"identity_section": "properties",
					},
					"properties": []any{
						map[string]any{
							"$ref":     "#/properties/products/product-id",
							"required": true,
						},
						map[string]any{
							"$ref":     "#/properties/products/price",
							"required": false,
						},
					},
				},
			},
		}, result)
	})

	t.Run("creates tracking plan with nested properties in rules", func(t *testing.T) {
		upstream := &catalog.TrackingPlanWithIdentifiers{
			Name: "Nested Properties Plan",
			Events: []*catalog.TrackingPlanEventPropertyIdentifiers{
				{
					ID:                   "evt_checkout",
					Name:                 "Checkout Completed",
					AdditionalProperties: true,
					IdentitySection:      "context",
					Properties: []*catalog.TrackingPlanEventProperty{
						{
							ID:       "prop_cart",
							Required: true,
							Properties: []*catalog.TrackingPlanEventProperty{
								{ID: "prop_cart_id", Required: true},
								{ID: "prop_total", Required: false},
							},
						},
					},
				},
			},
		}

		mockRes := &mockTPResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				switch entityType {
				case state.EventResourceType:
					return "#/events/checkout/checkout-completed", nil
				case state.PropertyResourceType:
					refMap := map[string]string{
						"prop_cart":    "#/properties/cart/cart-object",
						"prop_cart_id": "#/properties/cart/cart-id",
						"prop_total":   "#/properties/cart/total-amount",
					}
					if ref, ok := refMap[remoteID]; ok {
						return ref, nil
					}
				}
				return "", fmt.Errorf("unknown entity: %s/%s", entityType, remoteID)
			},
		}

		mockNamer := &mockNamer{
			nameFunc: func(scope namer.ScopeName) (string, error) {
				return "checkout-completed-rule", nil
			},
		}

		tp := &ImportableTrackingPlan{}
		result, err := tp.ForExport("nested-plan", upstream, mockRes, mockNamer)

		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"id":           "nested-plan",
			"display_name": "Nested Properties Plan",
			"rules": []any{
				map[string]any{
					"type": TypeEventRule,
					"id":   "checkout-completed-rule",
					"event": map[string]any{
						"$ref":             "#/events/checkout/checkout-completed",
						"allow_unplanned":  true,
						"identity_section": "context",
					},
					"properties": []any{
						map[string]any{
							"$ref":     "#/properties/cart/cart-object",
							"required": true,
							"properties": []any{
								map[string]any{
									"$ref":     "#/properties/cart/cart-id",
									"required": true,
								},
								map[string]any{
									"$ref":     "#/properties/cart/total-amount",
									"required": false,
								},
							},
						},
					},
				},
			},
		}, result)
	})

	t.Run("creates tracking plan with multiple events", func(t *testing.T) {
		upstream := &catalog.TrackingPlanWithIdentifiers{
			Name: "Multi Event Plan",
			Events: []*catalog.TrackingPlanEventPropertyIdentifiers{
				{
					ID:                   "evt_page_viewed",
					Name:                 "Page Viewed",
					AdditionalProperties: false,
					IdentitySection:      "properties",
					Properties:           []*catalog.TrackingPlanEventProperty{},
				},
				{
					ID:                   "evt_button_clicked",
					Name:                 "Button Clicked",
					AdditionalProperties: true,
					IdentitySection:      "context",
					Properties: []*catalog.TrackingPlanEventProperty{
						{ID: "prop_button_id", Required: true},
					},
				},
			},
		}

		mockRes := &mockTPResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				switch entityType {
				case state.EventResourceType:
					eventMap := map[string]string{
						"evt_page_viewed":    "#/events/navigation/page-viewed",
						"evt_button_clicked": "#/events/interaction/button-clicked",
					}
					if ref, ok := eventMap[remoteID]; ok {
						return ref, nil
					}
				case state.PropertyResourceType:
					if remoteID == "prop_button_id" {
						return "#/properties/ui/button-id", nil
					}
				}
				return "", fmt.Errorf("unknown entity: %s/%s", entityType, remoteID)
			},
		}

		mockNamer := &mockNamer{
			nameFunc: func(scope namer.ScopeName) (string, error) {
				nameMap := map[string]string{
					"Page Viewed Rule":    "page-viewed-rule",
					"Button Clicked Rule": "button-clicked-rule",
				}
				if name, ok := nameMap[scope.Name]; ok {
					return name, nil
				}
				return "", fmt.Errorf("unexpected scope: %s", scope.Name)
			},
		}

		tp := &ImportableTrackingPlan{}
		result, err := tp.ForExport("multi-event-plan", upstream, mockRes, mockNamer)

		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"id":           "multi-event-plan",
			"display_name": "Multi Event Plan",
			"rules": []any{
				map[string]any{
					"type": TypeEventRule,
					"id":   "page-viewed-rule",
					"event": map[string]any{
						"$ref":             "#/events/navigation/page-viewed",
						"allow_unplanned":  false,
						"identity_section": "properties",
					},
				},
				map[string]any{
					"type": TypeEventRule,
					"id":   "button-clicked-rule",
					"event": map[string]any{
						"$ref":             "#/events/interaction/button-clicked",
						"allow_unplanned":  true,
						"identity_section": "context",
					},
					"properties": []any{
						map[string]any{
							"$ref":     "#/properties/ui/button-id",
							"required": true,
						},
					},
				},
			},
		}, result)
	})

	t.Run("omits description when not provided", func(t *testing.T) {
		upstream := &catalog.TrackingPlanWithIdentifiers{
			Name:   "Simple Plan",
			Events: []*catalog.TrackingPlanEventPropertyIdentifiers{},
		}

		mockRes := &mockTPResolver{}
		mockNamer := &mockNamer{}

		tp := &ImportableTrackingPlan{}
		result, err := tp.ForExport("simple-plan", upstream, mockRes, mockNamer)

		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"id":           "simple-plan",
			"display_name": "Simple Plan",
		}, result)
	})

	t.Run("errors when event reference resolution fails", func(t *testing.T) {
		upstream := &catalog.TrackingPlanWithIdentifiers{
			Name: "Error Plan",
			Events: []*catalog.TrackingPlanEventPropertyIdentifiers{
				{
					ID:   "evt_invalid",
					Name: "Invalid Event",
				},
			},
		}

		mockRes := &mockTPResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				return "", fmt.Errorf("event not found")
			},
		}

		mockNamer := &mockNamer{}

		tp := &ImportableTrackingPlan{}
		result, err := tp.ForExport("error-plan", upstream, mockRes, mockNamer)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "event not found")
	})

	t.Run("errors when event reference is empty", func(t *testing.T) {
		upstream := &catalog.TrackingPlanWithIdentifiers{
			Name: "Empty Ref Plan",
			Events: []*catalog.TrackingPlanEventPropertyIdentifiers{
				{
					ID:   "evt_empty",
					Name: "Empty Event",
				},
			},
		}

		mockRes := &mockTPResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				return "", nil
			},
		}

		mockNamer := &mockNamer{}

		tp := &ImportableTrackingPlan{}
		result, err := tp.ForExport("empty-ref-plan", upstream, mockRes, mockNamer)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "resolved reference is empty for event")
	})

	t.Run("errors when property reference resolution fails", func(t *testing.T) {
		upstream := &catalog.TrackingPlanWithIdentifiers{
			Name: "Property Error Plan",
			Events: []*catalog.TrackingPlanEventPropertyIdentifiers{
				{
					ID:   "evt_test",
					Name: "Test Event",
					Properties: []*catalog.TrackingPlanEventProperty{
						{ID: "prop_invalid", Required: true},
					},
				},
			},
		}

		mockRes := &mockTPResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				if entityType == state.EventResourceType {
					return "#/events/test/test-event", nil
				}
				return "", fmt.Errorf("property not found")
			},
		}

		mockNamer := &mockNamer{
			nameFunc: func(scope namer.ScopeName) (string, error) {
				return "test-event-rule", nil
			},
		}

		tp := &ImportableTrackingPlan{}
		result, err := tp.ForExport("prop-error-plan", upstream, mockRes, mockNamer)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "property not found")
	})

	t.Run("errors when property reference is empty", func(t *testing.T) {
		upstream := &catalog.TrackingPlanWithIdentifiers{
			Name: "Property Empty Ref Plan",
			Events: []*catalog.TrackingPlanEventPropertyIdentifiers{
				{
					ID:   "evt_test",
					Name: "Test Event",
					Properties: []*catalog.TrackingPlanEventProperty{
						{ID: "prop_empty", Required: true},
					},
				},
			},
		}

		mockRes := &mockTPResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				if entityType == state.EventResourceType {
					return "#/events/test/test-event", nil
				}
				return "", nil
			},
		}

		mockNamer := &mockNamer{
			nameFunc: func(scope namer.ScopeName) (string, error) {
				return "test-event-rule", nil
			},
		}

		tp := &ImportableTrackingPlan{}
		result, err := tp.ForExport("prop-empty-ref-plan", upstream, mockRes, mockNamer)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "resolved property reference is empty")
	})

	t.Run("errors when nested property reference resolution fails", func(t *testing.T) {
		upstream := &catalog.TrackingPlanWithIdentifiers{
			Name: "Nested Error Plan",
			Events: []*catalog.TrackingPlanEventPropertyIdentifiers{
				{
					ID:   "evt_nested",
					Name: "Nested Event",
					Properties: []*catalog.TrackingPlanEventProperty{
						{
							ID:       "prop_parent",
							Required: true,
							Properties: []*catalog.TrackingPlanEventProperty{
								{ID: "prop_invalid_child", Required: true},
							},
						},
					},
				},
			},
		}

		mockRes := &mockTPResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				if entityType == state.EventResourceType {
					return "#/events/nested/nested-event", nil
				}
				if remoteID == "prop_parent" {
					return "#/properties/parent/parent-prop", nil
				}
				return "", fmt.Errorf("nested property not found")
			},
		}

		mockNamer := &mockNamer{
			nameFunc: func(scope namer.ScopeName) (string, error) {
				return "nested-event-rule", nil
			},
		}

		tp := &ImportableTrackingPlan{}
		result, err := tp.ForExport("nested-error-plan", upstream, mockRes, mockNamer)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "nested property not found")
	})

	t.Run("errors when namer fails to generate rule ID", func(t *testing.T) {
		upstream := &catalog.TrackingPlanWithIdentifiers{
			Name: "Namer Error Plan",
			Events: []*catalog.TrackingPlanEventPropertyIdentifiers{
				{
					ID:   "evt_namer_error",
					Name: "Namer Error Event",
				},
			},
		}

		mockRes := &mockTPResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				return "#/events/test/namer-error-event", nil
			},
		}

		mockNamer := &mockNamer{
			nameFunc: func(scope namer.ScopeName) (string, error) {
				return "", fmt.Errorf("namer failed to generate name")
			},
		}

		tp := &ImportableTrackingPlan{}
		result, err := tp.ForExport("namer-error-plan", upstream, mockRes, mockNamer)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "namer failed to generate name")
	})
}