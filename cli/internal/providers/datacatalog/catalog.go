package datacatalog

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

const (
	PropertyResourceType     = "property"
	EventResourceType        = "event"
	TrackingPlanResourceType = "tracking-plan"
	CustomTypeResourceType   = "custom-type"
	CategoryResourceType     = "category"
)

var resourceTypeCollection = map[string]catalog.ResourceCollection{
	PropertyResourceType:     catalog.ResourceCollectionProperties,
	EventResourceType:        catalog.ResourceCollectionEvents,
	TrackingPlanResourceType: catalog.ResourceCollectionTrackingPlans,
	CustomTypeResourceType:   catalog.ResourceCollectionCustomTypes,
	CategoryResourceType:     catalog.ResourceCollectionCategories,
}

type resourceProvider interface {
	Create(ctx context.Context, ID string, data resources.ResourceData) (*resources.ResourceData, error)
	Update(ctx context.Context, ID string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error)
	Delete(ctx context.Context, ID string, state resources.ResourceData) error
}

func (p *Provider) LoadState(ctx context.Context) (*state.State, error) {
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

func (p *Provider) PutResourceState(ctx context.Context, URN string, s *state.ResourceState) error {
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

func (p *Provider) DeleteResourceState(ctx context.Context, s *state.ResourceState) error {
	remoteID := s.Output["id"].(string)
	return p.client.DeleteResourceState(ctx, catalog.DeleteStateRequest{
		Collection: resourceTypeCollection[s.Type],
		ID:         remoteID,
	})
}

func (p *Provider) Create(ctx context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error) {
	provider, ok := p.providerStore[resourceType]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
	return provider.Create(ctx, ID, data)
}

func (p *Provider) Update(ctx context.Context, ID string, resourceType string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	provider, ok := p.providerStore[resourceType]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
	return provider.Update(ctx, ID, data, state)
}

func (p *Provider) Delete(ctx context.Context, ID string, resourceType string, state resources.ResourceData) error {
	provider, ok := p.providerStore[resourceType]
	if !ok {
		return fmt.Errorf("unknown resource type: %s", resourceType)
	}
	return provider.Delete(ctx, ID, state)
}
