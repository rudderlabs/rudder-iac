package datacatalog

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	impProvider "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/importremote/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	rstate "github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/samber/lo"
)

type EventEntityProvider struct {
	*EventProvider
	*impProvider.EventImportProvider
}

type EventProvider struct {
	catalog catalog.DataCatalog
	log     *logger.Logger
}

func NewEventProvider(dc catalog.DataCatalog, importDir string) *EventEntityProvider {
	ep := &EventProvider{
		catalog: dc,
		log: &logger.Logger{
			Logger: log.With("type", "event"),
		},
	}

	imp := impProvider.NewEventImportProvider(
		dc,
		logger.Logger{
			Logger: logger.New("importremote.provider").With("type", "event"),
		},
		importDir,
	)

	return &EventEntityProvider{
		EventProvider:       ep,
		EventImportProvider: imp,
	}
}

func (p *EventProvider) Create(ctx context.Context, ID string, data resources.ResourceData) (*resources.ResourceData, error) {
	p.log.Debug("creating event in upstream catalog", "id", ID)

	toArgs := state.EventArgs{}
	toArgs.FromResourceData(data)

	// TODO: read categoryID via the new mechanism for reading resovlved property refs
	var categoryId *string
	if cId, ok := data["categoryId"].(string); ok {
		categoryId = &cId
	}

	event, err := p.catalog.CreateEvent(ctx, catalog.EventCreate{
		Name:        toArgs.Name,
		Description: toArgs.Description,
		EventType:   toArgs.EventType,
		CategoryId:  categoryId,
		ExternalId:  ID,
	})

	if err != nil {
		if catalog.IsCatalogAlreadyExistsError(err) {
			p.log.Debug("event already exists in upstream catalog", "error", err)
			switch toArgs.EventType {
			case "track":
				return nil, fmt.Errorf("track event '%s' already exists", toArgs.Name)
			default:
				return nil, fmt.Errorf("%s event already exists", toArgs.EventType)
			}
		}
		return nil, fmt.Errorf("creating event in upstream catalog: %w", err)
	}

	eventState := state.EventState{
		EventArgs:   toArgs,
		ID:          event.ID,
		Name:        event.Name,
		Description: event.Description,
		EventType:   event.EventType,
		WorkspaceID: event.WorkspaceId,
		CategoryID:  event.CategoryId,
		CreatedAt:   event.CreatedAt.String(),
		UpdatedAt:   event.UpdatedAt.String(),
	}

	resourceData := eventState.ToResourceData()

	return &resourceData, nil
}

func (p *EventProvider) Update(ctx context.Context, ID string, input resources.ResourceData, olds resources.ResourceData) (*resources.ResourceData, error) {
	p.log.Debug("updating event in upstream catalog", "id", ID)

	toArgs := state.EventArgs{}
	toArgs.FromResourceData(input)

	prevState := state.EventState{}
	prevState.FromResourceData(olds)

	// TODO: read categoryID via the new mechanism for reading resovlved property refs
	var categoryId *string
	if cId, ok := input["categoryId"].(string); ok {
		categoryId = &cId
	}

	updatedEvent, err := p.catalog.UpdateEvent(ctx, prevState.ID, &catalog.EventUpdate{
		Name:        toArgs.Name,
		Description: toArgs.Description,
		EventType:   toArgs.EventType,
		CategoryId:  categoryId,
	})

	if err != nil {
		return nil, fmt.Errorf("updating event in upstream catalog: %w", err)
	}

	toState := state.EventState{
		EventArgs:   toArgs,
		ID:          updatedEvent.ID,
		Name:        updatedEvent.Name,
		Description: updatedEvent.Description,
		EventType:   updatedEvent.EventType,
		WorkspaceID: updatedEvent.WorkspaceId,
		CategoryID:  updatedEvent.CategoryId,
		CreatedAt:   updatedEvent.CreatedAt.String(),
		UpdatedAt:   updatedEvent.UpdatedAt.String(),
	}

	resourceData := toState.ToResourceData()
	return &resourceData, nil
}

