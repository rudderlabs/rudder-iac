package source

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	esClient "github.com/rudderlabs/rudder-iac/api/client/event-stream"
	sourceClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
	trackingplanClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/tracking-plan-connection"
	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	dcstate "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

type Handler struct {
	*provider.BaseHandler[sourceSpec, SourceResource, SourceStateRemote]
	client    esClient.EventStreamStore
	importDir string
}

func NewHandler(client esClient.EventStreamStore, importDir string) *Handler {
	return &Handler{
		client:      client,
		importDir:   filepath.Join(importDir, ImportPath),
		BaseHandler: provider.NewHandler(ResourceType, &handlerImpl{client: client}),
	}
}

// CreateIDRef creates a PropertyRef that resolves to the remote ID of an event stream source
func (h *Handler) CreateIDRef(urn string) *resources.PropertyRef {
	return h.BaseHandler.CreatePropertyRef(urn, func(state *SourceStateRemote) (string, error) {
		return state.ID, nil
	})
}

type handlerImpl struct {
	client esClient.EventStreamStore
}

func (h *handlerImpl) NewSpec() *sourceSpec {
	return &sourceSpec{}
}

func (h *handlerImpl) ValidateSpec(spec *sourceSpec) error {
	return nil
}

func (h *handlerImpl) ExtractResourcesFromSpec(path string, spec *sourceSpec) (map[string]*SourceResource, error) {
	enabled := true
	if spec.Enabled != nil {
		enabled = *spec.Enabled
	}
	sr := &SourceResource{
		ID:      spec.LocalId,
		Name:    spec.Name,
		Type:    spec.Type,
		Enabled: enabled,
	}
	if err := h.loadTrackingPlanSpec(spec, sr); err != nil {
		return nil, err
	}

	return map[string]*SourceResource{
		sr.ID: sr,
	}, nil
}

func (h *handlerImpl) loadTrackingPlanSpec(spec *sourceSpec, sourceResource *SourceResource) error {
	if spec.Governance == nil || spec.Governance.TrackingPlan == nil {
		return nil
	}

	tp := spec.Governance.TrackingPlan
	if tp.Ref == "" {
		return fmt.Errorf("governance.validations.tracking_plan is required")
	}
	if tp.Config == nil {
		return fmt.Errorf("governance.validations.config is required")
	}

	trackingPlanRef, err := parseTrackingPlanRef(tp.Ref)
	if err != nil {
		return fmt.Errorf("parsing tracking plan reference: %w", err)
	}

	sourceResource.Governance = &GovernanceResource{
		Validations: &ValidationsResource{
			TrackingPlanRef: trackingPlanRef,
			Config:          &TrackingPlanConfigResource{},
		},
	}

	config := sourceResource.Governance.Validations.Config
	tpConfigSpec := tp.Config

	if tpConfigSpec.Track != nil {
		config.Track = &TrackConfigResource{
			EventConfigResource: &EventConfigResource{
				PropagateViolations:     tpConfigSpec.Track.PropagateViolations,
				DropUnplannedProperties: tpConfigSpec.Track.DropUnplannedProperties,
				DropOtherViolations:     tpConfigSpec.Track.DropOtherViolations,
			},
			DropUnplannedEvents: tpConfigSpec.Track.DropUnplannedEvents,
		}
	}
	if tpConfigSpec.Identify != nil {
		config.Identify = buildEventConfigFromSpec(tpConfigSpec.Identify)
	}
	if tpConfigSpec.Group != nil {
		config.Group = buildEventConfigFromSpec(tpConfigSpec.Group)
	}
	if tpConfigSpec.Page != nil {
		config.Page = buildEventConfigFromSpec(tpConfigSpec.Page)
	}
	if tpConfigSpec.Screen != nil {
		config.Screen = buildEventConfigFromSpec(tpConfigSpec.Screen)
	}

	return nil
}

