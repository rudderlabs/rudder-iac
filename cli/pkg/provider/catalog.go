package provider

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
)

const (
	PropertyResourceType     = "property"
	EventResourceType        = "event"
	TrackingPlanResourceType = "tracking_plan"
)

var (
	log = logger.New("catalog-provider")
)

type CatalogProvider struct {
	client        client.DataCatalog
	providerStore map[string]syncer.Provider
}

func NewCatalogProvider(dc client.DataCatalog) syncer.Provider {
	return &CatalogProvider{
		client: dc,
		providerStore: map[string]syncer.Provider{
			PropertyResourceType:     NewPropertyProvider(dc),
			EventResourceType:        NewEventProvider(dc),
			TrackingPlanResourceType: NewTrackingPlanProvider(dc),
		},
	}
}

func (p *CatalogProvider) Create(ctx context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error) {
	provider, ok := p.providerStore[resourceType]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
	return provider.Create(ctx, ID, resourceType, data)
}

func (p *CatalogProvider) Update(ctx context.Context, ID string, resourceType string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	provider, ok := p.providerStore[resourceType]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
	return provider.Update(ctx, ID, resourceType, data, state)
}

func (p *CatalogProvider) Delete(ctx context.Context, ID string, resourceType string, state resources.ResourceData) error {
	provider, ok := p.providerStore[resourceType]
	if !ok {
		return fmt.Errorf("unknown resource type: %s", resourceType)
	}
	return provider.Delete(ctx, ID, resourceType, state)
}
