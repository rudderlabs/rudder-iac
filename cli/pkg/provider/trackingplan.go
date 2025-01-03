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

type trackingPlanProvider struct {
	client client.DataCatalog
	log    *logger.Logger
}

func newTrackingPlanProvider(client client.DataCatalog) syncer.Provider {
	return &trackingPlanProvider{
		client: client,
		log: &logger.Logger{
			Logger: log.With("type", "trackingplan"),
		},
	}
}

func (p *trackingPlanProvider) Create(ctx context.Context, ID string, resourceType string, input resources.ResourceData) (*resources.ResourceData, error) {
	p.log.Debug("creating tracking plan", "id", ID)

	args := state.TrackingPlanArgs{}
	args.FromResourceData(input)

	created, err := p.client.CreateTrackingPlan(ctx, client.TrackingPlanCreate{
		Name:        args.Name,
		Description: args.Description,
	})

	if err != nil {
		return nil, fmt.Errorf("creating tracking plan in catalog: %w", err)
	}

	var (
		eventStates []*state.TrackingPlanEventState
	)

	// FIXME: This is a hacky way to construct the state from each event upsert
	// which can fail in the middle itself and leave the system in a inconsistent state
	for _, event := range args.Events {
		lastupserted, err := p.client.UpsertTrackingPlan(
			ctx,
			created.ID,
			state.GetUpsertEventPayload(event),
		)
		if err != nil {
			return nil, fmt.Errorf("upserting event: %s tracking plan in catalog: %w", event.LocalID, err)
		}

		eventStates = append(eventStates, state.ConstructTPEventState(
			event,
			&lastupserted.Events[len(lastupserted.Events)-1]))
	}

	tpState := state.TrackingPlanState{
		TrackingPlanArgs: args,
		ID:               created.ID,
		Name:             created.Name,
		Description:      *created.Description,
		WorkspaceID:      created.WorkspaceID,
		CreatedAt:        created.CreatedAt.String(),
		UpdatedAt:        created.UpdatedAt.String(),
		Events:           eventStates,
	}

	resourceData := tpState.ToResourceData()
	return &resourceData, nil

}

func (p *trackingPlanProvider) Update(ctx context.Context, ID string, resourceType string, input resources.ResourceData, olds resources.ResourceData) (*resources.ResourceData, error) {
	p.log.Debug("updating tracking plan", "id", ID)

	prevState := state.TrackingPlanState{}
	prevState.FromResourceData(olds)

	toArgs := state.TrackingPlanArgs{}
	toArgs.FromResourceData(input)

	var (
		updated *client.TrackingPlan
		err     error
	)

	if prevState.TrackingPlanArgs.Name != toArgs.Name || prevState.TrackingPlanArgs.Description != toArgs.Description {
		if updated, err = p.client.UpdateTrackingPlan(
			ctx,
			prevState.ID,
			toArgs.Name,
			toArgs.Description); err != nil {
			return nil, fmt.Errorf("updating tracking plan in catalog: %w", err)
		}
	}

	diff := p.diff(&toArgs, &prevState.TrackingPlanArgs)
	updatedEventStates := make([]*state.TrackingPlanEventState, 0)

	for _, event := range diff.Deleted {

		catalogEventID := prevState.CatalogEventIDForLocalID(event.LocalID)
		if catalogEventID == "" {
			return nil, fmt.Errorf("state discrepancy as upstream event id not found for local id: %s", event.LocalID)
		}

		if err := p.client.DeleteTrackingPlanEvent(ctx, prevState.ID, catalogEventID); err != nil {
			return nil, fmt.Errorf("deleting tracking plan event in catalog: %w", err)
		}
	}

	for _, event := range diff.Added {
		updated, err = p.client.UpsertTrackingPlan(
			ctx,
			prevState.ID,
			state.GetUpsertEventPayload(event),
		)

		if err != nil {
			return nil, fmt.Errorf("upserting event: %s tracking plan in catalog: %w", event.LocalID, err)
		}

		updatedEventStates = append(updatedEventStates, state.ConstructTPEventState(
			event,
			&updated.Events[len(updated.Events)-1]))
	}

	for _, event := range diff.Updated {
		lastupserted, err := p.client.UpsertTrackingPlan(
			ctx,
			prevState.ID,
			state.GetUpsertEventPayload(event),
		)

		if err != nil {
			return nil, fmt.Errorf("upserting event: %s tracking plan in catalog: %w", event.LocalID, err)
		}

		updatedEventStates = append(updatedEventStates, state.ConstructTPEventState(
			event,
			&lastupserted.Events[len(lastupserted.Events)-1]))
	}

	tpState := state.TrackingPlanState{
		TrackingPlanArgs: toArgs,
		ID:               updated.ID,
		Name:             updated.Name,
		Description:      *updated.Description,
		WorkspaceID:      updated.WorkspaceID,
		CreatedAt:        updated.CreatedAt.String(),
		UpdatedAt:        updated.UpdatedAt.String(),
		Events:           updatedEventStates,
	}

	resourceData := tpState.ToResourceData()
	return &resourceData, nil
}

func (p *trackingPlanProvider) Delete(ctx context.Context, ID string, resourceType string, state resources.ResourceData) error {
	p.log.Debug("deleting tracking plan", "id", ID)

	if err := p.client.DeleteTrackingPlan(ctx, state["id"].(string)); err != nil {
		return fmt.Errorf("deleting tracking plan in catalog: %w", err)
	}

	return nil
}

func (p *trackingPlanProvider) diff(current *state.TrackingPlanArgs, old *state.TrackingPlanArgs) state.TrackingPlanStateDiff {
	diffResponse := state.TrackingPlanStateDiff{
		Added:   make([]*state.TrackingPlanEventArgs, 0),
		Updated: make([]*state.TrackingPlanEventArgs, 0),
		Deleted: make([]*state.TrackingPlanEventArgs, 0),
	}

	for _, event := range current.Events {
		// In new but not in old ->  Added
		if oldEvent := old.EventByLocalID(event.LocalID); oldEvent != nil {
			continue
		}
		diffResponse.Added = append(diffResponse.Added, event)
	}

	// Updated / Deleted calculation
	for _, event := range old.Events {
		var inputEvent *state.TrackingPlanEventArgs

		// In old but not in new ->  Deleted
		if inputEvent = current.EventByLocalID(event.LocalID); inputEvent == nil {
			diffResponse.Deleted = append(diffResponse.Deleted, event)
			continue
		}

		// Change in values local to event
		if event.AllowUnplanned != inputEvent.AllowUnplanned {
			diffResponse.Updated = append(diffResponse.Updated, inputEvent)
			continue
		}

		// Change in values local to properties
		if len(event.Properties) != len(inputEvent.Properties) {
			diffResponse.Updated = append(diffResponse.Updated, inputEvent)
			continue
		}

		for _, prop := range event.Properties {

			var inputProp *state.TrackingPlanPropertyArgs
			if inputProp = event.PropertyByLocalID(prop.LocalID); inputProp == nil {
				diffResponse.Updated = append(diffResponse.Updated, inputEvent)
				break
			}

			if prop.Required != inputProp.Required {
				diffResponse.Updated = append(diffResponse.Updated, inputEvent)
				break
			}
		}
	}

	return diffResponse
}