func buildEventConfigFromSpec(specConfig *eventConfigSpec) *EventConfigResource {
	return &EventConfigResource{
		PropagateViolations:     specConfig.PropagateViolations,
		DropUnplannedProperties: specConfig.DropUnplannedProperties,
		DropOtherViolations:     specConfig.DropOtherViolations,
	}
}

func (h *handlerImpl) ValidateResource(source *SourceResource, graph *resources.Graph) error {
	if source.ID == "" {
		return fmt.Errorf("id is required")
	}
	if source.Name == "" {
		return fmt.Errorf("name is required")
	}
	if source.Type == "" {
		return fmt.Errorf("type is required")
	}
	if !slices.Contains(sourceDefinitions, source.Type) {
		return fmt.Errorf("type '%s' is invalid, must be one of: %v", source.Type, sourceDefinitions)
	}
	if source.Governance != nil && source.Governance.Validations != nil {
		ref := source.Governance.Validations.TrackingPlanRef
		if ref == nil {
			return fmt.Errorf("governance.validations.tracking_plan is required")
		}

		tp, ok := graph.GetResource(ref.URN)
		if !ok {
			return fmt.Errorf("tracking plan with URN '%s' not found in the project", ref.URN)
		}

		if tp.Type() != dcstate.TrackingPlanResourceType {
			return fmt.Errorf("referenced URN '%s' is not a tracking plan", ref.URN)
		}
	}
	return nil
}

func (h *handlerImpl) Create(ctx context.Context, data *SourceResource) (*SourceStateRemote, error) {
	createRequest := &sourceClient.CreateSourceRequest{
		ExternalID: data.ID,
		Name:       data.Name,
		Type:       data.Type,
		Enabled:    data.Enabled,
	}
	resp, err := h.client.Create(ctx, createRequest)
	if err != nil {
		return nil, fmt.Errorf("creating event stream source: %w", err)
	}

	trackingPlanID := data.GetTrackingPlanID()
	if trackingPlanID != "" {
		err := h.linkTrackingPlan(ctx, trackingPlanID, resp.ID, data)
		if err != nil {
			return nil, fmt.Errorf("linking tracking plan to event stream source: %w", err)
		}
	}
	return &SourceStateRemote{ID: resp.ID, TrackingPlanID: trackingPlanID}, nil
}

func (h *handlerImpl) Update(ctx context.Context, newData *SourceResource, oldData *SourceResource, oldState *SourceStateRemote) (*SourceStateRemote, error) {
	if oldData.Type != newData.Type {
		return nil, fmt.Errorf("type cannot be changed")
	}

	err := h.updateSource(ctx, newData, oldData, oldState)
	if err != nil {
		return nil, fmt.Errorf("updating event stream source: %w", err)
	}
	if err := h.updateTrackingPlanConnection(ctx, newData, oldData, oldState); err != nil {
		return nil, err
	}
	return &SourceStateRemote{ID: oldState.ID, TrackingPlanID: newData.GetTrackingPlanID()}, nil
}

func (h *handlerImpl) updateSource(ctx context.Context, newData *SourceResource, oldData *SourceResource, oldState *SourceStateRemote) error {
	needsUpdate := oldData.Name != newData.Name || oldData.Enabled != newData.Enabled
	if !needsUpdate {
		return nil
	}
	updateRequest := &sourceClient.UpdateSourceRequest{
		Name:    newData.Name,
		Enabled: newData.Enabled,
	}
	_, err := h.client.Update(ctx, oldState.ID, updateRequest)
	if err != nil {
		return fmt.Errorf("updating event stream source: %w", err)
	}
	return nil
}

func (h *handlerImpl) updateTrackingPlanConnection(ctx context.Context, newData *SourceResource, oldData *SourceResource, oldState *SourceStateRemote) error {
	newTPID := newData.GetTrackingPlanID()

	if newTPID == "" && oldState.TrackingPlanID == "" {
		return nil
	}

	if newTPID != "" {
		if oldState.TrackingPlanID != "" {
			return h.updateExistingTrackingPlanConnection(ctx, newData, oldData, oldState)

		} else {
			return h.linkTrackingPlan(ctx, newTPID, oldState.ID, newData)

		}
	} else if oldState.TrackingPlanID != "" {
		return h.unlinkTrackingPlan(ctx, oldState.TrackingPlanID, oldState.ID)
	}

	return nil
}

