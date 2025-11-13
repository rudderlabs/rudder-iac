package datacatalog

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	impProvider "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/importremote/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	syncerstate "github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/utils"
	"github.com/samber/lo"
)

type TrackingPlanEntityProvider struct {
	*TrackingPlanProvider
	*impProvider.TrackingPlanImportProvider
}

type TrackingPlanProvider struct {
	client catalog.DataCatalog
	log    *logger.Logger
}

const (
	PropertiesIdentity    = "properties"
	TraitsIdentity        = "traits"
	ContextTraitsIdentity = "context.traits"
)

func NewTrackingPlanProvider(dc catalog.DataCatalog, importDir string) *TrackingPlanEntityProvider {
	tp := &TrackingPlanProvider{
		client: dc,
		log: &logger.Logger{
			Logger: log.With("type", "trackingplan"),
		},
	}

	imp := impProvider.NewTrackingPlanImportProvider(
		dc,
		logger.Logger{
			Logger: logger.New("importremote.provider").With("type", "trackingplan"),
		},
		importDir,
	)

	return &TrackingPlanEntityProvider{
		TrackingPlanProvider:       tp,
		TrackingPlanImportProvider: imp,
	}
}

func (p *TrackingPlanProvider) Create(ctx context.Context, ID string, input resources.ResourceData) (*resources.ResourceData, error) {
	p.log.Debug("creating tracking plan", "id", ID)

	args := state.TrackingPlanArgs{}
	args.FromResourceData(input)

	created, err := p.client.CreateTrackingPlan(ctx, catalog.TrackingPlanCreate{
		Name:        args.Name,
		Description: args.Description,
		ExternalID:  ID,
	})

	if err != nil {
		return nil, fmt.Errorf("creating tracking plan in catalog: %w", err)
	}

	var (
		eventStates []*state.TrackingPlanEventState
	)

	version := created.Version
	for _, event := range args.Events {
		lastupserted, err := p.client.UpdateTrackingPlanEvent(
			ctx,
			created.ID,
			GetUpsertEventIdentifier(event),
		)

		if err != nil {
			return nil, fmt.Errorf("upserting event: %s tracking plan in catalog: %w", event.LocalID, err)
		}

		eventStates = append(eventStates, &state.TrackingPlanEventState{
			LocalID: event.LocalID,
			EventID: event.ID.(string),
		})
		version = lastupserted.Version
	}

	// sort eventStates based on localId
	utils.SortByLocalID(eventStates)

	tpState := state.TrackingPlanState{
		TrackingPlanArgs: args,
		ID:               created.ID,
		Name:             created.Name,
		Version:          version,
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

func (p *TrackingPlanProvider) Update(ctx context.Context, ID string, input resources.ResourceData, olds resources.ResourceData) (*resources.ResourceData, error) {
	p.log.Debug("updating tracking plan", "id", ID)

	prevState := state.TrackingPlanState{}
	prevState.FromResourceData(olds)

	toArgs := state.TrackingPlanArgs{}
	toArgs.FromResourceData(input)

	var (
		updated            *catalog.TrackingPlan
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

	diff := prevState.TrackingPlanArgs.Diff(toArgs)

	var deletedEvents []string
	for _, event := range diff.Deleted {

		upstreamEvent := prevState.EventByID(event.ID.(string))
		if upstreamEvent == nil {
			return nil, fmt.Errorf("state discrepancy as upstream event not found for event id: %s", event.ID.(string))
		}

		if err := p.client.DeleteTrackingPlanEvent(ctx, prevState.ID, upstreamEvent.EventID); err != nil && !catalog.IsCatalogNotFoundError(err) {
			return nil, fmt.Errorf("deleting tracking plan event in catalog: %w", err)
		}

		deletedEvents = append(deletedEvents, upstreamEvent.EventID)
	}

	for _, event := range diff.Added {
		updated, err = p.client.UpdateTrackingPlanEvent(
			ctx,
			prevState.ID,
			GetUpsertEventIdentifier(event),
		)

		if err != nil {
			return nil, fmt.Errorf("upserting event: %s tracking plan in catalog: %w", event.LocalID, err)
		}

		updatedEventStates = append(updatedEventStates, &state.TrackingPlanEventState{
			EventID: event.ID.(string),
			LocalID: event.LocalID,
		})
	}

	for _, event := range diff.Updated {
		updated, err = p.client.UpdateTrackingPlanEvent(
			ctx,
			prevState.ID,
			GetUpsertEventIdentifier(event),
		)

		if err != nil {
			return nil, fmt.Errorf("upserting event: %s tracking plan in catalog: %w", event.LocalID, err)
		}

	}

	// filter the deleted events in it.
	updatedEventStates = lo.Filter(updatedEventStates, func(event *state.TrackingPlanEventState, idx int) bool {
		return !lo.Contains(deletedEvents, event.EventID)
	})

	// sort updatedEventStates based on localId
	utils.SortByLocalID(updatedEventStates)

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

func (p *TrackingPlanProvider) Delete(ctx context.Context, ID string, state resources.ResourceData) error {
	p.log.Debug("deleting tracking plan", "id", ID)

	if err := p.client.DeleteTrackingPlan(ctx, state["id"].(string)); err != nil && !catalog.IsCatalogNotFoundError(err) {
		return fmt.Errorf("deleting tracking plan in catalog: %w", err)
	}

	return nil
}

func (p *TrackingPlanProvider) Import(ctx context.Context, ID string, data resources.ResourceData, remoteId string) (*resources.ResourceData, error) {
	p.log.Debug("importing tracking plan resource", "id", ID, "remoteId", remoteId)

	trackingPlan, err := p.client.GetTrackingPlan(ctx, remoteId)
	if err != nil {
		return nil, fmt.Errorf("getting tracking plan from upstream: %w", err)
	}

	toArgs := state.TrackingPlanArgs{}
	toArgs.FromResourceData(data)

	changed, diffed := toArgs.DiffUpstream(trackingPlan)
	if changed {
		p.log.Debug("tracking plan has differences, updating", "id", ID, "remoteId", remoteId)

		_, err = p.client.UpdateTrackingPlan(ctx, remoteId, toArgs.Name, toArgs.Description)
		if err != nil {
			return nil, fmt.Errorf("updating tracking plan during import: %w", err)
		}

		for _, deleted := range diffed.Deleted {
			err = p.client.DeleteTrackingPlanEvent(ctx, remoteId, deleted.ID.(string))
			if err != nil {
				return nil, fmt.Errorf("deleting tracking plan event during import: %w", err)
			}
		}

		for _, added := range diffed.Added {
			_, err = p.client.UpdateTrackingPlanEvent(ctx, remoteId, GetUpsertEventIdentifier(added))
			if err != nil {
				return nil, fmt.Errorf("updating tracking plan event during import: %w", err)
			}
		}

		for _, updated := range diffed.Updated {
			_, err = p.client.UpdateTrackingPlanEvent(ctx, remoteId, GetUpsertEventIdentifier(updated))
			if err != nil {
				return nil, fmt.Errorf("updating tracking plan event during import: %w", err)
			}
		}
	}

	err = p.client.SetTrackingPlanExternalId(ctx, remoteId, ID)
	if err != nil {
		return nil, fmt.Errorf("setting tracking plan external id: %w", err)
	}

	trackingPlanState := state.TrackingPlanState{
		TrackingPlanArgs: toArgs,
		ID:               trackingPlan.ID,
		Name:             toArgs.Name,
		Description:      toArgs.Description,
		Version:          trackingPlan.Version,
		CreationType:     trackingPlan.CreationType,
		WorkspaceID:      trackingPlan.WorkspaceID,
		CreatedAt:        trackingPlan.CreatedAt.String(),
		UpdatedAt:        trackingPlan.UpdatedAt.String(),
	}

	resourceData := trackingPlanState.ToResourceData()
	return &resourceData, nil
}

// LoadResourcesFromRemote loads all tracking plans from the remote catalog
func (p *TrackingPlanProvider) LoadResourcesFromRemote(ctx context.Context) (*resources.ResourceCollection, error) {
	p.log.Debug("loading tracking plans from remote catalog ")

	collection := resources.NewResourceCollection()
	trackingPlans, err := p.client.GetTrackingPlans(ctx, catalog.ListOptions{HasExternalID: lo.ToPtr(true)})
	if err != nil {
		return nil, err
	}

	resourceMap := make(map[string]*resources.RemoteResource)
	for _, trackingPlan := range trackingPlans {
		resourceMap[trackingPlan.ID] = &resources.RemoteResource{
			ID:         trackingPlan.ID,
			ExternalID: trackingPlan.ExternalID,
			Data:       trackingPlan,
		}
	}
	collection.Set(state.TrackingPlanResourceType, resourceMap)
	return collection, nil
}

func (p *TrackingPlanProvider) LoadStateFromResources(ctx context.Context, collection *resources.ResourceCollection) (*syncerstate.State, error) {
	s := syncerstate.EmptyState()
	trackingPlans := collection.GetAll(state.TrackingPlanResourceType)
	for _, remoteTP := range trackingPlans {
		if remoteTP.ExternalID == "" {
			continue
		}
		trackingPlan, ok := remoteTP.Data.(*catalog.TrackingPlanWithIdentifiers)
		if !ok {
			return nil, fmt.Errorf("LoadStateFromResources: unable to cast remote resource to catalog.TrackingPlan")
		}
		args := &state.TrackingPlanArgs{}
		args.FromRemoteTrackingPlan(trackingPlan, collection)

		stateArgs := state.TrackingPlanState{}
		stateArgs.FromRemoteTrackingPlan(trackingPlan, collection)

		resourceState := &syncerstate.ResourceState{
			Type:         state.TrackingPlanResourceType,
			ID:           remoteTP.ExternalID,
			Input:        args.ToResourceData(),
			Output:       stateArgs.ToResourceData(),
			Dependencies: make([]string, 0),
		}

		urn := resources.URN(remoteTP.ExternalID, state.TrackingPlanResourceType)
		s.Resources[urn] = resourceState
	}
	return s, nil
}

func GetUpsertEventIdentifier(from *state.TrackingPlanEventArgs) catalog.EventIdentifierDetail {
	return catalog.EventIdentifierDetail{
		ID: from.ID.(string),
		Properties: lo.Map(
			from.Properties,
			func(prop *state.TrackingPlanPropertyArgs, _ int) catalog.PropertyIdentifierDetail {
				return GetUpsertPropertyIdentifier(prop)
			}),
		AdditionalProperties: from.AllowUnplanned,
		IdentitySection:      from.IdentitySection,
		Variants:             from.Variants.ToCatalogVariants(),
	}
}

func GetUpsertPropertyIdentifier(from *state.TrackingPlanPropertyArgs) catalog.PropertyIdentifierDetail {
	res := catalog.PropertyIdentifierDetail{
		ID:                   from.ID.(string),
		Required:             from.Required,
		AdditionalProperties: from.AdditionalProperties,
	}

	if len(from.Properties) > 0 {
		res.Properties = lo.Map(from.Properties, func(prop *state.TrackingPlanPropertyArgs, _ int) catalog.PropertyIdentifierDetail {
			return GetUpsertPropertyIdentifier(prop)
		})
	}

	return res
}
