package provider

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
)

const (
	PropertyResourceType     = "property"
	EventResourceType        = "event"
	TrackingPlanResourceType = "tracking-plan"
	CustomTypeResourceType   = "custom-type"
)

var (
	log = logger.New("catalog-provider")
)

var resourceTypeCollection = map[string]catalog.ResourceCollection{
	PropertyResourceType:     catalog.ResourceCollectionProperties,
	EventResourceType:        catalog.ResourceCollectionEvents,
	TrackingPlanResourceType: catalog.ResourceCollectionTrackingPlans,
	CustomTypeResourceType:   catalog.ResourceCollectionCustomTypes,
}

type CatalogProvider struct {
	client        catalog.DataCatalog
	providerStore map[string]resourceProvider
}

type resourceProvider interface {
	Create(ctx context.Context, ID string, data resources.ResourceData) (*resources.ResourceData, error)
	Update(ctx context.Context, ID string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error)
	Delete(ctx context.Context, ID string, state resources.ResourceData) error
}

func NewCatalogProvider(dc catalog.DataCatalog) syncer.Provider {
	return &CatalogProvider{
		client: dc,
		providerStore: map[string]resourceProvider{
			PropertyResourceType:     NewPropertyProvider(dc),
			EventResourceType:        NewEventProvider(dc),
			TrackingPlanResourceType: NewTrackingPlanProvider(dc),
			CustomTypeResourceType:   NewCustomTypeProvider(dc),
		},
	}
}

func (p *CatalogProvider) LoadState(ctx context.Context) (*state.State, error) {
	cs, err := p.client.ReadState(ctx)
	if err != nil {
		return nil, err
	}

	s := &state.State{
		Version:   cs.Version,
		Resources: make(map[string]*state.ResourceState),
	}

	for id, rs := range cs.Resources {
		decodedState := state.DecodeResourceState(&state.ResourceState{
			ID:           rs.ID,
			Type:         rs.Type,
			Input:        rs.Input,
			Output:       rs.Output,
			Dependencies: rs.Dependencies,
		})
		s.Resources[id] = decodedState
	}

	return s, nil
}

func (p *CatalogProvider) PutResourceState(ctx context.Context, URN string, s *state.ResourceState) error {
	encodedState := state.EncodeResourceState(s)

	remoteID := s.Output["id"].(string)
	return p.client.PutResourceState(ctx, catalog.PutStateRequest{
		Collection: resourceTypeCollection[s.Type],
		ID:         remoteID,
		URN:        URN,
		State: catalog.ResourceState{
			ID:           encodedState.ID,
			Type:         encodedState.Type,
			Input:        encodedState.Input,
			Output:       encodedState.Output,
			Dependencies: encodedState.Dependencies,
		},
	})
}

func (p *CatalogProvider) DeleteResourceState(ctx context.Context, s *state.ResourceState) error {
	remoteID := s.Output["id"].(string)
	return p.client.DeleteResourceState(ctx, catalog.DeleteStateRequest{
		Collection: resourceTypeCollection[s.Type],
		ID:         remoteID,
	})
}

func (p *CatalogProvider) Create(ctx context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error) {
	provider, ok := p.providerStore[resourceType]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
	return provider.Create(ctx, ID, data)
}

func (p *CatalogProvider) Update(ctx context.Context, ID string, resourceType string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	provider, ok := p.providerStore[resourceType]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
	return provider.Update(ctx, ID, data, state)
}

func (p *CatalogProvider) Delete(ctx context.Context, ID string, resourceType string, state resources.ResourceData) error {
	provider, ok := p.providerStore[resourceType]
	if !ok {
		return fmt.Errorf("unknown resource type: %s", resourceType)
	}
	return provider.Delete(ctx, ID, state)
}
