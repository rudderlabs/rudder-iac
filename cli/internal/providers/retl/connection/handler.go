package connection

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"time"

	"github.com/go-viper/mapstructure/v2"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

// Handler manages rETL connections. It mirrors the event stream connection
// handler — one graph resource per spec entry, endpoint changes handled as a
// replacement — and adds what the rETL API forces on top: a canonical config
// carried through the graph, and a create that has to claim its identity in a
// second call.
type Handler struct {
	client     retlClient.RETLStore
	registry   *definitions.Registry
	resources  map[string]*connectionResource
	importFile string
}

// NewHandler builds the handler. The destination registry is what separates a
// destination whose rETL flow the spec can express from one it cannot; the
// remote discovery and import paths DEX-827 adds read it to decide what is
// eligible for adoption.
func NewHandler(client retlClient.RETLStore, importDir string, registry *definitions.Registry) *Handler {
	return &Handler{
		client:     client,
		registry:   registry,
		resources:  make(map[string]*connectionResource),
		importFile: filepath.Join(importDir, ImportPath),
	}
}

// ParseSpec collects one URN per connection entry — the spec body is a list,
// unlike the single-resource SQL model spec. No LegacyResourceType: the kind
// has never had a rudder/0.1 form.
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

// LoadSpec decodes the spec strictly at every level: a misspelt key anywhere
// under config — destination_config included, which has no spec equivalent —
// is an error rather than a silently dropped setting.
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

// loadConnection builds the graph-side resource for one entry. Required-field
// and cross-field checks belong to the validation rules driven by the spec's
// validate tags; only reference parsing and the config conversion fail here.
func loadConnection(c ConnectionSpec) (*connectionResource, error) {
	sourceRef, err := parseSourceRef(c.Source)
	if err != nil {
		return nil, fmt.Errorf("connection %q: parsing source reference: %w", c.LocalID, err)
	}
	destinationRef, err := parseDestinationRef(c.Destination)
	if err != nil {
		return nil, fmt.Errorf("connection %q: parsing destination reference: %w", c.LocalID, err)
	}
	config, err := configToData(c.Config)
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

// parseDestinationRef parses a scalar "#destination:<id>" reference into a
// PropertyRef whose Resolve reads DestinationState.ID. Destinations are one
// provider for both connection families, so this is the event stream ref
// verbatim.
func parseDestinationRef(ref string) (*resources.PropertyRef, error) {
	kind, id, ok := refID(ref)
	if !ok || kind != destination.DestinationSpecKind {
		return nil, fmt.Errorf("invalid reference %q: expected format #%s:<id>", ref, destination.DestinationSpecKind)
	}

	propertyRef := handler.CreatePropertyRef(
		resources.URN(id, destination.DestinationResourceType),
		func(state *destination.DestinationState) (string, error) {
			if state.ID == "" {
				return "", fmt.Errorf("destination state has empty ID")
			}
			return state.ID, nil
		},
	)
	// Stamp the "id" property so the differ's comparePropertyRefs sees a
	// stable shape on both the spec and the state side.
	propertyRef.Property = "id"
	return propertyRef, nil
}

// GetResources emits each connection as a plain data-map resource; the graph
// derives its dependency edges from the two endpoint refs in the map, and the
// syncer dereferences them to remote ids before the lifecycle calls.
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

// addImportMetadata copies the spec's inline metadata.import entries into this
// connection's ImportMetadata map.
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

// applyImportManifest writes manifest entries into this connection's
// ImportMetadata map. Shared by the inline metadata.import path and the central
// import-manifest broadcast.
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
// connection. Each connection reads only its own URN back at graph time, so
// replicating the full manifest is safe. Nil-safe.
func (h *Handler) LoadImportMetadata(m *specs.WorkspacesImportMetadata) error {
	if m == nil {
		return nil
	}
	for _, c := range h.resources {
		c.applyImportManifest(m)
	}
	return nil
}

// Create creates the connection remotely and then claims it. The two steps are
// separate calls because the backend's per-flow allow-list rejects externalId
// in the create body, which leaves a window where the row exists unclaimed —
// see claimIdentity for what happens in it.
func (h *Handler) Create(ctx context.Context, id string, data resources.ResourceData) (*resources.ResourceData, error) {
	request, err := toCreateRequest(data)
	if err != nil {
		return nil, fmt.Errorf("building create request for rETL connection %q: %w", id, err)
	}

	created, err := h.client.CreateConnection(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("creating rETL connection %q: %w", id, err)
	}

	output, err := h.claimIdentity(ctx, id, created)
	if err != nil {
		return nil, fmt.Errorf("claiming external id for rETL connection %q: %w", id, err)
	}
	return output, nil
}

// recoveryTimeout bounds the read-back and the compensating delete that follow
// a failed claim. They run detached from the caller's context, so this is the
// only thing left to stop them hanging.
const recoveryTimeout = 30 * time.Second

// claimIdentity attaches the CLI identity to the row the POST just created.
// When the claim call fails — including a timeout that may still have been
// applied server-side — the row is read back rather than assumed lost, because
// blindly creating again on the next apply would duplicate it.
//
// The read-back decides what happens next:
//   - the desired external id is already there: the claim landed after all.
//   - no external id at all: the row is provably this invocation's orphan, so
//     it is deleted and the next apply can start clean.
//   - someone else's external id, or a read-back that failed: nothing is
//     touched, and the caller is told which remote row to reconcile.
//
// Only a row this call created is ever deleted here.
func (h *Handler) claimIdentity(ctx context.Context, id string, created *retlClient.RETLConnection) (*resources.ResourceData, error) {
	claimErr := h.client.SetConnectionExternalId(ctx, &retlClient.SetRETLConnectionExternalIDRequest{
		ID:         created.ID,
		ExternalID: id,
	})
	if claimErr == nil {
		return toOutput(created), nil
	}

	// The parent context may be the very thing that failed the claim, and a
	// dead one would fail the read-back instantly — turning every timeout into
	// the inconclusive branch and leaving the orphan behind. Recovery gets its
	// own bounded context so it can still reach the server.
	recoveryCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), recoveryTimeout)
	defer cancel()

	remote, err := h.client.GetConnection(recoveryCtx, created.ID)
	if err != nil {
		return nil, fmt.Errorf("%w; reading connection %s back failed too (%v), so it was left in place: import or delete it before the next apply", claimErr, created.ID, err)
	}
	if remote.ExternalID == id {
		return toOutput(remote), nil
	}
	if remote.ExternalID != "" {
		return nil, fmt.Errorf("%w; connection %s already carries external id %q, so it was left in place: reconcile or import it before the next apply", claimErr, created.ID, remote.ExternalID)
	}

	if err := h.client.DeleteConnection(recoveryCtx, created.ID); err != nil {
		return nil, fmt.Errorf("%w; connection %s carries no external id but deleting it failed too (%v): it is orphaned and has to be deleted or imported manually", claimErr, created.ID, err)
	}
	return nil, fmt.Errorf("%w; connection %s carried no external id and was deleted, so the next apply can create it again", claimErr, created.ID)
}