func (h *handlerImpl) linkTrackingPlan(ctx context.Context, trackingPlanID, remoteID string, data *SourceResource) error {
	remoteConfig, err := mapStateTPConfigToRemote(data.GetTrackingPlanConfig())
	if err != nil {
		return fmt.Errorf("invalid tracking plan config: %w", err)
	}

	err = h.client.LinkTP(ctx, trackingPlanID, remoteID, remoteConfig)
	if err != nil {
		return fmt.Errorf("linking tracking plan to event stream source: %w", err)
	}
	return nil
}

func (h *handlerImpl) unlinkTrackingPlan(ctx context.Context, trackingPlanID, remoteID string) error {
	err := h.client.UnlinkTP(ctx, trackingPlanID, remoteID)
	if err != nil {
		return fmt.Errorf("unlinking tracking plan from event stream source: %w", err)
	}
	return nil
}

func (h *handlerImpl) updateExistingTrackingPlanConnection(ctx context.Context, newData *SourceResource, oldData *SourceResource, oldState *SourceStateRemote) error {
	currentTrackingPlanID := oldState.TrackingPlanID
	newTrackingPlanID := newData.GetTrackingPlanID()
	remoteID := oldState.ID

	if currentTrackingPlanID == newTrackingPlanID {
		return h.updateTrackingPlanConfig(ctx, newData, oldData, oldState)
	}

	if err := h.unlinkTrackingPlan(ctx, currentTrackingPlanID, remoteID); err != nil {
		return fmt.Errorf("unlinking old tracking plan: %w", err)
	}

	if err := h.linkTrackingPlan(ctx, newTrackingPlanID, remoteID, newData); err != nil {
		return fmt.Errorf("linking new tracking plan: %w", err)
	}

	return nil
}

func (h *handlerImpl) updateTrackingPlanConfig(ctx context.Context, newData *SourceResource, oldData *SourceResource, oldState *SourceStateRemote) error {
	trackingPlanID := newData.GetTrackingPlanID()
	remoteID := oldState.ID
	newConfig := newData.GetTrackingPlanConfig()
	oldConfig := oldData.GetTrackingPlanConfig()

	if !reflect.DeepEqual(newConfig, oldConfig) {
		remoteConfig, err := mapStateTPConfigToRemote(newConfig)
		if err != nil {
			return fmt.Errorf("invalid tracking plan config: %w", err)
		}

		err = h.client.UpdateTPConnection(ctx, trackingPlanID, remoteID, remoteConfig)
		if err != nil {
			return fmt.Errorf("updating tracking plan connection: %w", err)
		}
	}

	return nil
}

func (h *Handler) LoadResourcesFromRemote(ctx context.Context) (*resources.ResourceCollection, error) {
	collection := resources.NewResourceCollection()
	sources, err := h.client.GetSources(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting event stream sources: %w", err)
	}
	resourceMap := make(map[string]*resources.RemoteResource)
	for _, source := range sources {
		if source.ExternalID == "" {
			// loop over the sources which have externalID not set
			// as they are not anyway part of the state
			continue
		}
		resourceMap[source.ID] = &resources.RemoteResource{
			ID:         source.ID,
			ExternalID: source.ExternalID,
			Data:       source,
		}
	}
	collection.Set(ResourceType, resourceMap)
	return collection, nil
}

