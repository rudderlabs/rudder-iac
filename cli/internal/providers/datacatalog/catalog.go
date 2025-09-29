package datacatalog

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	dcstate "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

var resourceTypeCollection = map[string]catalog.ResourceCollection{
	dcstate.PropertyResourceType:     catalog.ResourceCollectionProperties,
	dcstate.EventResourceType:        catalog.ResourceCollectionEvents,
	dcstate.TrackingPlanResourceType: catalog.ResourceCollectionTrackingPlans,
	dcstate.CustomTypeResourceType:   catalog.ResourceCollectionCustomTypes,
	dcstate.CategoryResourceType:     catalog.ResourceCollectionCategories,
}

type entityProvider interface {
	resourceProvider
	resourceImportProvider
}

type resourceImportProvider interface {
	LoadImportable(ctx context.Context) (*resources.ResourceCollection, error)
	IDResources(ctx context.Context, collection *resources.ResourceCollection, idNamer namer.Namer) error
	FormatForExport(
		ctx context.Context,
		collection *resources.ResourceCollection,
		idNamer namer.Namer,
		inputResolver resolver.ReferenceResolver,
	) ([]importremote.FormattableEntity, error)
}

type resourceProvider interface {
	Create(ctx context.Context, ID string, data resources.ResourceData) (*resources.ResourceData, error)
	Update(ctx context.Context, ID string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error)
	Delete(ctx context.Context, ID string, state resources.ResourceData) error
	Import(ctx context.Context, ID string, data resources.ResourceData, remoteId string) (*resources.ResourceData, error)
	LoadResourcesFromRemote(ctx context.Context) (*resources.ResourceCollection, error)
	LoadStateFromResources(ctx context.Context, collection *resources.ResourceCollection) (*state.State, error)
}

func (p *Provider) LoadState(ctx context.Context) (*state.State, error) {
	var apistate *state.State = state.EmptyState()

	cs, err := p.client.ReadState(ctx)
	if err != nil {
		return nil, err
	}

	apistate = &state.State{
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
		apistate.Resources[id] = decodedState
	}

	return apistate, nil
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

func (p *Provider) Delete(ctx context.Context, ID string, resourceType string, data resources.ResourceData) error {
	provider, ok := p.providerStore[resourceType]
	if !ok {
		return fmt.Errorf("unknown resource type: %s", resourceType)
	}
	return provider.Delete(ctx, ID, data)
}

func (p *Provider) Import(ctx context.Context, ID string, resourceType string, data resources.ResourceData, workspaceId, remoteId string) (*resources.ResourceData, error) {
	provider, ok := p.providerStore[resourceType]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
	return provider.Import(ctx, ID, data, remoteId)
}

// LoadResourcesFromRemote loads all resources from remote catalog into a ResourceCollection
func (p *Provider) LoadResourcesFromRemote(ctx context.Context) (*resources.ResourceCollection, error) {
	log.Debug("loading all resources from remote catalog")
	collection := resources.NewResourceCollection()

	// Load resources for stateless resources from provider store
	for resourceType, provider := range p.providerStore {
		c, err := provider.LoadResourcesFromRemote(ctx)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", resourceType, err)
		}

		collection, err = collection.Merge(c)
		if err != nil {
			return nil, err
		}
		log.Debug("loaded resources from remote", "type", resourceType)
	}

	return collection, nil
}

// LoadStateFromResources reconstructs CLI state from loaded remote resources
func (p *Provider) LoadStateFromResources(ctx context.Context, collection *resources.ResourceCollection) (*state.State, error) {
	log.Debug("reconstructing state from loaded resources")
	s := state.EmptyState()

	// loop over stateless resources and load state
	for resourceType, provider := range p.providerStore {
		providerState, err := provider.LoadStateFromResources(ctx, collection)
		if err != nil {
			return nil, fmt.Errorf("LoadStateFromResources: error loading state from provider store %s: %w", resourceType, err)
		}

		s, err = s.Merge(providerState)
		if err != nil {
			return nil, fmt.Errorf("LoadStateFromResources: error merging provider states: %w", err)
		}
	}

	log.Debug("reconstructed state", "resource_count", len(s.Resources))
	return s, nil
}
