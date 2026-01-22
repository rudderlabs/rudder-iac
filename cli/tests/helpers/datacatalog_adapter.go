package helpers

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/samber/lo"
)

var _ UpstreamAdapter = &DataCatalogAdapter{}

// DataCatalogAdapter wraps the catalog.DataCatalog client
// and implements the UpstreamAdapter interface.
type DataCatalogAdapter struct {
	client catalog.DataCatalog
}

// NewDataCatalogAdapter creates a new DataCatalogAdapter instance
func NewDataCatalogAdapter(client catalog.DataCatalog) *DataCatalogAdapter {
	return &DataCatalogAdapter{
		client: client,
	}
}

func urn(t, id string) string {
	return fmt.Sprintf("%s:%s", t, id)
}

func (a *DataCatalogAdapter) RemoteIDs(ctx context.Context) (map[string]string, error) {
	resourceIDs := make(map[string]string)

	categories, err := a.client.GetCategories(ctx, catalog.ListOptions{HasExternalID: lo.ToPtr(true)})
	if err != nil {
		return nil, err
	}
	for _, category := range categories {
		resourceIDs[urn(state.CategoryResourceType, category.ExternalID)] = category.ID
	}

	events, err := a.client.GetEvents(ctx, catalog.ListOptions{HasExternalID: lo.ToPtr(true)})
	if err != nil {
		return nil, err
	}
	for _, event := range events {
		resourceIDs[urn(state.EventResourceType, event.ExternalID)] = event.ID
	}

	properties, err := a.client.GetProperties(ctx, catalog.ListOptions{HasExternalID: lo.ToPtr(true)})
	if err != nil {
		return nil, err
	}
	for _, property := range properties {
		resourceIDs[urn(state.PropertyResourceType, property.ExternalID)] = property.ID
	}

	customTypes, err := a.client.GetCustomTypes(ctx, catalog.ListOptions{HasExternalID: lo.ToPtr(true)})
	if err != nil {
		return nil, err
	}
	for _, customType := range customTypes {
		resourceIDs[urn(state.CustomTypeResourceType, customType.ExternalID)] = customType.ID
	}

	trackingPlans, err := a.client.GetTrackingPlans(ctx, catalog.ListOptions{HasExternalID: lo.ToPtr(true)})
	if err != nil {
		return nil, err
	}
	for _, trackingPlan := range trackingPlans {
		resourceIDs[urn(state.TrackingPlanResourceType, trackingPlan.ExternalID)] = trackingPlan.ID
	}

	return resourceIDs, nil
}

// FetchResource fetches a datacatalog resource by type and ID
func (a *DataCatalogAdapter) FetchResource(ctx context.Context, resourceType, resourceID string) (any, error) {
	switch resourceType {
	case state.EventResourceType:
		return a.client.GetEvent(ctx, resourceID)
	case state.PropertyResourceType:
		return a.client.GetProperty(ctx, resourceID)
	case state.TrackingPlanResourceType:
		return a.client.GetTrackingPlanWithIdentifiers(ctx, resourceID, false)
	case state.CustomTypeResourceType:
		return a.client.GetCustomType(ctx, resourceID)
	case state.CategoryResourceType:
		return a.client.GetCategory(ctx, resourceID)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}
