package provider

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
)

type mockTrackingPlanDataCatalog struct {
	catalog.DataCatalog
	trackingPlans []*catalog.TrackingPlanWithIdentifiers
	err           error
}

func (m *mockTrackingPlanDataCatalog) GetTrackingPlansWithIdentifiers(ctx context.Context, options catalog.ListOptions) ([]*catalog.TrackingPlanWithIdentifiers, error) {
	return m.trackingPlans, m.err
}

func TestTrackingPlanLoadImportable(t *testing.T) {
	t.Run("filters tracking plans with ExternalID set", func(t *testing.T) {
		mockClient := &mockTrackingPlanDataCatalog{
			trackingPlans: []*catalog.TrackingPlanWithIdentifiers{
				{TrackingPlan: catalog.TrackingPlan{ID: "tp1", Name: "Tracking Plan 1", WorkspaceID: "ws1"}},
				{TrackingPlan: catalog.TrackingPlan{ID: "tp2", Name: "Tracking Plan 2", WorkspaceID: "ws1", ExternalID: "tracking-plan-2"}},
				{TrackingPlan: catalog.TrackingPlan{ID: "tp3", Name: "Tracking Plan 3", WorkspaceID: "ws1"}},
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

		trackingPlans := collection.GetAll(types.TrackingPlanResourceType)
		assert.Equal(t, 2, len(trackingPlans))

		resourceIDs := make([]string, 0, len(trackingPlans))
		for _, tp := range trackingPlans {
			resourceIDs = append(resourceIDs, tp.ID)
		}

		assert.True(t, lo.Every(resourceIDs, []string{"tp1", "tp3"}))
		assert.False(t, lo.Contains(resourceIDs, "tp2"))
	})

	t.Run("correctly assigns externalId and compact reference after namer is loaded", func(t *testing.T) {
		mockClient := &mockTrackingPlanDataCatalog{
			trackingPlans: []*catalog.TrackingPlanWithIdentifiers{
				{TrackingPlan: catalog.TrackingPlan{ID: "tp1", Name: "Mobile Tracking Plan", WorkspaceID: "ws1"}},
				{TrackingPlan: catalog.TrackingPlan{ID: "tp2", Name: "Web Tracking Plan", WorkspaceID: "ws1"}},
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

		trackingPlans := collection.GetAll(types.TrackingPlanResourceType)
		require.Equal(t, 2, len(trackingPlans))

		tp1, ok := trackingPlans["tp1"]
		require.True(t, ok)
		assert.NotEmpty(t, tp1.ExternalID)
		assert.Equal(t, tp1.Reference, fmt.Sprintf("#%s:%s", localcatalog.KindTrackingPlansV1, tp1.ExternalID))

		tp2, ok := trackingPlans["tp2"]
		require.True(t, ok)
		assert.NotEmpty(t, tp2.ExternalID)
		assert.Equal(t, tp2.Reference, fmt.Sprintf("#%s:%s", localcatalog.KindTrackingPlansV1, tp2.ExternalID))
	})
}

func TestTrackingPlanFormatForExport(t *testing.T) {
	t.Run("generates spec with correct relativePath and content structure", func(t *testing.T) {
		mockResolver := &mockResolver{
			references: map[string]map[string]string{
				types.EventResourceType: {
					"evt1": "#event:product-viewed",
					"evt2": "#event:checkout-completed",
				},
				types.PropertyResourceType: {
					"prop1": "#property:product-id",
					"prop2": "#property:revenue",
				},
			},
		}

		mockClient := &mockTrackingPlanDataCatalog{
			trackingPlans: []*catalog.TrackingPlanWithIdentifiers{
				{
					TrackingPlan: catalog.TrackingPlan{ID: "tp1", Name: "E-commerce Tracking", WorkspaceID: "ws1"},
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

		result, _, err := provider.FormatForExport(
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

		assert.Equal(t, specs.SpecVersionV1, spec.Version)
		assert.Equal(t, localcatalog.KindTrackingPlansV1, spec.Kind)
		assert.Equal(t, "e-commerce-tracking", spec.Metadata["name"])

		rules, ok := spec.Spec["rules"].([]any)
		require.True(t, ok)
		require.Len(t, rules, 1)
		rule, ok := rules[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "#event:product-viewed", rule["event"])
		properties, ok := rule["properties"].([]any)
		require.True(t, ok)
		require.Len(t, properties, 1)
		prop, ok := properties[0].(map[string]any)
		require.True(t, ok)
		assert.Contains(t, prop, "property")
	})

	t.Run("generates multiple tracking plans with different specs", func(t *testing.T) {
		mockResolver := &mockResolver{
			references: map[string]map[string]string{
				types.EventResourceType: {
					"evt1": "#event:product-viewed",
					"evt2": "#event:user-signup",
				},
				types.PropertyResourceType: {
					"prop1": "#property:product-id",
					"prop2": "#property:user-email",
				},
			},
		}

		mockClient := &mockTrackingPlanDataCatalog{
			trackingPlans: []*catalog.TrackingPlanWithIdentifiers{
				{
					TrackingPlan: catalog.TrackingPlan{ID: "tp1", Name: "E-commerce Tracking", WorkspaceID: "ws1"},
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
					TrackingPlan: catalog.TrackingPlan{ID: "tp2", Name: "User Analytics", WorkspaceID: "ws2"},
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

		result, _, err := provider.FormatForExport(
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

		require.NotNil(t, ecommerceSpec)
		assert.Equal(t, specs.SpecVersionV1, ecommerceSpec.Version)
		assert.Equal(t, localcatalog.KindTrackingPlansV1, ecommerceSpec.Kind)
		assert.Equal(t, "e-commerce-tracking", ecommerceSpec.Metadata["name"])

		require.NotNil(t, userAnalyticsSpec)
		assert.Equal(t, specs.SpecVersionV1, userAnalyticsSpec.Version)
		assert.Equal(t, localcatalog.KindTrackingPlansV1, userAnalyticsSpec.Kind)
		assert.Equal(t, "user-analytics", userAnalyticsSpec.Metadata["name"])

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

		result, _, err := provider.FormatForExport(
			collection,
			externalIdNamer,
			mockResolver,
		)
		require.Nil(t, err)
		require.Nil(t, result)
	})
}
