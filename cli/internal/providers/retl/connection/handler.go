package connection

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-viper/mapstructure/v2"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	esConnection "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/connection"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

var log = logger.New("retl-connection")

// Handler manages rETL connections the way the event stream connection handler
// does: one graph resource per spec entry, an endpoint change handled as a
// delete-then-create replacement, and an immutable config change rejected with
// the remedy rather than silently not applied.
type Handler struct {
	client    retlClient.RETLStore
	resources map[string]*connectionResource
}

func NewHandler(client retlClient.RETLStore) *Handler {
	return &Handler{client: client, resources: make(map[string]*connectionResource)}
}

func (h *Handler) ParseSpec(_ string, s *specs.Spec) (*specs.ParsedSpec, error) {
	raw, ok := s.Spec[ConnectionsKey].([]any)
	if !ok {
		return nil, fmt.Errorf("connections not found in rETL connections spec")
	}

	entries := make([]specs.URNEntry, 0, len(raw))
	for i, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("connection at index %d is not a map", i)
		}
		id, ok := entry["id"].(string)
		if !ok {
			return nil, fmt.Errorf("id not found in connection at index %d", i)
		}
		entries = append(entries, specs.URNEntry{
			URN:             resources.URN(id, ResourceType),
			JSONPointerPath: fmt.Sprintf("/spec/connections/%d/id", i),
		})
	}
	return &specs.ParsedSpec{URNs: entries}, nil
}

// LoadSpec decodes strictly at every level, so a misspelt key anywhere under
// config is an error rather than a silently dropped setting.
func (h *Handler) LoadSpec(_ string, s *specs.Spec) error {
	spec := &ConnectionsSpec{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused: true,
		Result:      spec,
	})
	if err != nil {
		return fmt.Errorf("creating decoder: %w", err)
	}
	if err := decoder.Decode(s.Spec); err != nil {
		return fmt.Errorf("decoding rETL connections spec: %w", err)
	}

	for _, entry := range spec.Connections {
		if _, ok := h.resources[entry.LocalID]; ok {
			return fmt.Errorf("rETL connection with id %s already exists", entry.LocalID)
		}
		resource, err := loadConnection(entry)
		if err != nil {
			return err
		}
		if err := resource.addImportMetadata(s); err != nil {
			return fmt.Errorf("loading import metadata: %w", err)
		}
		h.resources[entry.LocalID] = resource
	}
	return nil
}

// loadConnection only parses references and converts the config; required-field
// and cross-field checks belong to the validation rules.
func loadConnection(c ConnectionSpec) (*connectionResource, error) {
	sourceRef, err := parseSourceRef(c.Source)
	if err != nil {
		return nil, fmt.Errorf("connection %q: parsing source reference: %w", c.LocalID, err)
	}
	destinationRef, err := esConnection.ParseDestinationRef(c.Destination)
	if err != nil {
		return nil, fmt.Errorf("connection %q: parsing destination reference: %w", c.LocalID, err)
	}
	config, err := configToMap(c.Config)
	if err != nil {
		return nil, fmt.Errorf("connection %q: %w", c.LocalID, err)
	}

	enabled := true
	if c.Enabled != nil {
		enabled = *c.Enabled
	}

	return &connectionResource{
		LocalID:        c.LocalID,
		Source:         sourceRef,
		Destination:    destinationRef,
		Enabled:        enabled,
		Config:         config,
		ImportMetadata: make(map[string]*WorkspaceRemoteIDMapping),
	}, nil
}

// GetResources emits each connection as a data map; the graph derives its
// dependency edges from the two endpoint refs inside it.
func (h *Handler) GetResources() ([]*resources.Resource, error) {
	result := make([]*resources.Resource, 0, len(h.resources))
	for _, c := range h.resources {
		data := resources.ResourceData{
			SourceKey:      c.Source,
			DestinationKey: c.Destination,
			EnabledKey:     c.Enabled,
			ConfigKey:      c.Config,
		}
		opts := []resources.ResourceOpts{
			resources.WithResourceFileMetadata(fmt.Sprintf("#%s:%s", ResourceKind, c.LocalID)),
		}
		urn := resources.URN(c.LocalID, ResourceType)
		if importMetadata, ok := c.ImportMetadata[urn]; ok {
			opts = []resources.ResourceOpts{
				resources.WithResourceImportMetadata(importMetadata.RemoteID, importMetadata.WorkspaceID),
			}
		}
		result = append(result, resources.NewResource(c.LocalID, ResourceType, data, []string{}, opts...))
	}
	return result, nil
}

func (c *connectionResource) addImportMetadata(s *specs.Spec) error {
	metadata, err := s.CommonMetadata()
	if err != nil {
		return err
	}
	if metadata.Import == nil {
		return nil
	}
	c.applyImportManifest(metadata.Import)
	return nil
}

func (c *connectionResource) applyImportManifest(m *specs.WorkspacesImportMetadata) {
	for _, workspace := range m.Workspaces {
		for _, resource := range workspace.Resources {
			// Support both the URN field (new) and the LocalID field (legacy).
			urn := resource.URN
			if urn == "" {
				urn = resources.URN(resource.LocalID, ResourceType)
			}
			c.ImportMetadata[urn] = &WorkspaceRemoteIDMapping{
				WorkspaceID: workspace.WorkspaceID,
				RemoteID:    resource.RemoteID,
			}
		}
	}
}

