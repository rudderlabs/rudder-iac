package model

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
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
			TrackingPlan: catalog.TrackingPlan{Name: "E-commerce Tracking", Description: &desc},
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
				case types.EventResourceType:
					if remoteID == "evt_product_viewed" {
						return "#event:product-viewed", nil
					}
				case types.PropertyResourceType:
					switch remoteID {
					case "prop_product_id":
						return "#property:product-id", nil
					case "prop_price":
						return "#property:price", nil
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

		tp := &ImportableTrackingPlanV1{}
		result, err := tp.ForExport("ecommerce-tracking", upstream, mockRes, mockNamer)

		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"id":           "ecommerce-tracking",
			"display_name": "E-commerce Tracking",
			"description":  "Main tracking plan for e-commerce",
			"rules": []any{
				map[string]any{
					"type":             TypeEventRule,
					"id":               "product-viewed-rule",
					"event":            "#event:product-viewed",
					"identity_section": "properties",
					"properties": []any{
						map[string]any{
							"property": "#property:product-id",
							"required": true,
						},
						map[string]any{
							"property": "#property:price",
							"required": false,
						},
					},
				},
			},
		}, result)
	})

	t.Run("creates tracking plan with nested properties in rules", func(t *testing.T) {
		upstream := &catalog.TrackingPlanWithIdentifiers{
			TrackingPlan: catalog.TrackingPlan{Name: "Nested Properties Plan"},
			Events: []*catalog.TrackingPlanEventPropertyIdentifiers{
				{
					ID:                   "evt_checkout",
					Name:                 "Checkout Completed",
					AdditionalProperties: true,
					IdentitySection:      "context",
					Properties: []*catalog.TrackingPlanEventProperty{
						{
							ID:                   "prop_cart",
							Required:             true,
							AdditionalProperties: false,
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
				case types.EventResourceType:
					return "#event:checkout-completed", nil
				case types.PropertyResourceType:
					refMap := map[string]string{
						"prop_cart":    "#property:cart-object",
						"prop_cart_id": "#property:cart-id",
						"prop_total":   "#property:total-amount",
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

		tp := &ImportableTrackingPlanV1{}
		result, err := tp.ForExport("nested-plan", upstream, mockRes, mockNamer)

		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"id":           "nested-plan",
			"display_name": "Nested Properties Plan",
			"rules": []any{
				map[string]any{
					"type":                 TypeEventRule,
					"id":                   "checkout-completed-rule",
					"event":                "#event:checkout-completed",
					"additionalProperties": true,
					"identity_section":     "context",
					"properties": []any{
						map[string]any{
							"property":             "#property:cart-object",
							"required":             true,
							"additionalProperties": false,
							"properties": []any{
								map[string]any{
									"property": "#property:cart-id",
									"required": true,
								},
								map[string]any{
									"property": "#property:total-amount",
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
			TrackingPlan: catalog.TrackingPlan{Name: "Multi Event Plan"},
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
				case types.EventResourceType:
					eventMap := map[string]string{
						"evt_page_viewed":    "#event:page-viewed",
						"evt_button_clicked": "#event:button-clicked",
					}
					if ref, ok := eventMap[remoteID]; ok {
						return ref, nil
					}
				case types.PropertyResourceType:
					if remoteID == "prop_button_id" {
						return "#property:button-id", nil
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

		tp := &ImportableTrackingPlanV1{}
		result, err := tp.ForExport("multi-event-plan", upstream, mockRes, mockNamer)

		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"id":           "multi-event-plan",
			"display_name": "Multi Event Plan",
			"rules": []any{
				map[string]any{
					"type":             TypeEventRule,
					"id":               "page-viewed-rule",
					"event":            "#event:page-viewed",
					"identity_section": "properties",
				},
				map[string]any{
					"type":                 TypeEventRule,
					"id":                   "button-clicked-rule",
					"event":                "#event:button-clicked",
					"additionalProperties": true,
					"identity_section":     "context",
					"properties": []any{
						map[string]any{
							"property": "#property:button-id",
							"required": true,
						},
					},
				},
			},
		}, result)
	})

	t.Run("omits description when not provided", func(t *testing.T) {
		upstream := &catalog.TrackingPlanWithIdentifiers{
			TrackingPlan: catalog.TrackingPlan{Name: "Simple Plan"},
			Events:       []*catalog.TrackingPlanEventPropertyIdentifiers{},
		}

		mockRes := &mockTPResolver{}
		mockNamer := &mockNamer{}

		tp := &ImportableTrackingPlanV1{}
		result, err := tp.ForExport("simple-plan", upstream, mockRes, mockNamer)

		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"id":           "simple-plan",
			"display_name": "Simple Plan",
		}, result)
	})

	t.Run("errors when event reference resolution fails", func(t *testing.T) {
		upstream := &catalog.TrackingPlanWithIdentifiers{
			TrackingPlan: catalog.TrackingPlan{Name: "Error Plan"},
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

		tp := &ImportableTrackingPlanV1{}
		result, err := tp.ForExport("error-plan", upstream, mockRes, mockNamer)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "event not found")
	})

	t.Run("errors when event reference is empty", func(t *testing.T) {
		upstream := &catalog.TrackingPlanWithIdentifiers{
			TrackingPlan: catalog.TrackingPlan{Name: "Empty Ref Plan"},
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

		tp := &ImportableTrackingPlanV1{}
		result, err := tp.ForExport("empty-ref-plan", upstream, mockRes, mockNamer)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "resolved reference is empty for event")
	})

	t.Run("errors when property reference resolution fails", func(t *testing.T) {
		upstream := &catalog.TrackingPlanWithIdentifiers{
			TrackingPlan: catalog.TrackingPlan{Name: "Property Error Plan"},
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
				if entityType == types.EventResourceType {
					return "#event:test-event", nil
				}
				return "", fmt.Errorf("property not found")
			},
		}

		mockNamer := &mockNamer{
			nameFunc: func(scope namer.ScopeName) (string, error) {
				return "test-event-rule", nil
			},
		}

		tp := &ImportableTrackingPlanV1{}
		result, err := tp.ForExport("prop-error-plan", upstream, mockRes, mockNamer)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "property not found")
	})

	t.Run("errors when property reference is empty", func(t *testing.T) {
		upstream := &catalog.TrackingPlanWithIdentifiers{
			TrackingPlan: catalog.TrackingPlan{Name: "Property Empty Ref Plan"},
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
				if entityType == types.EventResourceType {
					return "#event:test-event", nil
				}
				return "", nil
			},
		}

		mockNamer := &mockNamer{
			nameFunc: func(scope namer.ScopeName) (string, error) {
				return "test-event-rule", nil
			},
		}

		tp := &ImportableTrackingPlanV1{}
		result, err := tp.ForExport("prop-empty-ref-plan", upstream, mockRes, mockNamer)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "resolved property reference is empty")
	})

	t.Run("errors when nested property reference resolution fails", func(t *testing.T) {
		upstream := &catalog.TrackingPlanWithIdentifiers{
			TrackingPlan: catalog.TrackingPlan{Name: "Nested Error Plan"},
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
				if entityType == types.EventResourceType {
					return "#event:nested-event", nil
				}
				if remoteID == "prop_parent" {
					return "#property:parent-prop", nil
				}
				return "", fmt.Errorf("nested property not found")
			},
		}

		mockNamer := &mockNamer{
			nameFunc: func(scope namer.ScopeName) (string, error) {
				return "nested-event-rule", nil
			},
		}

		tp := &ImportableTrackingPlanV1{}
		result, err := tp.ForExport("nested-error-plan", upstream, mockRes, mockNamer)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "nested property not found")
	})

	t.Run("errors when namer fails to generate rule ID", func(t *testing.T) {
		upstream := &catalog.TrackingPlanWithIdentifiers{
			TrackingPlan: catalog.TrackingPlan{Name: "Namer Error Plan"},
			Events: []*catalog.TrackingPlanEventPropertyIdentifiers{
				{
					ID:   "evt_namer_error",
					Name: "Namer Error Event",
				},
			},
		}

		mockRes := &mockTPResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				return "#event:namer-error-event", nil
			},
		}

		mockNamer := &mockNamer{
			nameFunc: func(scope namer.ScopeName) (string, error) {
				return "", fmt.Errorf("namer failed to generate name")
			},
		}

		tp := &ImportableTrackingPlanV1{}
		result, err := tp.ForExport("namer-error-plan", upstream, mockRes, mockNamer)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "namer failed to generate name")
	})

	t.Run("creates tracking plan with variants including discriminator in properties", func(t *testing.T) {
		desc := "Tracking plan with conditional variants"
		upstream := &catalog.TrackingPlanWithIdentifiers{
			TrackingPlan: catalog.TrackingPlan{Name: "Variant Based Tracking", Description: &desc},
			Events: []*catalog.TrackingPlanEventPropertyIdentifiers{
				{
					ID:                   "evt_user_action",
					Name:                 "User Action",
					AdditionalProperties: false,
					IdentitySection:      "properties",
					Properties: []*catalog.TrackingPlanEventProperty{
						{ID: "prop_action_type", Required: true},
						{ID: "prop_user_id", Required: true},
						{ID: "prop_timestamp", Required: true},
					},
					Variants: []catalog.Variant{
						{
							Type:          "discriminator",
							Discriminator: "prop_action_type",
							Cases: []catalog.VariantCase{
								{
									DisplayName: "Sign Up Action",
									Match:       []any{"signup"},
									Description: "Properties specific to signup actions",
									Properties: []catalog.PropertyReference{
										{ID: "prop_email", Required: true},
										{ID: "prop_referral_code", Required: false},
									},
								},
								{
									DisplayName: "Purchase Action",
									Match:       []any{"purchase", "buy"},
									Description: "Properties specific to purchase actions",
									Properties: []catalog.PropertyReference{
										{ID: "prop_product_id", Required: true},
										{ID: "prop_amount", Required: true},
										{ID: "prop_currency", Required: false},
									},
								},
							},
							Default: []catalog.PropertyReference{
								{ID: "prop_session_id", Required: true},
								{ID: "prop_device_type", Required: false},
							},
						},
					},
				},
			},
		}

		mockRes := &mockTPResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				switch entityType {
				case types.EventResourceType:
					if remoteID == "evt_user_action" {
						return "#event:user-action", nil
					}
				case types.PropertyResourceType:
					refMap := map[string]string{
						"prop_action_type":   "#property:action-type",
						"prop_user_id":       "#property:user-id",
						"prop_timestamp":     "#property:timestamp",
						"prop_email":         "#property:email",
						"prop_referral_code": "#property:referral-code",
						"prop_product_id":    "#property:product-id",
						"prop_amount":        "#property:amount",
						"prop_currency":      "#property:currency",
						"prop_session_id":    "#property:session-id",
						"prop_device_type":   "#property:device-type",
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
				assert.Equal(t, TypeEventRule, scope.Scope)
				assert.Equal(t, "User Action Rule", scope.Name)
				return "user-action-rule", nil
			},
		}

		tp := &ImportableTrackingPlanV1{}
		result, err := tp.ForExport("variant-tracking", upstream, mockRes, mockNamer)

		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"id":           "variant-tracking",
			"display_name": "Variant Based Tracking",
			"description":  "Tracking plan with conditional variants",
			"rules": []any{
				map[string]any{
					"type":             TypeEventRule,
					"id":               "user-action-rule",
					"event":            "#event:user-action",
					"identity_section": "properties",
					"properties": []any{
						map[string]any{
							"property": "#property:action-type",
							"required": true,
						},
						map[string]any{
							"property": "#property:user-id",
							"required": true,
						},
						map[string]any{
							"property": "#property:timestamp",
							"required": true,
						},
					},
					"variants": []any{
						map[string]any{
							"type":          "discriminator",
							"discriminator": "#property:action-type",
							"cases": []any{
								map[string]any{
									"display_name": "Sign Up Action",
									"match":        []any{"signup"},
									"description":  "Properties specific to signup actions",
									"properties": []any{
										map[string]any{
											"property": "#property:email",
											"required": true,
										},
										map[string]any{
											"property": "#property:referral-code",
											"required": false,
										},
									},
								},
								map[string]any{
									"display_name": "Purchase Action",
									"match":        []any{"purchase", "buy"},
									"description":  "Properties specific to purchase actions",
									"properties": []any{
										map[string]any{
											"property": "#property:product-id",
											"required": true,
										},
										map[string]any{
											"property": "#property:amount",
											"required": true,
										},
										map[string]any{
											"property": "#property:currency",
											"required": false,
										},
									},
								},
							},
							"default": map[string]any{
								"properties": []any{
									map[string]any{
										"property": "#property:session-id",
										"required": true,
									},
									map[string]any{
										"property": "#property:device-type",
										"required": false,
									},
								},
							},
						},
					},
				},
			},
		}, result)
	})
}

