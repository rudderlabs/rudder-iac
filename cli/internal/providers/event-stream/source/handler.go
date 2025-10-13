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
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	dcstate "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

type Handler struct {
	resources map[string]*sourceResource
	client    esClient.EventStreamStore
	importDir string
}

func NewHandler(client esClient.EventStreamStore, importDir string) *Handler {
	return &Handler{resources: make(map[string]*sourceResource), client: client, importDir: filepath.Join(importDir, ImportPath)}
}

func (h *Handler) LoadSpec(_ string, s *specs.Spec) error {
	spec := &sourceSpec{}
	if err := mapstructure.Decode(s.Spec, spec); err != nil {
		return fmt.Errorf("decoding spec: %w", err)
	}
	if _, exists := h.resources[spec.LocalId]; exists {
		return fmt.Errorf("event stream source with id %s already exists", spec.LocalId)
	}
	// Default enabled to true when not specified in the spec
	enabled := true
	if spec.Enabled != nil {
		enabled = *spec.Enabled
	}
	sourceResource := &sourceResource{
		LocalId:          spec.LocalId,
		Name:             spec.Name,
		SourceDefinition: spec.SourceDefinition,
		Enabled:          enabled,
		Governance:       &governanceResource{},
	}
	if err := h.loadTrackingPlanSpec(spec, sourceResource); err != nil {
		return err
	}
	h.resources[spec.LocalId] = sourceResource
	return nil
}