func (p *EventProvider) Delete(ctx context.Context, ID string, state resources.ResourceData) error {
	p.log.Debug("deleting event in upstream catalog", "id", ID)

	remoteID := state["id"].(string)
	err := p.catalog.DeleteEvent(ctx, remoteID)
	if err != nil && !catalog.IsCatalogNotFoundError(err) {
		return fmt.Errorf("deleting event in upstream catalog: %w", err)
	}

	return nil
}

func (p *EventProvider) Import(ctx context.Context, ID string, data resources.ResourceData, remoteId string) (*resources.ResourceData, error) {
	p.log.Debug("importing event resource", "id", ID, "remoteId", remoteId)

	event, err := p.catalog.GetEvent(ctx, remoteId)
	if err != nil {
		return nil, fmt.Errorf("getting event from upstream: %w", err)
	}

	toArgs := state.EventArgs{}
	toArgs.FromResourceData(data)

	if toArgs.DiffUpstream(event) {
		p.log.Debug("event has differences, updating", "id", ID, "remoteId", remoteId)

		var categoryId *string
		if cId, ok := data["categoryId"].(string); ok {
			categoryId = &cId
		}

		event, err = p.catalog.UpdateEvent(ctx, remoteId, &catalog.EventUpdate{
			Name:        toArgs.Name,
			Description: toArgs.Description,
			EventType:   toArgs.EventType,
			CategoryId:  categoryId,
		})
		if err != nil {
			return nil, fmt.Errorf("updating event during import: %w", err)
		}
	}

	err = p.catalog.SetEventExternalId(ctx, remoteId, ID)
	if err != nil {
		return nil, fmt.Errorf("setting event external id: %w", err)
	}

	eventState := state.EventState{
		EventArgs:   toArgs,
		ID:          event.ID,
		Name:        event.Name,
		Description: event.Description,
		EventType:   event.EventType,
		WorkspaceID: event.WorkspaceId,
		CategoryID:  event.CategoryId,
		CreatedAt:   event.CreatedAt.String(),
		UpdatedAt:   event.UpdatedAt.String(),
	}

	resourceData := eventState.ToResourceData()
	return &resourceData, nil
}

// LoadResourcesFromRemote loads all events from the remote catalog
func (p *EventProvider) LoadResourcesFromRemote(ctx context.Context) (*resources.RemoteResources, error) {
	p.log.Debug("loading events from remote catalog")
	collection := resources.NewRemoteResources()

	// fetch events from remote
	events, err := p.catalog.GetEvents(ctx, catalog.ListOptions{HasExternalID: lo.ToPtr(true)})
	if err != nil {
		return nil, err
	}

	// Convert slice to map[string]interface{} where key is the event's remoteId
	resourceMap := make(map[string]*resources.RemoteResource)
	for _, event := range events {
		resourceMap[event.ID] = &resources.RemoteResource{
			ID:         event.ID,
			ExternalID: event.ExternalID,
			Data:       event,
		}
	}
	collection.Set(types.EventResourceType, resourceMap)

	return collection, nil
}

func (p *EventProvider) MapRemoteToState(collection *resources.RemoteResources) (*rstate.State, error) {
	s := rstate.EmptyState()
	events := collection.GetAll(types.EventResourceType)
	for _, remoteEvent := range events {
		if remoteEvent.ExternalID == "" {
			continue
		}
		event, ok := remoteEvent.Data.(*catalog.Event)
		if !ok {
			return nil, fmt.Errorf("MapRemoteToState: unable to cast remote resource to catalog.Event")
		}
		args := &state.EventArgs{}
		args.FromRemoteEvent(event, collection.GetURNByID)

		stateArgs := state.EventState{}
		stateArgs.FromRemoteEvent(event, collection.GetURNByID)

		resourceState := &rstate.ResourceState{
			Type:         types.EventResourceType,
			ID:           event.ExternalID,
			Input:        args.ToResourceData(),
			Output:       stateArgs.ToResourceData(),
			Dependencies: make([]string, 0),
		}

		urn := resources.URN(event.ExternalID, types.EventResourceType)
		s.Resources[urn] = resourceState
	}
	return s, nil
}