// TODO: Consider simplifying the handler interface once common patterns are identified across providers:
//  1. ResourceType could be automatically added by BaseHandler instead of being repeated in each handler
//  2. The response types could be made type-safe to ensure InputRaw and OutputRaw have the correct types
//     (e.g., handler could return typed state directly instead of interface{})
func (p *Handler) LoadStateFromResources(ctx context.Context, collection *resources.ResourceCollection) (*state.State, error) {
	s := state.EmptyState()
	esResources := collection.GetAll(ResourceType)
	for _, esResource := range esResources {
		source, ok := esResource.Data.(sourceClient.EventStreamSource)
		if !ok {
			return nil, fmt.Errorf("unable to cast resource to event stream source")
		}
		var trackingPlanURN *string
		if source.TrackingPlan != nil {
			tpURN, err := collection.GetURNByID(dcstate.TrackingPlanResourceType, source.TrackingPlan.ID)
			if err == resources.ErrRemoteResourceExternalIdNotFound {
				// ErrRemoteResourceExternalIdNotFound would happen if the source was created via CLI
				// but the tracking plan was created and linked via the UI/API
				trackingPlanURN = nil
			} else if err != nil {
				return nil, fmt.Errorf("get urn by id: %w", err)
			} else {
				trackingPlanURN = &tpURN
			}
		}
		inputRaw, outputRaw, skip := mapRemoteToState(&source, trackingPlanURN)
		if skip {
			continue
		}

		s.AddResource(&state.ResourceState{
			ID:        source.ExternalID,
			Type:      ResourceType,
			InputRaw:  inputRaw,
			OutputRaw: outputRaw,
		})
	}
	return s, nil
}

func (h *handlerImpl) Delete(ctx context.Context, id string, oldData *SourceResource, oldState *SourceStateRemote) error {
	remoteID := oldState.ID
	trackingPlanID := oldState.TrackingPlanID
	if trackingPlanID != "" {
		err := h.client.UnlinkTP(ctx, trackingPlanID, remoteID)
		if err != nil {
			return fmt.Errorf("unlinking tracking plan from event stream source: %w", err)
		}
	}

	err := h.client.Delete(ctx, remoteID)
	if err != nil {
		return fmt.Errorf("deleting event stream source: %w", err)
	}
	return nil
}

func (h *handlerImpl) Import(ctx context.Context, data *SourceResource, remoteId string) (*SourceStateRemote, error) {
	// FIXME: Instead of fetching all sources, fetch the source with the matching remoteId
	sources, err := h.client.GetSources(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting event stream sources: %w", err)
	}

	// Find the source with matching remoteId
	var existingSource *sourceClient.EventStreamSource
	for _, source := range sources {
		if source.ID == remoteId {
			existingSource = &source
			break
		}
	}

	if existingSource == nil {
		return nil, fmt.Errorf("event stream source with ID %s not found", remoteId)
	}

	// Build old input state from existing source to compare with desired data
	oldData := &SourceResource{
		ID:      data.ID,
		Name:    existingSource.Name,
		Enabled: existingSource.Enabled,
		Type:    existingSource.Type,
	}

	// Build old output state (remote metadata)
	oldState := &SourceStateRemote{
		ID: remoteId,
	}

	// If there's a tracking plan on the existing source, include it in states
	if existingSource.TrackingPlan != nil {
		oldState.TrackingPlanID = existingSource.TrackingPlan.ID
		oldData.Governance = &GovernanceResource{
			Validations: &ValidationsResource{
				Config: mapRemoteTPConfigToState(existingSource.TrackingPlan.Config),
			},
		}
	}

	// Update the source if there are differences
	result, err := h.Update(ctx, data, oldData, oldState)
	if err != nil {
		return nil, fmt.Errorf("updating event stream source during import: %w", err)
	}

	err = h.client.SetExternalID(ctx, remoteId, data.ID)
	if err != nil {
		return nil, fmt.Errorf("setting external ID for event stream source during import: %w", err)
	}
	return result, nil
}

