package datacatalog

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	dcstate "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
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
	LoadResourcesFromRemote(ctx context.Context) (interface{}, error)
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

func (p *Provider) Delete(ctx context.Context, ID string, resourceType string, data resources.ResourceData) error {
	provider, ok := p.providerStore[resourceType]
	if !ok {
		return fmt.Errorf("unknown resource type: %s", resourceType)
	}
	return provider.Delete(ctx, ID, data)
}

func (p *Provider) Import(ctx context.Context, ID string, resourceType string, data resources.ResourceData, workspaceId, remoteId string) (*resources.ResourceData, error) {
	return nil, fmt.Errorf("import is not supported for %s", resourceType)
}

// LoadResourcesFromRemote loads all resources from remote catalog into a ResourceCollection
func (p *Provider) LoadResourcesFromRemote(ctx context.Context) (*resources.ResourceCollection, error) {
	log.Debug("loading all resources from remote catalog")
	collection := resources.NewResourceCollection()

	// Load events
	if provider, ok := p.providerStore[EventResourceType]; ok {
		eventsRaw, err := provider.LoadResourcesFromRemote(ctx)
		if err != nil {
			return nil, fmt.Errorf("loading events: %w", err)
		}
		events := eventsRaw.([]*catalog.Event)
		collection.SetEvents(events)
		log.Debug("loaded events", "count", len(events))
	}

	// Load properties
	if provider, ok := p.providerStore[PropertyResourceType]; ok {
		propertiesRaw, err := provider.LoadResourcesFromRemote(ctx)
		if err != nil {
			return nil, fmt.Errorf("loading properties: %w", err)
		}
		properties := propertiesRaw.([]*catalog.Property)
		collection.SetProperties(properties)
		log.Debug("loaded properties", "count", len(properties))
	}

	// Load categories
	if provider, ok := p.providerStore[CategoryResourceType]; ok {
		categoriesRaw, err := provider.LoadResourcesFromRemote(ctx)
		if err != nil {
			return nil, fmt.Errorf("loading categories: %w", err)
		}
		categories := categoriesRaw.([]*catalog.Category)
		collection.SetCategories(categories)
		log.Debug("loaded categories", "count", len(categories))
	}

	// Load custom types
	if provider, ok := p.providerStore[CustomTypeResourceType]; ok {
		customTypesRaw, err := provider.LoadResourcesFromRemote(ctx)
		if err != nil {
			return nil, fmt.Errorf("loading custom types: %w", err)
		}
		customTypes := customTypesRaw.([]*catalog.CustomType)
		collection.SetCustomTypes(customTypes)
		log.Debug("loaded custom types", "count", len(customTypes))
	}

	// Load tracking plans
	if provider, ok := p.providerStore[TrackingPlanResourceType]; ok {
		trackingPlansRaw, err := provider.LoadResourcesFromRemote(ctx)
		if err != nil {
			return nil, fmt.Errorf("loading tracking plans: %w", err)
		}
		trackingPlans := trackingPlansRaw.([]*catalog.TrackingPlan)
		collection.SetTrackingPlans(trackingPlans)
		log.Debug("loaded tracking plans", "count", len(trackingPlans))
	}

	return collection, nil
}

// LoadStateFromResources reconstructs CLI state from loaded remote resources
func (p *Provider) LoadStateFromResources(ctx context.Context, collection *resources.ResourceCollection) (*state.State, error) {
	log.Debug("reconstructing state from loaded resources")

	s := state.EmptyState()

	// Convert events to state
	for _, event := range collection.GetEvents() {
		args := &dcstate.EventArgs{}
		args.FromRemoteEvent(event, collection)

		stateArgs := dcstate.EventState{}
		stateArgs.FromRemoteEvent(event, collection)

		resourceState := &state.ResourceState{
			Type:         EventResourceType,
			ID:           event.ID,
			Input:        args.ToResourceData(),
			Output:       stateArgs.ToResourceData(),
			Dependencies: make([]string, 0),
		}

		urn := resources.URN(event.ProjectId, EventResourceType)
		s.Resources[urn] = resourceState
	}

	// Convert categories to state
	for _, category := range collection.GetCategories() {
		args := &dcstate.CategoryArgs{}
		args.FromRemoteCategory(category, collection)

		stateArgs := dcstate.CategoryState{}
		stateArgs.FromRemoteCategory(category, collection)

		resourceState := &state.ResourceState{
			Type:         CategoryResourceType,
			ID:           category.ID,
			Input:        args.ToResourceData(),
			Output:       stateArgs.ToResourceData(),
			Dependencies: make([]string, 0),
		}

		urn := resources.URN(category.ProjectId, CategoryResourceType)
		s.Resources[urn] = resourceState
	}

	// Convert properties to state
	for _, property := range collection.GetProperties() {
		args := &dcstate.PropertyArgs{}
		args.FromRemoteProperty(property, collection)

		stateArgs := dcstate.PropertyState{}
		stateArgs.FromRemoteProperty(property, collection)

		resourceState := &state.ResourceState{
			Type:         PropertyResourceType,
			ID:           property.ID,
			Input:        args.ToResourceData(),
			Output:       stateArgs.ToResourceData(),
			Dependencies: make([]string, 0),
		}

		urn := resources.URN(property.ProjectId, PropertyResourceType)
		s.Resources[urn] = resourceState
	}

	// Convert custom types to state
	for _, customType := range collection.GetCustomTypes() {
		args := &dcstate.CustomTypeArgs{}
		args.FromRemoteCustomType(customType, collection)

		stateArgs := dcstate.CustomTypeState{}
		stateArgs.FromRemoteCustomType(customType, collection)

		resourceState := &state.ResourceState{
			Type:         CustomTypeResourceType,
			ID:           customType.ID,
			Input:        args.ToResourceData(),
			Output:       stateArgs.ToResourceData(),
			Dependencies: make([]string, 0),
		}

		urn := resources.URN(customType.ProjectId, CustomTypeResourceType)
		s.Resources[urn] = resourceState
	}

	// Convert tracking plans to state
	for _, trackingPlan := range collection.GetTrackingPlans() {
		args := &dcstate.TrackingPlanArgs{}
		args.FromRemoteTrackingPlan(trackingPlan)

		stateArgs := dcstate.TrackingPlanState{}
		stateArgs.FromRemoteTrackingPlan(trackingPlan)

		resourceState := &state.ResourceState{
			Type:         TrackingPlanResourceType,
			ID:           trackingPlan.ID,
			Input:        args.ToResourceData(),
			Output:       stateArgs.ToResourceData(),
			Dependencies: make([]string, 0),
		}

		urn := resources.URN(trackingPlan.ID, TrackingPlanResourceType)
		s.Resources[urn] = resourceState
	}

	log.Debug("reconstructed state", "resource_count", len(s.Resources))
	return s, nil
}