func (h *Handler) loadTrackingPlanSpec(spec *sourceSpec, sourceResource *sourceResource) error {
	if spec.Governance == nil || spec.Governance.TrackingPlan == nil {
		return nil
	}
	if spec.Governance.TrackingPlan != nil && !config.GetConfig().ExperimentalFlags.StatelessCLI {
		return fmt.Errorf("governance.validations is supported only in stateless CLI mode")
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

	sourceResource.Governance.TrackingPlan = &trackingPlanResource{
		Ref:    trackingPlanRef,
		Config: &trackingPlanConfigResource{},
	}

	config := sourceResource.Governance.TrackingPlan.Config
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

func validateSource(source *sourceResource) error {
	if source.LocalId == "" {
		return fmt.Errorf("id is required")
	}
	if source.Name == "" {
		return fmt.Errorf("name is required")
	}
	if source.SourceDefinition == "" {
		return fmt.Errorf("type is required")
	}
	if !slices.Contains(sourceDefinitions, source.SourceDefinition) {
		return fmt.Errorf("type '%s' is invalid, must be one of: %v", source.SourceDefinition, sourceDefinitions)
	}
	return nil
}

func (h *Handler) Validate() error {
	for _, source := range h.resources {
		if err := validateSource(source); err != nil {
			return fmt.Errorf("validating event stream source spec: %w", err)
		}
	}
	return nil
}

func (h *Handler) GetResources() ([]*resources.Resource, error) {
	result := make([]*resources.Resource, 0, len(h.resources))
	for _, s := range h.resources {
		data := resources.ResourceData{
			NameKey:             s.Name,
			EnabledKey:          s.Enabled,
			SourceDefinitionKey: s.SourceDefinition,
		}
		if s.Governance.TrackingPlan != nil {
			data[TrackingPlanKey] = s.Governance.TrackingPlan.Ref
			data[TrackingPlanConfigKey] = buildTrackingPlanConfigState(s.Governance.TrackingPlan.Config)
		}
		r := resources.NewResource(s.LocalId, ResourceType, data, []string{})
		result = append(result, r)
	}
	return result, nil
}

func buildTrackingPlanConfigState(config *trackingPlanConfigResource) map[string]interface{} {
	result := make(map[string]interface{})
	if config.Track != nil {
		trackState := buildEventConfigState(config.Track.EventConfigResource)
		if config.Track.DropUnplannedEvents != nil {
			trackState[DropUnplannedEventsKey] = *config.Track.DropUnplannedEvents
		}
		result[TrackKey] = trackState
	}
	if config.Identify != nil {
		identifyState := buildEventConfigState(config.Identify)
		result[IdentifyKey] = identifyState
	}
	if config.Group != nil {
		groupState := buildEventConfigState(config.Group)
		result[GroupKey] = groupState
	}
	if config.Page != nil {
		pageState := buildEventConfigState(config.Page)
		result[PageKey] = pageState
	}
	if config.Screen != nil {
		screenState := buildEventConfigState(config.Screen)
		result[ScreenKey] = screenState
	}
	return result
}

func buildEventConfigState(config *EventConfigResource) map[string]interface{} {
	result := make(map[string]interface{})
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

func (h *Handler) Create(ctx context.Context, id string, data resources.ResourceData) (*resources.ResourceData, error) {
	createRequest := &sourceClient.CreateSourceRequest{
		ExternalID: id,
		Name:       data[NameKey].(string),
		Type:       data[SourceDefinitionKey].(string),
		Enabled:    data[EnabledKey].(bool),
	}
	resp, err := h.client.Create(ctx, createRequest)
	if err != nil {
		return nil, fmt.Errorf("creating event stream source: %w", err)
	}
	trackingPlanID, ok := data[TrackingPlanKey].(string)
	if ok {
		err := h.linkTrackingPlan(ctx, trackingPlanID, resp.ID, data)
		if err != nil {
			return nil, fmt.Errorf("linking tracking plan to event stream source: %w", err)
		}
	}
	return toResourceData(resp.ID, trackingPlanID), nil
}

func (h *Handler) Update(ctx context.Context, id string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	if state[SourceDefinitionKey] != data[SourceDefinitionKey] {
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
	newTrackingPlanID, newHasTP := data[TrackingPlanKey].(string)
	if newHasTP {
		return toResourceData(remoteID, newTrackingPlanID), nil
	}
	return toResourceData(remoteID, ""), nil
}

func (h *Handler) updateSource(ctx context.Context, remoteID string, data, state resources.ResourceData) error {
	needsUpdate := state[NameKey] != data[NameKey] || state[EnabledKey] != data[EnabledKey]
	if !needsUpdate {
		return nil
	}
	updateRequest := &sourceClient.UpdateSourceRequest{
		Name:    data[NameKey].(string),
		Enabled: data[EnabledKey].(bool),
	}
	_, err := h.client.Update(ctx, remoteID, updateRequest)
	if err != nil {
		return fmt.Errorf("updating event stream source: %w", err)
	}
	return nil
}

func (h *Handler) updateTrackingPlanConnection(ctx context.Context, remoteID string, data, state resources.ResourceData) error {
	if data[TrackingPlanKey] == nil && state[TrackingPlanIDKey] == nil {
		return nil
	}

	currentTrackingPlanID, currentHasTP := state[TrackingPlanIDKey].(string)
	newTrackingPlanID, newHasTP := data[TrackingPlanKey].(string)

	switch {
	case !currentHasTP && newHasTP:
		return h.linkTrackingPlan(ctx, newTrackingPlanID, remoteID, data)
	case currentHasTP && !newHasTP:
		return h.unlinkTrackingPlan(ctx, currentTrackingPlanID, remoteID)
	case currentHasTP && newHasTP:
		return h.updateExistingTrackingPlanConnection(ctx, currentTrackingPlanID, newTrackingPlanID, remoteID, data, state)
	default:
		return nil
	}
}

func (h *Handler) linkTrackingPlan(ctx context.Context, trackingPlanID, remoteID string, data resources.ResourceData) error {
	config, ok := data[TrackingPlanConfigKey].(map[string]interface{})
	if !ok {
		return fmt.Errorf("missing tracking plan config in resource data")
	}

	remoteConfig, err := mapStateTPConfigToRemote(config)
	if err != nil {
		return fmt.Errorf("invalid tracking plan config: %w", err)
	}

	err = h.client.LinkTP(ctx, trackingPlanID, remoteID, remoteConfig)
	if err != nil {
		return fmt.Errorf("linking tracking plan to event stream source: %w", err)
	}
	return nil
}

func (h *Handler) unlinkTrackingPlan(ctx context.Context, trackingPlanID, remoteID string) error {
	err := h.client.UnlinkTP(ctx, trackingPlanID, remoteID)
	if err != nil {
		return fmt.Errorf("unlinking tracking plan from event stream source: %w", err)
	}
	return nil
}

func (h *Handler) updateExistingTrackingPlanConnection(ctx context.Context, currentTrackingPlanID, newTrackingPlanID, remoteID string, data, state resources.ResourceData) error {
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

func (h *Handler) updateTrackingPlanConfig(ctx context.Context, trackingPlanID, remoteID string, data, state resources.ResourceData) error {
	currentConfig, ok := state[TrackingPlanConfigKey].(map[string]interface{})
	if !ok {
		return fmt.Errorf("missing tracking plan config in state")
	}

	newConfig, ok := data[TrackingPlanConfigKey].(map[string]interface{})
	if !ok {
		return fmt.Errorf("missing tracking plan config in resource data")
	}

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

func (h *Handler) LoadState(ctx context.Context) (*state.State, error) {
	sources, err := h.client.GetSources(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting event stream sources: %w", err)
	}
	st := state.EmptyState()
	for _, source := range sources {
		resourceState, skip := mapRemoteToState(&source, "")
		if skip {
			continue
		}
		st.AddResource(resourceState)
	}
	return st, nil
}

func (h *Handler) LoadResourcesFromRemote(ctx context.Context) (*resources.ResourceCollection, error) {
	collection := resources.NewResourceCollection()
	sources, err := h.client.GetSources(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting event stream sources: %w", err)
	}
	resourceMap := make(map[string]*resources.RemoteResource)
	for _, source := range sources {
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
		var trackingPlanURN string
		var err error
		if source.TrackingPlan != nil {
			trackingPlanURN, err = collection.GetURNByID(dcstate.TrackingPlanResourceType, source.TrackingPlan.ID)
			if err != nil {
				return nil, fmt.Errorf("get urn by id: %w", err)
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

func (h *Handler) Delete(ctx context.Context, id string, state resources.ResourceData) error {
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

func (h *Handler) Import(_ context.Context, _ string, data resources.ResourceData, _ string) (*resources.ResourceData, error) {
	return nil, fmt.Errorf("importing event stream source is not supported")
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
			Reference: fmt.Sprintf("#/%s/%s/%s",
				ResourceKind,
				MetadataName,
				externalID,
			),
			Data: &source,
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
		spec, err := h.toImportSpec(data, source.ExternalID, workspaceMetadata, collection)
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

func (p *Handler) toImportSpec(source *sourceClient.EventStreamSource, externalID string, workspaceMetadata importremote.WorkspaceImportMetadata, collection *resources.ResourceCollection) (*specs.Spec, error) {
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
		trackingPlanResource, exists := collection.GetByID(dcstate.TrackingPlanResourceType, source.TrackingPlan.ID)
		if !exists {
			return nil, fmt.Errorf("tracking plan with ID %s not found in collection", source.TrackingPlan.ID)
		}

		validations := map[string]any{
			TrackingPlanRefYAMLKey:     fmt.Sprintf("#/tracking-plans/%s/%s", dcstate.TrackingPlanResourceType, trackingPlanResource.ExternalID),
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

func mapRemoteToState(source *sourceClient.EventStreamSource, trackingPlanURN string) (*state.ResourceState, bool) {
	if source.ExternalID == "" {
		return nil, true
	}
	input := resources.ResourceData{
		NameKey:             source.Name,
		EnabledKey:          source.Enabled,
		SourceDefinitionKey: source.Type,
	}
	if source.TrackingPlan != nil {
		input[TrackingPlanKey] = &resources.PropertyRef{
			URN:      trackingPlanURN,
			Property: "id",
		}
		input[TrackingPlanConfigKey] = mapRemoteTPConfigToState(source.TrackingPlan.Config)
	}
	var output *resources.ResourceData
	if source.TrackingPlan != nil {
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
	// Format: #/tracking-plans/group/id
	parts := strings.Split(ref, "/")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid ref format: %s", ref)
	}
	if parts[1] != "tracking-plans" {
		return nil, fmt.Errorf("invalid entity type: %s", parts[1])
	}
	return &resources.PropertyRef{
		URN:      resources.URN(parts[3], dcstate.TrackingPlanResourceType),
		Property: "id",
	}, nil
}

func mapStateTPConfigToRemote(config map[string]interface{}) (*trackingplanClient.ConnectionConfig, error) {
	result := &trackingplanClient.ConnectionConfig{}

	if trackConfig := mapStateTrackConfigToRemote(config, TrackKey); trackConfig != nil {
		result.Track = trackConfig
	}

	if identifyConfig := mapStateEventTypeConfigToRemote(config, IdentifyKey); identifyConfig != nil {
		result.Identify = identifyConfig
	}

	if groupConfig := mapStateEventTypeConfigToRemote(config, GroupKey); groupConfig != nil {
		result.Group = groupConfig
	}

	if pageConfig := mapStateEventTypeConfigToRemote(config, PageKey); pageConfig != nil {
		result.Page = pageConfig
	}

	if screenConfig := mapStateEventTypeConfigToRemote(config, ScreenKey); screenConfig != nil {
		result.Screen = screenConfig
	}

	return result, nil
}

func mapStateTrackConfigToRemote(config map[string]interface{}, key string) *trackingplanClient.TrackConfig {
	eventConfig, exists := config[key]
	if !exists {
		return nil
	}

	configMap, ok := eventConfig.(map[string]interface{})
	if !ok {
		return nil
	}

	// Get the common EventTypeConfig using the shared function
	eventTypeConfig := mapStateEventTypeConfigToRemote(config, key)
	if eventTypeConfig == nil {
		return nil
	}

	// Handle track-specific field - only set if explicitly provided
	trackConfig := &trackingplanClient.TrackConfig{
		EventTypeConfig: eventTypeConfig,
	}

	if val, exists := configMap[DropUnplannedEventsKey]; exists {
		if dropUnplannedEvents, ok := val.(bool); ok {
			var allowUnplannedEvents trackingplanClient.StringBool
			if dropUnplannedEvents {
				allowUnplannedEvents = trackingplanClient.False
			} else {
				allowUnplannedEvents = trackingplanClient.True
			}
			trackConfig.AllowUnplannedEvents = &allowUnplannedEvents
		}
	}

	return trackConfig
}

func mapStateEventTypeConfigToRemote(config map[string]interface{}, key string) *trackingplanClient.EventTypeConfig {
	eventConfig, exists := config[key]
	if !exists {
		return nil
	}

	configMap, ok := eventConfig.(map[string]interface{})
	if !ok {
		return nil
	}

	eventTypeConfig := &trackingplanClient.EventTypeConfig{}

	// Only set PropagateValidationErrors if explicitly provided
	if val, exists := configMap[PropagateViolationsKey]; exists {
		if propagateViolations, ok := val.(bool); ok {
			var propagateViolationsStr trackingplanClient.StringBool
			if propagateViolations {
				propagateViolationsStr = trackingplanClient.True
			} else {
				propagateViolationsStr = trackingplanClient.False
			}
			eventTypeConfig.PropagateValidationErrors = &propagateViolationsStr
		}
	}

	// Only set UnplannedProperties if explicitly provided
	if val, exists := configMap[DropUnplannedPropertiesKey]; exists {
		if dropUnplannedProperties, ok := val.(bool); ok {
			action := dropToAction(dropUnplannedProperties)
			eventTypeConfig.UnplannedProperties = &action
		}
	}

	// Only set AnyOtherViolations if explicitly provided
	if val, exists := configMap[DropOtherViolationsKey]; exists {
		if dropOtherViolations, ok := val.(bool); ok {
			action := dropToAction(dropOtherViolations)
			eventTypeConfig.AnyOtherViolation = &action
		}
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