func (h *Handler) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.ResourceCollection, error) {
	collection := resources.NewResourceCollection()
	sources, err := h.client.GetSources(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting event stream sources: %w", err)
	}
	resourceMap := make(map[string]*resources.RemoteResource)
	for _, source := range sources {
		if source.ExternalID != "" {
			continue
		}
		externalID, err := idNamer.Name(namer.ScopeName{
			Name:  source.Name,
			Scope: ResourceType,
		})
		if err != nil {
			return nil, fmt.Errorf("generating externalID for source %s: %w", source.Name, err)
		}
		remoteResource := &resources.RemoteResource{
			ID:         source.ID,
			ExternalID: externalID,
			Reference:  fmt.Sprintf("#/%s/%s/%s", ResourceKind, MetadataName, externalID),
			Data:       &source,
		}
		resourceMap[source.ID] = remoteResource
	}
	collection.Set(ResourceType, resourceMap)
	return collection, nil
}

func (h *Handler) FormatForExport(
	ctx context.Context,
	collection *resources.ResourceCollection,
	idNamer namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]importremote.FormattableEntity, error) {
	sources := collection.GetAll(ResourceType)
	if len(sources) == 0 {
		return nil, nil
	}
	workspaceMetadata := importremote.WorkspaceImportMetadata{
		Resources: make([]importremote.ImportIds, 0),
	}
	var result []importremote.FormattableEntity
	for _, source := range sources {
		data, ok := source.Data.(*sourceClient.EventStreamSource)
		if !ok {
			return nil, fmt.Errorf("unable to cast remote resource to event stream source")
		}
		workspaceMetadata.WorkspaceID = data.WorkspaceID
		workspaceMetadata.Resources = []importremote.ImportIds{
			{
				LocalID:  source.ExternalID,
				RemoteID: source.ID,
			},
		}
		spec, err := h.toImportSpec(
			data,
			source.ExternalID,
			workspaceMetadata,
			inputResolver,
		)
		if err != nil {
			return nil, fmt.Errorf("creating spec: %w", err)
		}
		result = append(result, importremote.FormattableEntity{
			Content:      spec,
			RelativePath: filepath.Join(h.importDir, fmt.Sprintf("%s.yaml", source.ExternalID)),
		})
	}
	return result, nil
}

func (p *Handler) toImportSpec(
	source *sourceClient.EventStreamSource,
	externalID string,
	workspaceMetadata importremote.WorkspaceImportMetadata,
	resolver resolver.ReferenceResolver,
) (*specs.Spec, error) {
	metadata := importremote.Metadata{
		Name: MetadataName,
		Import: importremote.WorkspacesImportMetadata{
			Workspaces: []importremote.WorkspaceImportMetadata{workspaceMetadata},
		},
	}
	metadataMap := make(map[string]any)
	err := mapstructure.Decode(metadata, &metadataMap)
	if err != nil {
		return nil, fmt.Errorf("decoding metadata: %w", err)
	}

	specMap := map[string]any{
		"id":      externalID,
		"name":    source.Name,
		"type":    source.Type,
		"enabled": source.Enabled,
	}

	if source.TrackingPlan != nil {
		tpRef, err := resolver.ResolveToReference(
			dcstate.TrackingPlanResourceType,
			source.TrackingPlan.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("resolving tracking plan reference: %w", err)
		}

		validations := map[string]any{
			TrackingPlanRefYAMLKey:    tpRef,
			TrackingPlanConfigYAMLKey: toTrackingPlanConfigImportSpec(source.TrackingPlan.Config),
		}

		specMap[GovernanceYAMLKey] = map[string]any{
			ValidationsYAMLKey: validations,
		}
	}

	return &specs.Spec{
		Version:  specs.SpecVersion,
		Kind:     ResourceKind,
		Metadata: metadataMap,
		Spec:     specMap,
	}, nil
}

