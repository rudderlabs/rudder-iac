package provider

import (
	"context"
	"fmt"
	"time"

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

func newEventProvider(client client.DataCatalog) syncer.Provider {
	return &eventProvider{
		client: client,
		log: &logger.Logger{
			Logger: log.With("type", "event"),
		},
	}
}

func (p *eventProvider) toResourceData(event *client.Event) *resources.ResourceData {

	resp := resources.ResourceData{
		"id":           event.ID,
		"display_name": event.Name,
		"description":  event.Description,
		"event_type":   event.EventType,
		"workspaceId":  event.WorkspaceId,
		"categoryId":   event.CategoryId,
		"created_at":   event.CreatedAt.UTC().String(),
		"updated_at":   event.UpdatedAt.UTC().String(),
	}

	return &resp
}

func (p *eventProvider) fromResourceData(data resources.ResourceData) *client.Event {
	return &client.Event{
		ID:          data["id"].(string),
		Name:        data["display_name"].(string),
		Description: data["description"].(string),
		EventType:   data["event_type"].(string),
		WorkspaceId: data["workspaceId"].(string),
		CategoryId:  data["categoryId"].(*string),
		UpdatedAt:   time.Now().UTC(),
	}
}

func (p *eventProvider) Create(ctx context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error) {
	p.log.Debug("creating event in upstream catalog", "id", ID)

	toArgs := state.EventArgs{}
	toArgs.FromResourceData(&data)

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
		ID:          event.ID,
		Name:        event.Name,
		Description: event.Description,
		EventType:   event.EventType,
		WorkspaceID: event.WorkspaceId,
		CategoryID:  event.CategoryId,
		CreatedAt:   event.CreatedAt.UTC().String(),
		UpdatedAt:   event.UpdatedAt.UTC().String(),
	}

	return eventState.ToResourceData(), nil
}

func (p *eventProvider) Update(ctx context.Context, ID string, resourceType string, input resources.ResourceData, olds resources.ResourceData) (*resources.ResourceData, error) {
	p.log.Debug("updating event in upstream catalog", "id", ID)

	toArgs := state.EventArgs{}
	toArgs.FromResourceData(&input)

	prevState := state.EventState{}
	prevState.FromResourceData(&olds)

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
		ID:          updatedEvent.ID,
		Name:        updatedEvent.Name,
		Description: updatedEvent.Description,
		EventType:   updatedEvent.EventType,
		WorkspaceID: updatedEvent.WorkspaceId,
		CategoryID:  updatedEvent.CategoryId,
		CreatedAt:   updatedEvent.CreatedAt.String(),
		UpdatedAt:   updatedEvent.UpdatedAt.String(),
	}

	return toState.ToResourceData(), nil
}

func (p *eventProvider) Delete(ctx context.Context, ID string, resourceType string, state resources.ResourceData) error {
	p.log.Debug("deleting event in upstream catalog", "id", ID)

	err := p.client.DeleteEvent(ctx, state["id"].(string))
	if err != nil {
		return fmt.Errorf("deleting event in upstream catalog: %w", err)
	}

	return nil
}