// LoadImportMetadata replicates the aggregated manifest into every loaded
// connection; each one reads only its own URN back at graph time.
func (h *Handler) LoadImportMetadata(m *specs.WorkspacesImportMetadata) error {
	if m == nil {
		return nil
	}
	for _, c := range h.resources {
		c.applyImportManifest(m)
	}
	return nil
}

func (h *Handler) Create(ctx context.Context, id string, data resources.ResourceData) (*resources.ResourceData, error) {
	request, err := toCreateRequest(data)
	if err != nil {
		return nil, fmt.Errorf("building create request for rETL connection %q: %w", id, err)
	}
	request.ExternalID = id

	created, err := h.client.CreateConnection(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("creating rETL connection %q: %w", id, err)
	}
	return toResourceData(created), nil
}

// Update changes the mutable fields in one PUT; toUpdateRequest rejects a change
// to a field the API refuses on update, as every other handler does. An
// endpoint change is a replacement — delete then create — because the backend
// allows one connection per source–destination pair, and the create response
// is what lands in state.
func (h *Handler) Update(ctx context.Context, id string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	remoteID, ok := state[IDKey].(string)
	if !ok || remoteID == "" {
		return nil, fmt.Errorf("connection %q: missing id in state", id)
	}
	sourceID, err := endpointIDFromData(data, SourceKey)
	if err != nil {
		return nil, fmt.Errorf("connection %q: %w", id, err)
	}
	destinationID, err := endpointIDFromData(data, DestinationKey)
	if err != nil {
		return nil, fmt.Errorf("connection %q: %w", id, err)
	}
	enabled, err := enabledFromData(data)
	if err != nil {
		return nil, fmt.Errorf("connection %q: %w", id, err)
	}
	desired, err := configFromMap(data[ConfigKey])
	if err != nil {
		return nil, fmt.Errorf("connection %q: %w", id, err)
	}
	stored, err := configFromMap(state[ConfigKey])
	if err != nil {
		return nil, fmt.Errorf("connection %q: reading stored connection config: %w", id, err)
	}

	if sourceID != state[SourceIDKey] || destinationID != state[DestinationIDKey] {
		// Build the create body before the delete: a config the API would
		// refuse must not cost the live connection.
		if _, err := toCreateRequest(data); err != nil {
			return nil, fmt.Errorf("connection %q: %w", id, err)
		}
		log.Warn("replacing rETL connection after an endpoint change",
			"connection", id, "connectionId", remoteID)
		if err := h.Delete(ctx, id, state); err != nil {
			return nil, err
		}
		created, err := h.Create(ctx, id, data)
		if err != nil {
			return nil, fmt.Errorf("recreating rETL connection %q after an endpoint change (the previous connection was deleted): %w", id, err)
		}
		return created, nil
	}

	if enabled == state[EnabledKey] && reflect.DeepEqual(desired, stored) {
		return toResourceData(&retlClient.RETLConnection{ID: remoteID, SourceID: sourceID, DestinationID: destinationID}), nil
	}

	request, err := toUpdateRequest(data, state)
	if err != nil {
		return nil, fmt.Errorf("connection %q: %w", id, err)
	}
	updated, err := h.client.UpdateConnection(ctx, remoteID, request)
	if err != nil {
		return nil, fmt.Errorf("updating rETL connection %q: %w", id, err)
	}
	return toResourceData(updated), nil
}

// Delete removes the connection only; the endpoints are their own resources.
func (h *Handler) Delete(ctx context.Context, id string, state resources.ResourceData) error {
	remoteID, ok := state[IDKey].(string)
	if !ok || remoteID == "" {
		return fmt.Errorf("connection %q: missing id in state", id)
	}
	if err := h.client.DeleteConnection(ctx, remoteID); err != nil {
		return fmt.Errorf("deleting rETL connection %q: %w", id, err)
	}
	return nil
}

// Preview is a SQL model capability and single-resource import is exposed for
// SQL models only; connections are adopted through the workspace-wide import.
func (h *Handler) Preview(_ context.Context, _ string, _ resources.ResourceData, _ int) ([]map[string]any, error) {
	return nil, fmt.Errorf("preview is not supported for rETL connections")
}

func (h *Handler) FetchImportData(_ context.Context, _ specs.ImportIds) (writer.FormattableEntity, error) {
	return writer.FormattableEntity{}, fmt.Errorf("importing a single rETL connection is not supported")
}

// The remaining methods are completed in DEX-827 (remote discovery and import).
// Until then they answer with valid empty results so the handler cannot panic
// or invent state.

func (h *Handler) List(_ context.Context, _ *bool) ([]resources.ResourceData, error) {
	return []resources.ResourceData{}, nil
}

func (h *Handler) LoadResourcesFromRemote(_ context.Context) (*resources.RemoteResources, error) {
	return resources.NewRemoteResources(), nil
}

func (h *Handler) MapRemoteToState(_ *resources.RemoteResources) (*state.State, error) {
	return state.EmptyState(), nil
}

func (h *Handler) LoadImportable(_ context.Context, _ namer.Namer) (*resources.RemoteResources, error) {
	return resources.NewRemoteResources(), nil
}

func (h *Handler) FormatForExport(_ *resources.RemoteResources, _ namer.Namer, _ resolver.ReferenceResolver) ([]writer.FormattableEntity, []importmanifest.ImportEntry, error) {
	return nil, nil, nil
}

func (h *Handler) Import(_ context.Context, _ string, _ resources.ResourceData, _ string) (*resources.ResourceData, error) {
	return nil, fmt.Errorf("importing rETL connections is not supported yet")
}