// func (srcResource *sourceResource) addImportMetadata(s *specs.Spec) error {
// 	metadata := importremote.Metadata{}
// 	err := mapstructure.Decode(s.Metadata, &metadata)
// 	if err != nil {
// 		return fmt.Errorf("decoding import metadata: %w", err)
// 	}
// 	lo.ForEach(metadata.Import.Workspaces, func(workspace importremote.WorkspaceImportMetadata, _ int) {
// 		lo.ForEach(workspace.Resources, func(resource importremote.ImportIds, _ int) {
// 			srcResource.ImportMetadata[resources.URN(s.Kind, srcResource.LocalId)] = &WorkspaceRemoteIDMapping{
// 				WorkspaceId: workspace.WorkspaceID,
// 				RemoteId:    resource.RemoteID,
// 			}
// 		})
// 	})
// 	return nil
// }

func toTrackingPlanConfigImportSpec(config *sourceClient.TrackingPlanConfig) map[string]any {
	result := make(map[string]any)

	if config.Track != nil {
		trackSpec := toEventConfigImportSpec(config.Track.EventTypeConfig)
		if config.Track.DropUnplannedEvents != nil {
			trackSpec[DropUnplannedEventsYAMLKey] = *config.Track.DropUnplannedEvents
		}
		result[TrackYAMLKey] = trackSpec
	}

	if config.Identify != nil {
		identifySpec := toEventConfigImportSpec(config.Identify)
		result[IdentifyYAMLKey] = identifySpec
	}

	if config.Group != nil {
		groupSpec := toEventConfigImportSpec(config.Group)
		result[GroupYAMLKey] = groupSpec
	}

	if config.Page != nil {
		pageSpec := toEventConfigImportSpec(config.Page)
		result[PageYAMLKey] = pageSpec
	}

	if config.Screen != nil {
		screenSpec := toEventConfigImportSpec(config.Screen)
		result[ScreenYAMLKey] = screenSpec
	}

	return result
}

func toEventConfigImportSpec(config *sourceClient.EventTypeConfig) map[string]any {
	result := make(map[string]any)
	if config.PropagateViolations != nil {
		result[PropagateViolationsYAMLKey] = *config.PropagateViolations
	}
	if config.DropUnplannedProperties != nil {
		result[DropUnplannedPropertiesYAMLKey] = *config.DropUnplannedProperties
	}
	if config.DropOtherViolations != nil {
		result[DropOtherViolationsYAMLKey] = *config.DropOtherViolations
	}
	return result
}

func mapRemoteToState(source *sourceClient.EventStreamSource, trackingPlanURN *string) (*SourceResource, *SourceStateRemote, bool) {
	if source.ExternalID == "" {
		return nil, nil, true
	}

	input := &SourceResource{
		ID:      source.ExternalID,
		Type:    source.Type,
		Enabled: source.Enabled,
		Name:    source.Name,
	}

	output := &SourceStateRemote{
		ID: source.ID,
	}

	if trackingPlanURN != nil {
		input.Governance = &GovernanceResource{
			Validations: &ValidationsResource{
				TrackingPlanRef: &resources.PropertyRef{
					URN:      *trackingPlanURN,
					Property: "id",
				},
				Config: mapRemoteTPConfigToState(source.TrackingPlan.Config),
			},
		}
		output.TrackingPlanID = source.TrackingPlan.ID
	}

	return input, output, false
}

func parseTrackingPlanRef(ref string) (*resources.PropertyRef, error) {
	// Format: #/tp/group/id
	parts := strings.Split(ref, "/")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid ref format: %s", ref)
	}
	if parts[1] != "tp" {
		return nil, fmt.Errorf("invalid entity type: %s", parts[1])
	}
	return &resources.PropertyRef{
		URN:      resources.URN(parts[3], dcstate.TrackingPlanResourceType),
		Property: "id",
	}, nil
}

