package source

import (
	"context"
	"fmt"
	"slices"

	"github.com/go-viper/mapstructure/v2"
	sourceClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

type Handler struct {
	resources map[string]*sourceSpec
	client    sourceClient.SourceStore
}

func NewHandler(client sourceClient.SourceStore) *Handler {
	return &Handler{resources: make(map[string]*sourceSpec), client: client}
}

func (h *Handler) LoadSpec(_ string, s *specs.Spec) error {
	source := &sourceSpec{}
	if err := mapstructure.Decode(s.Spec, source); err != nil {
		return fmt.Errorf("decoding spec: %w", err)
	}
	if _, exists := h.resources[source.LocalId]; exists {
		return fmt.Errorf("event stream source with id %s already exists", source.LocalId)
	}
	h.resources[source.LocalId] = source
	return nil
}

func validateSource(source *sourceSpec) error {
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
		r := resources.NewResource(s.LocalId, ResourceType, data, []string{})
		result = append(result, r)
	}
	return result, nil
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
	return mapRemoteToResourceData(resp), nil
}

func (h *Handler) Update(ctx context.Context, id string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	if state[SourceDefinitionKey] != data[SourceDefinitionKey] {
		return nil, fmt.Errorf("type cannot be changed")
	}
	remoteID, ok := state[IDKey].(string)
	if !ok {
		return nil, fmt.Errorf("missing id in resource data")
	}
	updateRequest := &sourceClient.UpdateSourceRequest{
		Name:    data[NameKey].(string),
		Enabled: data[EnabledKey].(bool),
	}
	resp, err := h.client.Update(ctx, remoteID, updateRequest)
	if err != nil {
		return nil, fmt.Errorf("updating event stream source: %w", err)
	}
	return mapRemoteToResourceData(resp), nil
}

func (h *Handler) LoadState(ctx context.Context) (*state.State, error) {
	sources, err := h.client.GetSources(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting event stream sources: %w", err)
	}
	st := state.EmptyState()
	for _, source := range sources {
		if source.ExternalID != "" {
			st.AddResource(mapRemoteToState(source))
		}
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
		if source.ExternalID != "" {
			resourceState := mapRemoteToState(source)
			urn := resources.URN(esResource.ExternalID, ResourceType)
			s.Resources[urn] = resourceState
		}
	}
	return s, nil
}

func (h *Handler) Delete(ctx context.Context, id string, state resources.ResourceData) error {
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

func mapRemoteToState(source sourceClient.EventStreamSource) *state.ResourceState {
	return &state.ResourceState{
		Type: ResourceType,
		ID:   source.ExternalID,
		Input: *mapRemoteToResourceData(&source),
		Output: resources.ResourceData{
			IDKey: source.ID,
		},
	}
}

func mapRemoteToResourceData(source *sourceClient.EventStreamSource) *resources.ResourceData {
	return &resources.ResourceData{
		NameKey:             source.Name,
		EnabledKey:          source.Enabled,
		SourceDefinitionKey: source.Type,
	}
}