// Update changes the mutable fields in one PUT. A change to an endpoint or to
// any field the API refuses on update is a replacement — delete then create —
// because the backend allows one connection per source–destination pair.
// Recreating the same pair revives the soft-deleted row and its remote id,
// while a new pair returns a new one, so the create response is what lands back
// in state either way.
//
// Everything that can fail on conversion is checked up front: the delete is
// destructive, and a config that does not decode must not cost a connection.
func (h *Handler) Update(ctx context.Context, id string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	remoteID, err := stateID(state, IDKey)
	if err != nil {
		return nil, fmt.Errorf("connection %q: %w", id, err)
	}

	sourceID, err := endpointID(data, SourceKey)
	if err != nil {
		return nil, fmt.Errorf("connection %q: %w", id, err)
	}
	destinationID, err := endpointID(data, DestinationKey)
	if err != nil {
		return nil, fmt.Errorf("connection %q: %w", id, err)
	}
	// The stored endpoint ids are half of the comparison that decides whether
	// this is a replacement. State that cannot answer reads as "the endpoint
	// changed", which would delete a live connection, so it stops here instead.
	storedSourceID, err := stateID(state, SourceIDKey)
	if err != nil {
		return nil, fmt.Errorf("connection %q: %w", id, err)
	}
	storedDestinationID, err := stateID(state, DestinationIDKey)
	if err != nil {
		return nil, fmt.Errorf("connection %q: %w", id, err)
	}
	enabled, err := enabledFlag(data)
	if err != nil {
		return nil, fmt.Errorf("connection %q: %w", id, err)
	}
	desired, err := configFromData(data[ConfigKey])
	if err != nil {
		return nil, fmt.Errorf("connection %q: %w", id, err)
	}
	stored, err := configFromData(state[ConfigKey])
	if err != nil {
		return nil, fmt.Errorf("connection %q: reading stored connection config: %w", id, err)
	}
	desired, stored = canonicalConfig(desired), canonicalConfig(stored)

	if field := immutableChange(sourceID, storedSourceID, destinationID, storedDestinationID, desired, stored); field != "" {
		if err := h.Delete(ctx, id, state); err != nil {
			return nil, err
		}
		// The delete already happened, so a failure here leaves the connection
		// gone while state still carries its remote id; say so rather than
		// reporting a bare create failure.
		created, err := h.Create(ctx, id, data)
		if err != nil {
			return nil, fmt.Errorf("recreating rETL connection %q after %s changed (the previous connection was deleted): %w", id, field, err)
		}
		return created, nil
	}

	if enabled == state[EnabledKey] && reflect.DeepEqual(desired, stored) {
		return toOutput(&retlClient.RETLConnection{ID: remoteID, SourceID: sourceID, DestinationID: destinationID}), nil
	}

	request, err := toUpdateRequest(data, state)
	if err != nil {
		return nil, fmt.Errorf("connection %q: %w", id, err)
	}
	updated, err := h.client.UpdateConnection(ctx, remoteID, request)
	if err != nil {
		return nil, fmt.Errorf("updating rETL connection %q: %w", id, err)
	}
	return toOutput(updated), nil
}