func mapStateTPConfigToRemote(config *TrackingPlanConfigResource) (*trackingplanClient.ConnectionConfig, error) {
	result := &trackingplanClient.ConnectionConfig{}

	if config.Track != nil {
		if trackConfig := mapStateTrackConfigToRemote(config.Track); trackConfig != nil {
			result.Track = trackConfig
		}
	}

	if config.Identify != nil {
		if identifyConfig := mapStateEventTypeConfigToRemote(config.Identify); identifyConfig != nil {
			result.Identify = identifyConfig
		}
	}

	if config.Group != nil {
		if groupConfig := mapStateEventTypeConfigToRemote(config.Group); groupConfig != nil {
			result.Group = groupConfig
		}
	}

	if config.Page != nil {
		if pageConfig := mapStateEventTypeConfigToRemote(config.Page); pageConfig != nil {
			result.Page = pageConfig
		}
	}

	if config.Screen != nil {
		if screenConfig := mapStateEventTypeConfigToRemote(config.Screen); screenConfig != nil {
			result.Screen = screenConfig
		}
	}

	return result, nil
}

func mapStateTrackConfigToRemote(config *TrackConfigResource) *trackingplanClient.TrackConfig {
	// Get the common EventTypeConfig using the shared function
	eventTypeConfig := mapStateEventTypeConfigToRemote(config.EventConfigResource)
	if eventTypeConfig == nil {
		return nil
	}

	// Handle track-specific field - only set if explicitly provided
	trackConfig := &trackingplanClient.TrackConfig{
		EventTypeConfig: eventTypeConfig,
	}

	if config.DropUnplannedEvents != nil {
		var allowUnplannedEvents trackingplanClient.StringBool
		if *config.DropUnplannedEvents {
			allowUnplannedEvents = trackingplanClient.False
		} else {
			allowUnplannedEvents = trackingplanClient.True
		}
		trackConfig.AllowUnplannedEvents = &allowUnplannedEvents
	}

	return trackConfig
}

func mapStateEventTypeConfigToRemote(config *EventConfigResource) *trackingplanClient.EventTypeConfig {
	eventTypeConfig := &trackingplanClient.EventTypeConfig{}

	// Only set PropagateValidationErrors if explicitly provided
	if config.PropagateViolations != nil {
		var propagateViolationsStr trackingplanClient.StringBool
		if *config.PropagateViolations {
			propagateViolationsStr = trackingplanClient.True
		} else {
			propagateViolationsStr = trackingplanClient.False
		}
		eventTypeConfig.PropagateValidationErrors = &propagateViolationsStr
	}

	// Only set UnplannedProperties if explicitly provided
	if config.DropUnplannedProperties != nil {
		action := dropToAction(*config.DropUnplannedProperties)
		eventTypeConfig.UnplannedProperties = &action
	}

	// Only set AnyOtherViolations if explicitly provided
	if config.DropOtherViolations != nil {
		action := dropToAction(*config.DropOtherViolations)
		eventTypeConfig.AnyOtherViolation = &action
	}

	return eventTypeConfig
}

func mapRemoteTPConfigToState(config *sourceClient.TrackingPlanConfig) *TrackingPlanConfigResource {
	result := &TrackingPlanConfigResource{}

	if config.Track != nil {
		result.Track = &TrackConfigResource{
			EventConfigResource: buildEventConfigState(config.Track.EventTypeConfig),
			DropUnplannedEvents: config.Track.DropUnplannedEvents,
		}
	}

	if config.Identify != nil {
		result.Identify = buildEventConfigState(config.Identify)
	}

	if config.Group != nil {
		result.Group = buildEventConfigState(config.Group)
	}

	if config.Page != nil {
		result.Page = buildEventConfigState(config.Page)
	}

	if config.Screen != nil {
		result.Screen = buildEventConfigState(config.Screen)
	}

	return result
}

func buildEventConfigState(config *sourceClient.EventTypeConfig) *EventConfigResource {
	if config == nil {
		return &EventConfigResource{}
	}

	return &EventConfigResource{
		PropagateViolations:     config.PropagateViolations,
		DropUnplannedProperties: config.DropUnplannedProperties,
		DropOtherViolations:     config.DropOtherViolations,
	}
}

func dropToAction(drop bool) trackingplanClient.Action {
	if drop {
		return trackingplanClient.Drop
	}
	return trackingplanClient.Forward
}
