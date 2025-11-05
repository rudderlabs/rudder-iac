package provider

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
)

type mockTrackingPlanDataCatalog struct {
	catalog.DataCatalog
	trackingPlans []*catalog.TrackingPlanWithIdentifiers
	err           error
}

func (m *mockTrackingPlanDataCatalog) GetTrackingPlans(ctx context.Context) ([]*catalog.TrackingPlanWithIdentifiers, error) {
	return m.trackingPlans, m.err
}

func TestTrackingPlanLoadImportable(t *testing.T) {
	t.Run("filters tracking plans with ExternalID set", func(t *testing.T) {
		mockClient := &mockTrackingPlanDataCatalog{
			trackingPlans: []*catalog.TrackingPlanWithIdentifiers{
				{ID: "tp1", Name: "Tracking Plan 1", WorkspaceID: "ws1"},
				{ID: "tp2", Name: "Tracking Plan 2", WorkspaceID: "ws1", ExternalID: "tracking-plan-2"},
				{ID: "tp3", Name: "Tracking Plan 3", WorkspaceID: "ws1"},
			},
		}

		provider := &TrackingPlanImportProvider{
			client:        mockClient,
			log:           *logger.New("test"),
			baseImportDir: "data-catalog",
		}

		externalIdNamer := namer.NewExternalIdNamer(namer.NewKebabCase())
		collection, err := provider.LoadImportable(context.Background(), externalIdNamer)
		require.Nil(t, err)

		trackingPlans := collection.GetAll(state.TrackingPlanResourceType)
		assert.Equal(t, 2, len(trackingPlans))

		resourceIDs := make([]string, 0, len(trackingPlans))
		for _, tp := range trackingPlans {
			resourceIDs = append(resourceIDs, tp.ID)
		}

		assert.True(t, lo.Every(resourceIDs, []string{"tp1", "tp3"}))
		assert.False(t, lo.Contains(resourceIDs, "tp2"))
	})

	t.Run("correctly assigns externalId and reference after namer is loaded", func(t *testing.T) {
		mockClient := &mockTrackingPlanDataCatalog{
			trackingPlans: []*catalog.TrackingPlanWithIdentifiers{
				{ID: "tp1", Name: "Mobile Tracking Plan", WorkspaceID: "ws1"},
				{ID: "tp2", Name: "Web Tracking Plan", WorkspaceID: "ws1"},
			},
		}

		provider := &TrackingPlanImportProvider{
			client:        mockClient,
			log:           *logger.New("test"),
			baseImportDir: "data-catalog",
		}

		externalIdNamer := namer.NewExternalIdNamer(namer.NewKebabCase())
		collection, err := provider.LoadImportable(context.Background(), externalIdNamer)
		require.Nil(t, err)

		trackingPlans := collection.GetAll(state.TrackingPlanResourceType)
		require.Equal(t, 2, len(trackingPlans))

		tp1, ok := trackingPlans["tp1"]
		require.True(t, ok)
		assert.NotEmpty(t, tp1.ExternalID)
		assert.NotEmpty(t, tp1.Reference)
		assert.Contains(t, tp1.Reference, tp1.ExternalID)

		tp2, ok := trackingPlans["tp2"]
		require.True(t, ok)
		assert.NotEmpty(t, tp2.ExternalID)
		assert.NotEmpty(t, tp2.Reference)
		assert.Contains(t, tp2.Reference, tp2.ExternalID)
	})
}

