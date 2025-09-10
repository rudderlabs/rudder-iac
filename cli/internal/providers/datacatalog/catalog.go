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
	LoadResourcesFromRemote(ctx context.Context) (map[string]interface{}, error)
}

func (p *Provider) LoadState(ctx context.Context) (*state.State, error) {
	var apistate *state.State = state.EmptyState()

	// Load resources and reconstruct state from them
	resources, err := p.LoadResourcesFromRemote(ctx)
	if err != nil {
		return nil, err
	}
	resourcestate, err := p.LoadStateFromResources(ctx, resources)
	if err != nil {
		return nil, err
	}
	_ = resourcestate // TODO: compare rstate and astate for events and categories

	// Load state from API
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
	return nil, fmt.Errorf("import is not supported for %s", resourceType)
}

// LoadResourcesFromRemote loads all resources from remote catalog into a ResourceCollection
func (p *Provider) LoadResourcesFromRemote(ctx context.Context) (*resources.ResourceCollection, error) {
	log.Debug("loading all resources from remote catalog")
	collection := resources.NewResourceCollection()

	// Load resources for each provider store
	for resourceType, provider := range p.providerStore {
		resourceMap, err := provider.LoadResourcesFromRemote(ctx)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", resourceType, err)
		}

		collection.Set(resourceType, resourceMap)
		log.Debug("loaded resources", "type", resourceType, "count", len(resourceMap))
	}

	return collection, nil
}

// LoadStateFromResources reconstructs CLI state from loaded remote resources
func (p *Provider) LoadStateFromResources(ctx context.Context, collection *resources.ResourceCollection) (*state.State, error) {
	log.Debug("reconstructing state from loaded resources")

	s := state.EmptyState()

	// Create URN resolver function that can get URN from remoteId and resourceType
	getURNFromRemoteId := func(resourceType string, remoteId string) string {
		resource, exists := collection.GetById(resourceType, remoteId)
		if !exists {
			return ""
		}

		var projectId string
		switch resourceType {
		case EventResourceType:
			if event, ok := resource.(*catalog.Event); ok {
				projectId = event.ProjectId
			}
		case CategoryResourceType:
			if category, ok := resource.(*catalog.Category); ok {
				projectId = category.ProjectId
			}
		case PropertyResourceType:
			if property, ok := resource.(*catalog.Property); ok {
				projectId = property.ProjectId
			}
		case CustomTypeResourceType:
			if customType, ok := resource.(*catalog.CustomType); ok {
				projectId = customType.ProjectId
			}
		case TrackingPlanResourceType:
			if trackingPlan, ok := resource.(*catalog.TrackingPlan); ok {
				projectId = trackingPlan.ID // TrackingPlan uses ID as projectId
			}
		}

		if projectId == "" {
			return ""
		}

		return resources.URN(projectId, resourceType)
	}

	// Convert events to state
	events := collection.GetAll(EventResourceType)
	for _, eventInterface := range events {
		event, ok := eventInterface.(*catalog.Event)
		if !ok {
			return nil, fmt.Errorf("LoadStateFromResources: unable to cast event to catalog.Event")
		}
		args := &dcstate.EventArgs{}
		args.FromRemoteEvent(event, getURNFromRemoteId)

		stateArgs := dcstate.EventState{}
		stateArgs.FromRemoteEvent(event, getURNFromRemoteId)

		resourceState := &state.ResourceState{
			Type:         EventResourceType,
			ID:           event.ProjectId,
			Input:        args.ToResourceData(),
			Output:       stateArgs.ToResourceData(),
			Dependencies: make([]string, 0),
		}

		urn := resources.URN(event.ProjectId, EventResourceType)
		s.Resources[urn] = resourceState
	}

	// Convert categories to state
	categories := collection.GetAll(CategoryResourceType)
	for _, categoryInterface := range categories {
		category, ok := categoryInterface.(*catalog.Category)
		if !ok {
			return nil, fmt.Errorf("LoadStateFromResources: unable to cast category to catalog.Category")
		}
		args := &dcstate.CategoryArgs{}
		args.FromRemoteCategory(category, getURNFromRemoteId)

		stateArgs := dcstate.CategoryState{}
		stateArgs.FromRemoteCategory(category, getURNFromRemoteId)

		resourceState := &state.ResourceState{
			Type:         CategoryResourceType,
			ID:           category.ProjectId,
			Input:        args.ToResourceData(),
			Output:       stateArgs.ToResourceData(),
			Dependencies: make([]string, 0),
		}

		urn := resources.URN(category.ProjectId, CategoryResourceType)
		s.Resources[urn] = resourceState
	}

	// Convert properties to state
	properties := collection.GetAll(PropertyResourceType)
	for _, propertyInterface := range properties {
		property, ok := propertyInterface.(*catalog.Property)
		if !ok {
			return nil, fmt.Errorf("LoadStateFromResources: unable to cast property to catalog.Property")
		}
		args := &dcstate.PropertyArgs{}
		args.FromRemoteProperty(property, getURNFromRemoteId)

		stateArgs := dcstate.PropertyState{}
		stateArgs.FromRemoteProperty(property, getURNFromRemoteId)

		resourceState := &state.ResourceState{
			Type:         PropertyResourceType,
			ID:           property.ProjectId,
			Input:        args.ToResourceData(),
			Output:       stateArgs.ToResourceData(),
			Dependencies: make([]string, 0),
		}

		urn := resources.URN(property.ProjectId, PropertyResourceType)
		s.Resources[urn] = resourceState
	}

	// Convert custom types to state
	customTypes := collection.GetAll(CustomTypeResourceType)
	for _, customTypeInterface := range customTypes {
		customType, ok := customTypeInterface.(*catalog.CustomType)
		if !ok {
			return nil, fmt.Errorf("LoadStateFromResources: unable to cast custom type to catalog.CustomType")
		}
		args := &dcstate.CustomTypeArgs{}
		args.FromRemoteCustomType(customType, getURNFromRemoteId)

		stateArgs := dcstate.CustomTypeState{}
		stateArgs.FromRemoteCustomType(customType, getURNFromRemoteId)

		resourceState := &state.ResourceState{
			Type:         CustomTypeResourceType,
			ID:           customType.ProjectId,
			Input:        args.ToResourceData(),
			Output:       stateArgs.ToResourceData(),
			Dependencies: make([]string, 0),
		}

		urn := resources.URN(customType.ProjectId, CustomTypeResourceType)
		s.Resources[urn] = resourceState
	}

	// Convert tracking plans to state
	trackingPlans := collection.GetAll(TrackingPlanResourceType)
	for _, trackingPlanInterface := range trackingPlans {
		trackingPlan, ok := trackingPlanInterface.(*catalog.TrackingPlan)
		if !ok {
			return nil, fmt.Errorf("LoadStateFromResources: unable to cast tracking plan to catalog.TrackingPlan")
		}
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