// immutableChange names the field that forces a replacement, or "" when the
// change is one a PUT can carry. The endpoints and the four config fields
// listed here are the ones the API refuses on update, even unchanged.
//
// The two pointer fields go through DeepEqual rather than a dereference: an
// omitted object or event and an explicitly empty one are different configs,
// and the config map round-trip keeps them apart.
func immutableChange(sourceID, storedSourceID, destinationID, storedDestinationID string, desired, stored ConfigSpec) string {
	switch {
	case sourceID != storedSourceID:
		return "source"
	case destinationID != storedDestinationID:
		return "destination"
	case desired.SyncBehaviour != stored.SyncBehaviour:
		return "sync_behaviour"
	case desired.CursorColumn != stored.CursorColumn:
		return "cursor_column"
	case !reflect.DeepEqual(desired.Object, stored.Object):
		return "object"
	case !reflect.DeepEqual(desired.Event, stored.Event):
		return "event"
	}
	return ""
}

// stateID reads one of the identifiers the lifecycle stored. A missing or
// malformed value is an error rather than a usable zero: Update compares these
// to decide whether a change is a destructive replacement.
func stateID(state resources.ResourceData, key string) (string, error) {
	value, ok := state[key].(string)
	if !ok || value == "" {
		return "", fmt.Errorf("missing %s in state", key)
	}
	return value, nil
}

// Delete removes the remote connection only — the endpoints it links are their
// own resources and are never touched from here.
func (h *Handler) Delete(ctx context.Context, id string, state resources.ResourceData) error {
	remoteID, err := stateID(state, IDKey)
	if err != nil {
		return fmt.Errorf("connection %q: %w", id, err)
	}
	if err := h.client.DeleteConnection(ctx, remoteID); err != nil {
		return fmt.Errorf("deleting rETL connection %q: %w", id, err)
	}
	return nil
}

// Preview is a SQL model capability: a connection has nothing to preview, and
// the rETL preview command only ever targets sources.
func (h *Handler) Preview(_ context.Context, _ string, _ resources.ResourceData, _ int) ([]map[string]any, error) {
	return nil, fmt.Errorf("preview is not supported for rETL connections")
}

// FetchImportData backs `import` for a single named resource, which the rETL
// family exposes for SQL models only. Connections are adopted through the
// workspace-wide import instead.
func (h *Handler) FetchImportData(_ context.Context, _ specs.ImportIds) (writer.FormattableEntity, error) {
	return writer.FormattableEntity{}, fmt.Errorf("importing a single rETL connection is not supported")
}

// The remaining methods are completed in DEX-827, which adds remote discovery
// and import. Until then they answer with valid empty results so a handler
// under construction cannot panic or invent state, and Import refuses outright
// rather than half-adopting a row.

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
