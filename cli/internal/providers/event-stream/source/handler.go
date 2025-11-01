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
	*provider.BaseHandler[sourceSpec, SourceResource]
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

func (h *handlerImpl) Create(ctx context.Context, data *SourceResource) (*resources.ResourceData, error) {
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

	trackingPlanID := ""
	if data.Governance != nil && data.Governance.Validations != nil && data.Governance.Validations.TrackingPlanRef != nil {
		trackingPlanRef := data.Governance.Validations.TrackingPlanRef
		trackingPlanID = trackingPlanRef.Resolved.Value
		if trackingPlanRef != nil {
			err := h.linkTrackingPlan(ctx, trackingPlanRef.Resolved.Value, resp.ID, data)
			if err != nil {
				return nil, fmt.Errorf("linking tracking plan to event stream source: %w", err)
			}
		}
	}
	return toResourceData(resp.ID, trackingPlanID), nil
}

func (h *handlerImpl) Update(ctx context.Context, data *SourceResource, state resources.ResourceData) (*resources.ResourceData, error) {
	if state[SourceDefinitionKey] != data.Type {
		return nil, fmt.Errorf("type cannot be changed")
	}
	remoteID, ok := state[IDKey].(string)
	if !ok {
		return nil, fmt.Errorf("missing id in resource data")
	}
	err := h.updateSource(ctx, remoteID, data, state)
	if err != nil {
		return nil, fmt.Errorf("updating event stream source: %w", err)
	}
	if err := h.updateTrackingPlanConnection(ctx, remoteID, data, state); err != nil {
		return nil, err
	}
	if data.Governance != nil && data.Governance.Validations != nil && data.Governance.Validations.TrackingPlanRef != nil {
		return toResourceData(remoteID, data.Governance.Validations.TrackingPlanRef.Resolved.Value), nil
	}
	return toResourceData(remoteID, ""), nil
}

func (h *handlerImpl) updateSource(ctx context.Context, remoteID string, data *SourceResource, state resources.ResourceData) error {
	needsUpdate := state[NameKey] != data.Name || state[EnabledKey] != data.Enabled
	if !needsUpdate {
		return nil
	}
	updateRequest := &sourceClient.UpdateSourceRequest{
		Name:    data.Name,
		Enabled: data.Enabled,
	}
	_, err := h.client.Update(ctx, remoteID, updateRequest)
	if err != nil {
		return fmt.Errorf("updating event stream source: %w", err)
	}
	return nil
}

func (h *handlerImpl) updateTrackingPlanConnection(ctx context.Context, remoteID string, data *SourceResource, state resources.ResourceData) error {
	var trackingPlanRef *resources.PropertyRef = nil
	if data.Governance != nil && data.Governance.Validations != nil && data.Governance.Validations.TrackingPlanRef != nil {
		trackingPlanRef = data.Governance.Validations.TrackingPlanRef
	}

	if trackingPlanRef == nil && state[TrackingPlanIDKey] == nil {
		return nil
	}

	currentTrackingPlanID, currentHasTP := state[TrackingPlanIDKey].(string)
	if trackingPlanRef != nil {
		newTPID := trackingPlanRef.Resolved.Value
		if currentHasTP {
			return h.updateExistingTrackingPlanConnection(ctx, currentTrackingPlanID, newTPID, remoteID, data, state)

		} else {
			return h.linkTrackingPlan(ctx, newTPID, remoteID, data)

		}
	} else if currentHasTP {
		return h.unlinkTrackingPlan(ctx, currentTrackingPlanID, remoteID)
	}

	return nil
}

