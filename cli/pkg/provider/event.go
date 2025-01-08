package provider

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
	"github.com/rudderlabs/rudder-iac/cli/pkg/provider/state"
)

// TODO: implement on the same lines as the propertyProvider
type eventProvider struct {
	client client.DataCatalog
	log    *logger.Logger
}

func NewEventProvider(client client.DataCatalog) syncer.Provider {
	return &eventProvider{
		client: client,
		log: &logger.Logger{
			Logger: log.With("type", "event"),
		},
	}
}

func (p *eventProvider) Create(ctx context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error) {
	p.log.Debug("creating event in upstream catalog", "id", ID)

	toArgs := state.EventArgs{}
	toArgs.FromResourceData(data)

	event, err := p.client.CreateEvent(ctx, client.EventCreate{
		Name:        toArgs.Name,
		Description: toArgs.Description,
		EventType:   toArgs.EventType,
		CategoryId:  toArgs.CategoryID,
	})

	if err != nil {
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

func (p *eventProvider) Update(ctx context.Context, ID string, resourceType string, input resources.ResourceData, olds resources.ResourceData) (*resources.ResourceData, error) {
	p.log.Debug("updating event in upstream catalog", "id", ID)

	toArgs := state.EventArgs{}
	toArgs.FromResourceData(input)

	prevState := state.EventState{}
	prevState.FromResourceData(olds)

	updatedEvent, err := p.client.UpdateEvent(ctx, prevState.ID, &client.Event{
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

func (p *eventProvider) Delete(ctx context.Context, ID string, resourceType string, state resources.ResourceData) error {
	p.log.Debug("deleting event in upstream catalog", "id", ID)

	err := p.client.DeleteEvent(ctx, state["id"].(string))
	if err != nil {
		return fmt.Errorf("deleting event in upstream catalog: %w", err)
	}

	return nil
}
