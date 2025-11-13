package helpers

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/samber/lo"
)

var _ UpstreamStateReader = &APIClientAdapter{}

// UpstreamStateReader provides an interface for reading state in a raw format.
type UpstreamStateReader interface {
	RemoteIDs(ctx context.Context) (map[string]string, error)
}

// APIClientAdapter wraps the catalog.DataCatalog client
// and implements the UpstreamStateReader interface.
type APIClientAdapter struct {
	client catalog.DataCatalog
}

// NewAPIClientAdapter creates a new APIClientAdapter instance
func NewAPIClientAdapter(client catalog.DataCatalog) *APIClientAdapter {
	return &APIClientAdapter{
		client: client,
	}
}

func urn(t, id string) string {
	return fmt.Sprintf("%s:%s", t, id)
}

func (a *APIClientAdapter) RemoteIDs(ctx context.Context) (map[string]string, error) {
	resourceIDs := make(map[string]string)

	categories, err := a.client.GetCategories(ctx, catalog.ListOptions{HasExternalID: lo.ToPtr(true)})
	if err != nil {
		return nil, err
	}
	for _, category := range categories {
		resourceIDs[urn("category", category.ExternalID)] = category.ID
	}
	// panic("Exit")

	events, err := a.client.GetEvents(ctx, catalog.ListOptions{HasExternalID: lo.ToPtr(true)})
	if err != nil {
		return nil, err
	}
	for _, event := range events {
		resourceIDs[urn("event", event.ExternalID)] = event.ID
	}

	properties, err := a.client.GetProperties(ctx, catalog.ListOptions{HasExternalID: lo.ToPtr(true)})
	if err != nil {
		return nil, err
	}
	for _, property := range properties {
		resourceIDs[urn("property", property.ExternalID)] = property.ID
	}

	customTypes, err := a.client.GetCustomTypes(ctx, catalog.ListOptions{HasExternalID: lo.ToPtr(true)})
	if err != nil {
		return nil, err
	}
	for _, customType := range customTypes {
		resourceIDs[urn("custom-type", customType.ExternalID)] = customType.ID
	}

	trackingPlans, err := a.client.GetTrackingPlans(ctx, catalog.ListOptions{HasExternalID: lo.ToPtr(true)})
	if err != nil {
		return nil, err
	}
	for _, trackingPlan := range trackingPlans {
		resourceIDs[urn("tracking-plan", trackingPlan.ExternalID)] = trackingPlan.ID
	}

	for urn, id := range resourceIDs {
		fmt.Println(urn, id)
	}

	return resourceIDs, nil
}
