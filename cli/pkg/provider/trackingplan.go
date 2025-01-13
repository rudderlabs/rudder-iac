package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
	"github.com/rudderlabs/rudder-iac/cli/pkg/provider/state"
	"github.com/samber/lo"
)

type TrackingPlanProvider struct {
	client client.DataCatalog
	log    *logger.Logger
}

const (
	PropertiesIdentity    = "properties"
	TraitsIdentity        = "traits"
	ContextTraitsIdentity = "context.traits"
)

func NewTrackingPlanProvider(client client.DataCatalog) syncer.Provider {
	return &TrackingPlanProvider{
		client: client,
		log: &logger.Logger{
			Logger: log.With("type", "trackingplan"),
		},
	}
}

func (p *TrackingPlanProvider) Create(ctx context.Context, ID string, resourceType string, input resources.ResourceData) (*resources.ResourceData, error) {
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

	for _, event := range args.Events {
		lastupserted, err := p.client.UpsertTrackingPlan(
			ctx,
			created.ID,
			getUpsertEvent(event),
		)
		if err != nil {
			return nil, fmt.Errorf("upserting event: %s tracking plan in catalog: %w", event.LocalID, err)
		}

		lastEvent := lastupserted.Events[len(lastupserted.Events)-1]
		eventStates = append(eventStates, &state.TrackingPlanEventState{
			ID:      lastEvent.ID,
			EventID: lastEvent.EventID,
			LocalID: event.LocalID,
		})
	}

	tpState := state.TrackingPlanState{
		TrackingPlanArgs: args,
		ID:               created.ID,
		Name:             created.Name,
		Version:          created.Version,
		CreationType:     created.CreationType,
		Description:      *created.Description,
		WorkspaceID:      created.WorkspaceID,
		CreatedAt:        created.CreatedAt.String(),
		UpdatedAt:        created.UpdatedAt.String(),
		Events:           eventStates,
	}

	resourceData := tpState.ToResourceData()
	return &resourceData, nil

}

func (p *TrackingPlanProvider) Update(ctx context.Context, ID string, resourceType string, input resources.ResourceData, olds resources.ResourceData) (*resources.ResourceData, error) {
	p.log.Debug("updating tracking plan", "id", ID)

	prevState := state.TrackingPlanState{}
	prevState.FromResourceData(olds)

	toArgs := state.TrackingPlanArgs{}
	toArgs.FromResourceData(input)

	var (
		updated            *client.TrackingPlan
		err                error
		updatedEventStates = make([]*state.TrackingPlanEventState, 0)
	)

	// Start with the previous event states
	updatedEventStates = append(updatedEventStates, prevState.Events...)
	if prevState.TrackingPlanArgs.Name != toArgs.Name || prevState.TrackingPlanArgs.Description != toArgs.Description {
		if updated, err = p.client.UpdateTrackingPlan(
			ctx,
			prevState.ID,
			toArgs.Name,
			toArgs.Description); err != nil {
			return nil, fmt.Errorf("updating tracking plan in catalog: %w", err)
		}
	}

	diff := prevState.Diff(toArgs)

	// diff := p.Diff(&toArgs, &prevState.TrackingPlanArgs)

	var deletedEvents []string
	for _, event := range diff.Deleted {

		upstreamEvent := prevState.EventByLocalID(event.LocalID)
		if upstreamEvent == nil {
			return nil, fmt.Errorf("state discrepancy as upstream event not found for local id: %s", event.LocalID)
		}

		if err := p.client.DeleteTrackingPlanEvent(ctx, prevState.ID, upstreamEvent.EventID); err != nil && !client.IsCatalogNotFoundError(err) {
			return nil, fmt.Errorf("deleting tracking plan event in catalog: %w", err)
		}

		// capture the catalogeventID which are unique as
		// the newly created events can have same localID
		deletedEvents = append(deletedEvents, upstreamEvent.ID)
	}

	for _, event := range diff.Added {
		updated, err = p.client.UpsertTrackingPlan(
			ctx,
			prevState.ID,
			getUpsertEvent(event),
		)

		if err != nil {
			return nil, fmt.Errorf("upserting event: %s tracking plan in catalog: %w", event.LocalID, err)
		}

		updatedEventStates = append(updatedEventStates, &state.TrackingPlanEventState{
			ID:      updated.Events[len(updated.Events)-1].ID,
			EventID: updated.Events[len(updated.Events)-1].EventID,
			LocalID: event.LocalID,
		})
	}

	for _, event := range diff.Updated {
		updated, err = p.client.UpsertTrackingPlan(
			ctx,
			prevState.ID,
			getUpsertEvent(event),
		)

		if err != nil {
			return nil, fmt.Errorf("upserting event: %s tracking plan in catalog: %w", event.LocalID, err)
		}

	}

	// filter the deleted events in it.
	updatedEventStates = lo.Filter(updatedEventStates, func(event *state.TrackingPlanEventState, idx int) bool {
		return !lo.Contains(deletedEvents, event.ID)
	})

	var tpState state.TrackingPlanState

	if updated == nil {
		// Copy from previous if anything isn't getting updated so we don't panic
		tpState = state.TrackingPlanState{
			TrackingPlanArgs: toArgs,
			ID:               prevState.ID,
			Name:             prevState.Name,
			Description:      prevState.Description,
			CreationType:     prevState.CreationType,
			Version:          prevState.Version,
			WorkspaceID:      prevState.WorkspaceID,
			CreatedAt:        prevState.CreatedAt,
			UpdatedAt:        prevState.UpdatedAt,
			Events:           prevState.Events,
		}
	} else {
		tpState = state.TrackingPlanState{
			TrackingPlanArgs: toArgs,
			ID:               updated.ID,
			Name:             updated.Name,
			Description:      *updated.Description,
			CreationType:     updated.CreationType,
			Version:          updated.Version,
			WorkspaceID:      updated.WorkspaceID,
			CreatedAt:        updated.CreatedAt.String(),
			UpdatedAt:        updated.UpdatedAt.String(),
			Events:           updatedEventStates,
		}
	}

	resourceData := tpState.ToResourceData()
	return &resourceData, nil
}

