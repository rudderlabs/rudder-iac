package datacatalog

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
)

// TODO: implement on the same lines as the propertyProvider
type EventProvider struct {
	catalog catalog.DataCatalog
	log     *logger.Logger
}

func NewEventProvider(catalog catalog.DataCatalog) *EventProvider {
	return &EventProvider{
		catalog: catalog,
		log: &logger.Logger{
			Logger: log.With("type", "event"),
		},
	}
}

func (p *EventProvider) Create(ctx context.Context, ID string, data resources.ResourceData) (*resources.ResourceData, error) {
	p.log.Debug("creating event in upstream catalog", "id", ID)

	toArgs := state.EventArgs{}
	toArgs.FromResourceData(data)

	event, err := p.catalog.CreateEvent(ctx, catalog.EventCreate{
		Name:        toArgs.Name,
		Description: toArgs.Description,
		EventType:   toArgs.EventType,
		CategoryId:  toArgs.CategoryID,
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

	updatedEvent, err := p.catalog.UpdateEvent(ctx, prevState.ID, &catalog.Event{
		Name:        toArgs.Name,
		Description: toArgs.Description,
		EventType:   toArgs.EventType,
		WorkspaceId: prevState.WorkspaceID,
		CategoryId:  toArgs.CategoryID,
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