func TestTrackingPlanFormatForExport(t *testing.T) {
	t.Run("generates spec with correct relativePath and content structure", func(t *testing.T) {
		mockResolver := &mockResolver{
			references: map[string]map[string]string{
				state.EventResourceType: {
					"evt1": "#/events/product-viewed/product-viewed",
					"evt2": "#/events/checkout-completed/checkout-completed",
				},
				state.PropertyResourceType: {
					"prop1": "#/properties/product-id/product-id",
					"prop2": "#/properties/revenue/revenue",
				},
			},
		}

		mockClient := &mockTrackingPlanDataCatalog{
			trackingPlans: []*catalog.TrackingPlanWithIdentifiers{
				{
					ID:          "tp1",
					Name:        "E-commerce Tracking",
					WorkspaceID: "ws1",
					Events: []*catalog.TrackingPlanEventPropertyIdentifiers{
						{
							ID:   "evt1",
							Name: "Product Viewed",
							Properties: []*catalog.TrackingPlanEventProperty{
								{
									ID:       "prop1",
									Required: true,
								},
							},
							AdditionalProperties: false,
							IdentitySection:      "properties",
						},
					},
				},
			},
		}

		provider := NewTrackingPlanImportProvider(
			mockClient,
			*logger.New("test"),
			"data-catalog",
		)

		externalIdNamer := namer.NewExternalIdNamer(namer.NewKebabCase())
		collection, err := provider.LoadImportable(context.Background(), externalIdNamer)
		require.Nil(t, err)

		result, err := provider.FormatForExport(
			context.Background(),
			collection,
			externalIdNamer,
			mockResolver,
		)
		require.Nil(t, err)
		require.Equal(t, 1, len(result))

		entity := result[0]
		assert.Equal(t, filepath.Join("data-catalog", "trackingplans", "e-commerce-tracking.yaml"), entity.RelativePath)

		spec, ok := entity.Content.(*specs.Spec)
		require.True(t, ok)

		assert.Equal(t, &specs.Spec{
			Version: specs.SpecVersion,
			Kind:    "tp",
			Metadata: map[string]any{
				"name": "e-commerce-tracking",
				"import": map[string]any{
					"workspaces": []importremote.WorkspaceImportMetadata{
						{
							WorkspaceID: "ws1",
							Resources: []importremote.ImportIds{
								{
									LocalID:  "e-commerce-tracking",
									RemoteID: "tp1",
								},
							},
						},
					},
				},
			},
			Spec: map[string]any{
				"id":           "e-commerce-tracking",
				"display_name": "E-commerce Tracking",
				"rules": []any{
					map[string]any{
						"type": "event_rule",
						"id":   "product-viewed-rule",
						"event": map[string]any{
							"$ref":             "#/events/product-viewed/product-viewed",
							"allow_unplanned":  false,
							"identity_section": "properties",
						},
						"properties": []any{
							map[string]any{
								"$ref":     "#/properties/product-id/product-id",
								"required": true,
							},
						},
					},
				},
			},
		}, spec)
	})

	t.Run("generates multiple tracking plans with different specs", func(t *testing.T) {
		mockResolver := &mockResolver{
			references: map[string]map[string]string{
				state.EventResourceType: {
					"evt1": "#/events/product-viewed/product-viewed",
					"evt2": "#/events/user-signup/user-signup",
				},
				state.PropertyResourceType: {
					"prop1": "#/properties/product-id/product-id",
					"prop2": "#/properties/user-email/user-email",
				},
			},
		}

		mockClient := &mockTrackingPlanDataCatalog{
			trackingPlans: []*catalog.TrackingPlanWithIdentifiers{
				{
					ID:          "tp1",
					Name:        "E-commerce Tracking",
					WorkspaceID: "ws1",
					Events: []*catalog.TrackingPlanEventPropertyIdentifiers{
						{
							ID:                   "evt1",
							Name:                 "Product Viewed",
							AdditionalProperties: false,
							IdentitySection:      "properties",
							Properties: []*catalog.TrackingPlanEventProperty{
								{ID: "prop1", Required: true},
							},
						},
					},
				},
				{
					ID:          "tp2",
					Name:        "User Analytics",
					WorkspaceID: "ws2",
					Events: []*catalog.TrackingPlanEventPropertyIdentifiers{
						{
							ID:                   "evt2",
							Name:                 "User Signup",
							AdditionalProperties: true,
							IdentitySection:      "context",
							Properties: []*catalog.TrackingPlanEventProperty{
								{ID: "prop2", Required: false},
							},
						},
					},
				},
			},
		}

		provider := NewTrackingPlanImportProvider(
			mockClient,
			*logger.New("test"),
			"data-catalog",
		)

		externalIdNamer := namer.NewExternalIdNamer(namer.NewKebabCase())
		collection, err := provider.LoadImportable(context.Background(), externalIdNamer)
		require.Nil(t, err)

		result, err := provider.FormatForExport(
			context.Background(),
			collection,
			externalIdNamer,
			mockResolver,
		)
		require.Nil(t, err)
		require.Equal(t, 2, len(result))

		var (
			ecommerceSpec     *specs.Spec
			userAnalyticsSpec *specs.Spec
		)

		for _, entity := range result {
			spec, ok := entity.Content.(*specs.Spec)
			require.True(t, ok)

			if spec.Metadata["name"] == "e-commerce-tracking" {
				ecommerceSpec = spec
			}
			if spec.Metadata["name"] == "user-analytics" {
				userAnalyticsSpec = spec
			}
		}

		assert.Equal(t, &specs.Spec{
			Version: specs.SpecVersion,
			Kind:    "tp",
			Metadata: map[string]any{
				"name": "e-commerce-tracking",
				"import": map[string]any{
					"workspaces": []importremote.WorkspaceImportMetadata{
						{
							WorkspaceID: "ws1",
							Resources: []importremote.ImportIds{
								{
									LocalID:  "e-commerce-tracking",
									RemoteID: "tp1",
								},
							},
						},
					},
				},
			},
			Spec: map[string]any{
				"id":           "e-commerce-tracking",
				"display_name": "E-commerce Tracking",
				"rules": []any{
					map[string]any{
						"type": "event_rule",
						"id":   "product-viewed-rule",
						"event": map[string]any{
							"$ref":             "#/events/product-viewed/product-viewed",
							"allow_unplanned":  false,
							"identity_section": "properties",
						},
						"properties": []any{
							map[string]any{
								"$ref":     "#/properties/product-id/product-id",
								"required": true,
							},
						},
					},
				},
			},
		}, ecommerceSpec)

		assert.Equal(t, &specs.Spec{
			Version: specs.SpecVersion,
			Kind:    "tp",
			Metadata: map[string]any{
				"name": "user-analytics",
				"import": map[string]any{
					"workspaces": []importremote.WorkspaceImportMetadata{
						{
							WorkspaceID: "ws2",
							Resources: []importremote.ImportIds{
								{
									LocalID:  "user-analytics",
									RemoteID: "tp2",
								},
							},
						},
					},
				},
			},
			Spec: map[string]any{
				"id":           "user-analytics",
				"display_name": "User Analytics",
				"rules": []any{
					map[string]any{
						"type": "event_rule",
						"id":   "user-signup-rule",
						"event": map[string]any{
							"$ref":             "#/events/user-signup/user-signup",
							"allow_unplanned":  true,
							"identity_section": "context",
						},
						"properties": []any{
							map[string]any{
								"$ref":     "#/properties/user-email/user-email",
								"required": false,
							},
						},
					},
				},
			},
		}, userAnalyticsSpec)

	})

	t.Run("returns nil for empty tracking plans collection", func(t *testing.T) {
		mockResolver := &mockResolver{
			references: map[string]map[string]string{},
		}

		mockClient := &mockTrackingPlanDataCatalog{
			trackingPlans: []*catalog.TrackingPlanWithIdentifiers{},
		}

		provider := NewTrackingPlanImportProvider(
			mockClient,
			*logger.New("test"),
			"data-catalog",
		)

		externalIdNamer := namer.NewExternalIdNamer(namer.NewKebabCase())
		collection, err := provider.LoadImportable(context.Background(), externalIdNamer)
		require.Nil(t, err)

		result, err := provider.FormatForExport(
			context.Background(),
			collection,
			externalIdNamer,
			mockResolver,
		)
		require.Nil(t, err)
		require.Nil(t, result)
	})
}