func (h *handlerImpl) linkTrackingPlan(ctx context.Context, trackingPlanID, remoteID string, data *SourceResource) error {
	remoteConfig, err := mapStateTPConfigToRemote(data.Governance.Validations.Config)
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

func (h *handlerImpl) updateExistingTrackingPlanConnection(ctx context.Context, currentTrackingPlanID, newTrackingPlanID, remoteID string, data *SourceResource, state resources.ResourceData) error {
	if currentTrackingPlanID == newTrackingPlanID {
		return h.updateTrackingPlanConfig(ctx, currentTrackingPlanID, remoteID, data, state)
	}

	if err := h.unlinkTrackingPlan(ctx, currentTrackingPlanID, remoteID); err != nil {
		return fmt.Errorf("unlinking old tracking plan: %w", err)
	}

	if err := h.linkTrackingPlan(ctx, newTrackingPlanID, remoteID, data); err != nil {
		return fmt.Errorf("linking new tracking plan: %w", err)
	}

	return nil
}

func (h *handlerImpl) updateTrackingPlanConfig(ctx context.Context, trackingPlanID, remoteID string, data *SourceResource, state resources.ResourceData) error {
	currentConfig, ok := state[TrackingPlanConfigKey].(*TrackingPlanConfigResource)
	if !ok {
		return fmt.Errorf("missing tracking plan config in state")
	}

	newConfig := data.Governance.Validations.Config
	// NOTE: DeepEqual will fail until ResourceData has been converted to trackingPlanConfigResource
	if !reflect.DeepEqual(newConfig, currentConfig) {
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
		resourceState, skip := mapRemoteToState(&source, trackingPlanURN)
		if skip {
			continue
		}
		urn := resources.URN(esResource.ExternalID, ResourceType)
		s.Resources[urn] = resourceState
	}
	return s, nil
}

func (h *handlerImpl) Delete(ctx context.Context, id string, state resources.ResourceData) error {
	sourceID, ok := state[IDKey].(string)
	if !ok {
		return fmt.Errorf("missing id in resource data")
	}
	trackingPlanID, ok := state[TrackingPlanIDKey].(string)
	if ok {
		err := h.client.UnlinkTP(ctx, trackingPlanID, sourceID)
		if err != nil {
			return fmt.Errorf("unlinking tracking plan from event stream source: %w", err)
		}
	}
	remoteID, ok := state[IDKey].(string)
	if !ok {
		return fmt.Errorf("missing id in resource data")
	}
	err := h.client.Delete(ctx, remoteID)
	if err != nil {
		return fmt.Errorf("deleting event stream source: %w", err)
	}
	return nil
}

func (h *handlerImpl) Import(ctx context.Context, data *SourceResource, remoteId string) (*resources.ResourceData, error) {
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

	// Build state from existing source to compare with desired data
	existingState := resources.ResourceData{
		IDKey:               remoteId,
		NameKey:             existingSource.Name,
		EnabledKey:          existingSource.Enabled,
		SourceDefinitionKey: existingSource.Type,
	}

	// If there's a tracking plan on the existing source, include it in state
	if existingSource.TrackingPlan != nil {
		existingState[TrackingPlanIDKey] = existingSource.TrackingPlan.ID
		existingState[TrackingPlanConfigKey] = mapRemoteTPConfigToState(existingSource.TrackingPlan.Config)
	}

	// Update the source if there are differences
	result, err := h.Update(ctx, data, existingState)
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
		IDKey:               externalID,
		NameKey:             source.Name,
		SourceDefinitionKey: source.Type,
		EnabledKey:          source.Enabled,
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

func mapRemoteToState(source *sourceClient.EventStreamSource, trackingPlanURN *string) (*state.ResourceState, bool) {
	if source.ExternalID == "" {
		return nil, true
	}
	input := resources.ResourceData{
		NameKey:             source.Name,
		EnabledKey:          source.Enabled,
		SourceDefinitionKey: source.Type,
	}
	var output *resources.ResourceData
	if trackingPlanURN != nil {
		input[TrackingPlanKey] = &resources.PropertyRef{
			URN:      *trackingPlanURN,
			Property: "id",
		}
		input[TrackingPlanConfigKey] = source.TrackingPlan.Config
		output = toResourceData(source.ID, source.TrackingPlan.ID)
	} else {
		output = toResourceData(source.ID, "")
	}
	return &state.ResourceState{
		Type:   ResourceType,
		ID:     source.ExternalID,
		Input:  input,
		Output: *output,
	}, false
}

func toResourceData(sourceID string, trackingPlanID string) *resources.ResourceData {
	result := map[string]interface{}{
		IDKey: sourceID,
	}
	if trackingPlanID != "" {
		result[TrackingPlanIDKey] = trackingPlanID
	}
	resourceData := resources.ResourceData(result)
	return &resourceData
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

func mapRemoteTPConfigToState(config *sourceClient.TrackingPlanConfig) map[string]interface{} {
	result := make(map[string]interface{})

	if config.Track != nil {
		trackMap := buildStateEventConfigMap(config.Track.EventTypeConfig)
		if config.Track.DropUnplannedEvents != nil {
			trackMap[DropUnplannedEventsKey] = *config.Track.DropUnplannedEvents
		}
		result[TrackKey] = trackMap
	}

	if config.Identify != nil {
		result[IdentifyKey] = buildStateEventConfigMap(config.Identify)
	}

	if config.Group != nil {
		result[GroupKey] = buildStateEventConfigMap(config.Group)
	}

	if config.Page != nil {
		result[PageKey] = buildStateEventConfigMap(config.Page)
	}

	if config.Screen != nil {
		result[ScreenKey] = buildStateEventConfigMap(config.Screen)
	}

	return result
}

func buildStateEventConfigMap(config *sourceClient.EventTypeConfig) map[string]interface{} {
	if config == nil {
		return map[string]interface{}{}
	}

	result := map[string]interface{}{}

	// Only include fields that are explicitly set
	if config.PropagateViolations != nil {
		result[PropagateViolationsKey] = *config.PropagateViolations
	}

	if config.DropUnplannedProperties != nil {
		result[DropUnplannedPropertiesKey] = *config.DropUnplannedProperties
	}

	if config.DropOtherViolations != nil {
		result[DropOtherViolationsKey] = *config.DropOtherViolations
	}

	return result
}

func dropToAction(drop bool) trackingplanClient.Action {
	if drop {
		return trackingplanClient.Drop
	}
	return trackingplanClient.Forward
}