func (p *TrackingPlanProvider) Delete(ctx context.Context, ID string, resourceType string, state resources.ResourceData) error {
	p.log.Debug("deleting tracking plan", "id", ID)

	if err := p.client.DeleteTrackingPlan(ctx, state["id"].(string)); err != nil && !client.IsCatalogNotFoundError(err) {
		return fmt.Errorf("deleting tracking plan in catalog: %w", err)
	}

	return nil
}

func getUpsertEvent(from *state.TrackingPlanEventArgs) client.TrackingPlanUpsertEvent {
	// Get the properties in correct shape before we can
	// send it to the catalog
	var (
		requiredProps   = make([]string, 0)
		propLookup      = make(map[string]interface{})
		identitySection = from.IdentitySection
	)

	// If the identity section empty, default to properties
	if from.IdentitySection == "" {
		identitySection = PropertiesIdentity
	}

	// Only for simple types
	for _, prop := range from.Properties {

		typ := lo.Map(strings.Split(prop.Type, ","), func(t string, _ int) string {
			return strings.TrimSpace(t)
		})

		propLookup[prop.Name] = map[string]interface{}{
			"type": typ,
		}

		for k, v := range prop.Config {

			if k == "itemTypes" {
				propLookup[prop.Name].(map[string]interface{})["items"] = map[string]interface{}{
					"type": v,
				}
			} else {
				propLookup[prop.Name].(map[string]interface{})[k] = v
			}

		}

		// keep on updating the required properties
		if prop.Required {
			requiredProps = append(requiredProps, prop.Name)
		}
	}

	return client.TrackingPlanUpsertEvent{
		Name:            from.Name,
		Description:     from.Description,
		EventType:       from.Type,
		IdentitySection: identitySection,
		Rules: getRulesBasedonIdentity(identitySection, &client.TrackingPlanUpsertEventProperties{
			Type:                 "object",
			AdditionalProperties: from.AllowUnplanned,
			Required:             requiredProps,
			Properties:           propLookup,
		}),
	}
}

func getRulesBasedonIdentity(identity string, properties *client.TrackingPlanUpsertEventProperties) client.TrackingPlanUpsertEventRules {
	var (
		propertiesIdentity *client.TrackingPlanUpsertEventProperties
		traitsIdentity     *client.TrackingPlanUpsertEventProperties
		contextIdentity    *client.TrackingPlanUpsertEventContextTraitsIdentity
	)

	switch identity {

	case PropertiesIdentity:
		propertiesIdentity = properties

	case TraitsIdentity:
		traitsIdentity = properties

	case ContextTraitsIdentity:
		contextIdentity = &client.TrackingPlanUpsertEventContextTraitsIdentity{
			Properties: struct {
				Traits client.TrackingPlanUpsertEventProperties `json:"traits,omitempty"`
			}{
				Traits: *properties,
			},
		}

	default:
		propertiesIdentity = properties // fallback to properties
	}

	return client.TrackingPlanUpsertEventRules{
		Type: "object",
		Properties: struct {
			Properties *client.TrackingPlanUpsertEventProperties            `json:"properties,omitempty"`
			Traits     *client.TrackingPlanUpsertEventProperties            `json:"traits,omitempty"`
			Context    *client.TrackingPlanUpsertEventContextTraitsIdentity `json:"context,omitempty"`
		}{
			Properties: propertiesIdentity,
			Traits:     traitsIdentity,
			Context:    contextIdentity,
		},
	}
}