func TestTrackingPlanForExportV0(t *testing.T) {
	t.Run("creates tracking plan with event rules and properties", func(t *testing.T) {
		desc := "Main tracking plan for e-commerce"
		upstream := &catalog.TrackingPlanWithIdentifiers{
			TrackingPlan: catalog.TrackingPlan{Name: "E-commerce Tracking", Description: &desc},
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
				case types.EventResourceType:
					if remoteID == "evt_product_viewed" {
						return "#/events/default/product-viewed", nil
					}
				case types.PropertyResourceType:
					switch remoteID {
					case "prop_product_id":
						return "#/properties/default/product-id", nil
					case "prop_price":
						return "#/properties/default/price", nil
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
					"type": "event_rule",
					"id":   "product-viewed-rule",
					"event": map[string]any{
						"$ref":             "#/events/default/product-viewed",
						"allow_unplanned":  false,
						"identity_section": "properties",
					},
					"properties": []any{
						map[string]any{
							"$ref":     "#/properties/default/product-id",
							"required": true,
						},
						map[string]any{
							"$ref":     "#/properties/default/price",
							"required": false,
						},
					},
				},
			},
		}, result)
	})

	t.Run("creates tracking plan with nested properties in rules", func(t *testing.T) {
		upstream := &catalog.TrackingPlanWithIdentifiers{
			TrackingPlan: catalog.TrackingPlan{Name: "Nested Properties Plan"},
			Events: []*catalog.TrackingPlanEventPropertyIdentifiers{
				{
					ID:                   "evt_checkout",
					Name:                 "Checkout Completed",
					AdditionalProperties: true,
					IdentitySection:      "context",
					Properties: []*catalog.TrackingPlanEventProperty{
						{
							ID:                   "prop_cart",
							Required:             true,
							AdditionalProperties: false,
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
				case types.EventResourceType:
					return "#/events/default/checkout-completed", nil
				case types.PropertyResourceType:
					refMap := map[string]string{
						"prop_cart":    "#/properties/default/cart-object",
						"prop_cart_id": "#/properties/default/cart-id",
						"prop_total":   "#/properties/default/total-amount",
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
					"type": "event_rule",
					"id":   "checkout-completed-rule",
					"event": map[string]any{
						"$ref":             "#/events/default/checkout-completed",
						"allow_unplanned":  true,
						"identity_section": "context",
					},
					"properties": []any{
						map[string]any{
							"$ref":                 "#/properties/default/cart-object",
							"required":             true,
							"additionalProperties": false,
							"properties": []any{
								map[string]any{
									"$ref":     "#/properties/default/cart-id",
									"required": true,
								},
								map[string]any{
									"$ref":     "#/properties/default/total-amount",
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
			TrackingPlan: catalog.TrackingPlan{Name: "Multi Event Plan"},
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
				case types.EventResourceType:
					eventMap := map[string]string{
						"evt_page_viewed":    "#/events/default/page-viewed",
						"evt_button_clicked": "#/events/default/button-clicked",
					}
					if ref, ok := eventMap[remoteID]; ok {
						return ref, nil
					}
				case types.PropertyResourceType:
					if remoteID == "prop_button_id" {
						return "#/properties/default/button-id", nil
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
					"type": "event_rule",
					"id":   "page-viewed-rule",
					"event": map[string]any{
						"$ref":             "#/events/default/page-viewed",
						"allow_unplanned":  false,
						"identity_section": "properties",
					},
				},
				map[string]any{
					"type": "event_rule",
					"id":   "button-clicked-rule",
					"event": map[string]any{
						"$ref":             "#/events/default/button-clicked",
						"allow_unplanned":  true,
						"identity_section": "context",
					},
					"properties": []any{
						map[string]any{
							"$ref":     "#/properties/default/button-id",
							"required": true,
						},
					},
				},
			},
		}, result)
	})

	t.Run("omits description when not provided", func(t *testing.T) {
		upstream := &catalog.TrackingPlanWithIdentifiers{
			TrackingPlan: catalog.TrackingPlan{Name: "Simple Plan"},
			Events:       []*catalog.TrackingPlanEventPropertyIdentifiers{},
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
			TrackingPlan: catalog.TrackingPlan{Name: "Error Plan"},
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
			TrackingPlan: catalog.TrackingPlan{Name: "Empty Ref Plan"},
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
			TrackingPlan: catalog.TrackingPlan{Name: "Property Error Plan"},
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
				if entityType == types.EventResourceType {
					return "#/events/default/test-event", nil
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
			TrackingPlan: catalog.TrackingPlan{Name: "Property Empty Ref Plan"},
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
				if entityType == types.EventResourceType {
					return "#/events/default/test-event", nil
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
			TrackingPlan: catalog.TrackingPlan{Name: "Nested Error Plan"},
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
				if entityType == types.EventResourceType {
					return "#/events/default/nested-event", nil
				}
				if remoteID == "prop_parent" {
					return "#/properties/default/parent-prop", nil
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
			TrackingPlan: catalog.TrackingPlan{Name: "Namer Error Plan"},
			Events: []*catalog.TrackingPlanEventPropertyIdentifiers{
				{
					ID:   "evt_namer_error",
					Name: "Namer Error Event",
				},
			},
		}

		mockRes := &mockTPResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				return "#/events/default/namer-error-event", nil
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

	t.Run("creates tracking plan with variants including discriminator in properties", func(t *testing.T) {
		desc := "Tracking plan with conditional variants"
		upstream := &catalog.TrackingPlanWithIdentifiers{
			TrackingPlan: catalog.TrackingPlan{Name: "Variant Based Tracking", Description: &desc},
			Events: []*catalog.TrackingPlanEventPropertyIdentifiers{
				{
					ID:                   "evt_user_action",
					Name:                 "User Action",
					AdditionalProperties: false,
					IdentitySection:      "properties",
					Properties: []*catalog.TrackingPlanEventProperty{
						{ID: "prop_action_type", Required: true},
						{ID: "prop_user_id", Required: true},
						{ID: "prop_timestamp", Required: true},
					},
					Variants: []catalog.Variant{
						{
							Type:          "discriminator",
							Discriminator: "prop_action_type",
							Cases: []catalog.VariantCase{
								{
									DisplayName: "Sign Up Action",
									Match:       []any{"signup"},
									Description: "Properties specific to signup actions",
									Properties: []catalog.PropertyReference{
										{ID: "prop_email", Required: true},
										{ID: "prop_referral_code", Required: false},
									},
								},
								{
									DisplayName: "Purchase Action",
									Match:       []any{"purchase", "buy"},
									Description: "Properties specific to purchase actions",
									Properties: []catalog.PropertyReference{
										{ID: "prop_product_id", Required: true},
										{ID: "prop_amount", Required: true},
										{ID: "prop_currency", Required: false},
									},
								},
							},
							Default: []catalog.PropertyReference{
								{ID: "prop_session_id", Required: true},
								{ID: "prop_device_type", Required: false},
							},
						},
					},
				},
			},
		}

		mockRes := &mockTPResolver{
			resolveFunc: func(entityType string, remoteID string) (string, error) {
				switch entityType {
				case types.EventResourceType:
					if remoteID == "evt_user_action" {
						return "#/events/default/user-action", nil
					}
				case types.PropertyResourceType:
					refMap := map[string]string{
						"prop_action_type":   "#/properties/default/action-type",
						"prop_user_id":       "#/properties/default/user-id",
						"prop_timestamp":     "#/properties/default/timestamp",
						"prop_email":         "#/properties/default/email",
						"prop_referral_code": "#/properties/default/referral-code",
						"prop_product_id":    "#/properties/default/product-id",
						"prop_amount":        "#/properties/default/amount",
						"prop_currency":      "#/properties/default/currency",
						"prop_session_id":    "#/properties/default/session-id",
						"prop_device_type":   "#/properties/default/device-type",
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
				assert.Equal(t, TypeEventRule, scope.Scope)
				assert.Equal(t, "User Action Rule", scope.Name)
				return "user-action-rule", nil
			},
		}

		tp := &ImportableTrackingPlan{}
		result, err := tp.ForExport("variant-tracking", upstream, mockRes, mockNamer)

		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"id":           "variant-tracking",
			"display_name": "Variant Based Tracking",
			"description":  "Tracking plan with conditional variants",
			"rules": []any{
				map[string]any{
					"type": "event_rule",
					"id":   "user-action-rule",
					"event": map[string]any{
						"$ref":             "#/events/default/user-action",
						"allow_unplanned":  false,
						"identity_section": "properties",
					},
					"properties": []any{
						map[string]any{
							"$ref":     "#/properties/default/action-type",
							"required": true,
						},
						map[string]any{
							"$ref":     "#/properties/default/user-id",
							"required": true,
						},
						map[string]any{
							"$ref":     "#/properties/default/timestamp",
							"required": true,
						},
					},
					"variants": []any{
						map[string]any{
							"type":          "discriminator",
							"discriminator": "#/properties/default/action-type",
							"cases": []any{
								map[string]any{
									"display_name": "Sign Up Action",
									"match":        []any{"signup"},
									"description":  "Properties specific to signup actions",
									"properties": []any{
										map[string]any{
											"$ref":     "#/properties/default/email",
											"required": true,
										},
										map[string]any{
											"$ref":     "#/properties/default/referral-code",
											"required": false,
										},
									},
								},
								map[string]any{
									"display_name": "Purchase Action",
									"match":        []any{"purchase", "buy"},
									"description":  "Properties specific to purchase actions",
									"properties": []any{
										map[string]any{
											"$ref":     "#/properties/default/product-id",
											"required": true,
										},
										map[string]any{
											"$ref":     "#/properties/default/amount",
											"required": true,
										},
										map[string]any{
											"$ref":     "#/properties/default/currency",
											"required": false,
										},
									},
								},
							},
							"default": []any{
								map[string]any{
									"$ref":     "#/properties/default/session-id",
									"required": true,
								},
								map[string]any{
									"$ref":     "#/properties/default/device-type",
									"required": false,
								},
							},
						},
					},
				},
			},
		}, result)
	})
}
