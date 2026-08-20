package connection

import (
	"cmp"
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/api/client"
	esClient "github.com/rudderlabs/rudder-iac/api/client/event-stream"
	sourceClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/lister"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

type Handler struct {
	resources  map[string]*connectionResource
	client     esClient.EventStreamStore
	importFile string
}

func NewHandler(client esClient.EventStreamStore, importDir string) *Handler {
	return &Handler{
		resources:  make(map[string]*connectionResource),
		client:     client,
		importFile: filepath.Join(importDir, ImportPath),
	}
}

// ParseSpec collects one URN per connection entry — the spec body is a list,
// unlike the single-resource event stream source spec.
func (h *Handler) ParseSpec(_ string, s *specs.Spec) (*specs.ParsedSpec, error) {
	raw, ok := s.Spec[ConnectionsKey].([]any)
	if !ok {
		return nil, fmt.Errorf("connections not found in event stream connections spec")
	}
	entries := make([]specs.URNEntry, 0, len(raw))
	for i, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("connection at index %d is not a map", i)
		}
		id, ok := m["id"].(string)
		if !ok {
			return nil, fmt.Errorf("id not found in connection at index %d", i)
		}
		entries = append(entries, specs.URNEntry{
			URN:             resources.URN(id, EventStreamConnectionResourceType),
			JSONPointerPath: fmt.Sprintf("/spec/connections/%d/id", i),
		})
	}
	return &specs.ParsedSpec{URNs: entries}, nil
}

func (h *Handler) LoadSpec(_ string, s *specs.Spec) error {
	spec := &ConnectionsSpec{}

	// Use strict decoding to reject unknown fields.
	decoderConfig := &mapstructure.DecoderConfig{
		ErrorUnused: true,
		Result:      spec,
	}
	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return fmt.Errorf("creating decoder: %w", err)
	}
	if err := decoder.Decode(s.Spec); err != nil {
		return fmt.Errorf("decoding event stream connections spec: %w", err)
	}
	for _, c := range spec.Connections {
		resource, err := h.loadConnection(c)
		if err != nil {
			return err
		}
		if err := resource.addImportMetadata(s); err != nil {
			return fmt.Errorf("loading import metadata: %w", err)
		}
		// When we are at this point, we expect the spec along with the
		// localID to be valid and unique (see project/duplicate-urn rule)
		h.resources[resource.LocalID] = resource
	}
	return nil
}

// loadConnection builds the graph-side resource for one entry. Required-field
// checks are owned by the validation rules driven by the spec's validate tags
// (DEX-652); only reference parsing can fail here.
func (h *Handler) loadConnection(c ConnectionSpec) (*connectionResource, error) {
	sourceRef, err := parseSourceRef(c.Source)
	if err != nil {
		return nil, fmt.Errorf("connection %q: parsing source reference: %w", c.LocalID, err)
	}
	destinationRef, err := parseDestinationRef(c.Destination)
	if err != nil {
		return nil, fmt.Errorf("connection %q: parsing destination reference: %w", c.LocalID, err)
	}

	// Default enabled to true when not specified in the spec
	enabled := true
	if c.Enabled != nil {
		enabled = *c.Enabled
	}

	return &connectionResource{
		LocalID:        c.LocalID,
		Source:         sourceRef,
		Destination:    destinationRef,
		Enabled:        enabled,
		ImportMetadata: make(map[string]*WorkspaceRemoteIDMapping),
	}, nil
}

// MigrateSpec is an identity migration: the kind is v1-only, so there are no
// legacy formats to rewrite.
func (h *Handler) MigrateSpec(s *specs.Spec) (*specs.Spec, error) {
	return s, nil
}

// GetResources emits each connection as a plain data-map resource; the graph
// derives dependency edges from the two endpoint refs in the map, and the
// syncer dereferences them to remote ids before the lifecycle calls.
func (h *Handler) GetResources() ([]*resources.Resource, error) {
	result := make([]*resources.Resource, 0, len(h.resources))
	for _, c := range h.resources {
		data := resources.ResourceData{
			SourceKey:      c.Source,
			DestinationKey: c.Destination,
			EnabledKey:     c.Enabled,
		}
		opts := []resources.ResourceOpts{
			resources.WithResourceFileMetadata(fmt.Sprintf("#%s:%s", EventStreamConnectionResourceKind, c.LocalID)),
		}
		urn := resources.URN(c.LocalID, EventStreamConnectionResourceType)
		if importMetadata, ok := c.ImportMetadata[urn]; ok {
			opts = []resources.ResourceOpts{
				resources.WithResourceImportMetadata(importMetadata.RemoteId, importMetadata.WorkspaceId),
			}
		}
		r := resources.NewResource(
			c.LocalID,
			EventStreamConnectionResourceType,
			data,
			[]string{},
			opts...,
		)
		result = append(result, r)
	}
	return result, nil
}

// parseSourceRef parses a scalar "#event-stream-source:<id>" reference.
func parseSourceRef(ref string) (*resources.PropertyRef, error) {
	id, err := refID(ref, source.ResourceKind)
	if err != nil {
		return nil, err
	}
	return &resources.PropertyRef{
		URN:      resources.URN(id, source.ResourceType),
		Property: "id",
	}, nil
}

// parseDestinationRef parses a scalar "#destination:<id>" reference into a
// PropertyRef whose Resolve function reads DestinationState.ID, mirroring the
// destination provider's transformation ref.
func parseDestinationRef(ref string) (*resources.PropertyRef, error) {
	id, err := refID(ref, destination.DestinationSpecKind)
	if err != nil {
		return nil, err
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
	// stable shape on both the spec and state sides.
	propertyRef.Property = "id"
	return propertyRef, nil
}

// refID extracts <id> from a scalar "#<kind>:<id>" reference.
func refID(ref string, kind string) (string, error) {
	trimmed := strings.TrimSpace(ref)
	if !strings.HasPrefix(trimmed, "#") {
		return "", fmt.Errorf("invalid reference %q: expected format #%s:<id>", ref, kind)
	}
	parts := strings.SplitN(strings.TrimPrefix(trimmed, "#"), ":", 2)
	if len(parts) != 2 || parts[0] != kind || parts[1] == "" {
		return "", fmt.Errorf("invalid reference %q: expected format #%s:<id>", ref, kind)
	}
	return parts[1], nil
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
	return c.applyImportManifest(metadata.Import)
}

// applyImportManifest writes manifest entries into this connection's
// ImportMetadata map. Shared by the inline metadata.import path
// (addImportMetadata) and the central import-manifest broadcast
// (Handler.LoadImportMetadata).
func (c *connectionResource) applyImportManifest(m *specs.WorkspacesImportMetadata) error {
	for _, workspace := range m.Workspaces {
		for _, resource := range workspace.Resources {
			// Support both URN field (new) and LocalID field (legacy)
			urn := resource.URN
			if urn == "" {
				urn = resources.URN(resource.LocalID, EventStreamConnectionResourceType)
			}
			c.ImportMetadata[urn] = &WorkspaceRemoteIDMapping{
				WorkspaceId: workspace.WorkspaceID,
				RemoteId:    resource.RemoteID,
			}
		}
	}
	return nil
}

// LoadImportMetadata replicates the aggregated manifest into every loaded
// connection. Each connection reads only its own URN from ImportMetadata at
// graph time (see GetResources), so replicating the full manifest into every
// connection is safe. Nil-safe.
func (h *Handler) LoadImportMetadata(m *specs.WorkspacesImportMetadata) error {
	if m == nil {
		return nil
	}
	for _, c := range h.resources {
		if err := c.applyImportManifest(m); err != nil {
			return err
		}
	}
	return nil
}

// sourcesByID indexes the workspace's event stream sources by remote id — the
// membership set that separates event stream connections from rETL rows.
func (h *Handler) sourcesByID(ctx context.Context) (map[string]sourceClient.EventStreamSource, error) {
	sources, err := h.client.GetSources(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting event stream sources: %w", err)
	}
	byID := make(map[string]sourceClient.EventStreamSource, len(sources))
	for _, s := range sources {
		byID[s.ID] = s
	}
	return byID, nil
}

// eventStreamConnections pages through the connections list with the given
// options. The generic connections list also returns rETL rows; only
// connections whose source is an event stream source are kept. The sources
// consulted for the filter are returned too, keyed by remote id, so callers
// can read names and workspace ids without refetching.
func (h *Handler) eventStreamConnections(ctx context.Context, opts ...client.ListConnectionsOption) ([]client.Connection, map[string]sourceClient.EventStreamSource, error) {
	sourcesByID, err := h.sourcesByID(ctx)
	if err != nil {
		return nil, nil, err
	}

	var conns []client.Connection
	page, err := h.client.ListConnections(ctx, opts...)
	for page != nil && err == nil {
		for _, conn := range page.Connections {
			if _, ok := sourcesByID[conn.SourceID]; !ok {
				continue
			}
			conns = append(conns, conn)
		}
		page, err = h.client.NextConnections(ctx, page.Paging)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("listing event stream connections: %w", err)
	}
	return conns, sourcesByID, nil
}

// LoadResourcesFromRemote lists the remote connections that carry an
// externalId — the CLI-managed ones.
func (h *Handler) LoadResourcesFromRemote(ctx context.Context) (*resources.RemoteResources, error) {
	collection := resources.NewRemoteResources()

	conns, _, err := h.eventStreamConnections(ctx, client.WithConnectionsHasExternalID(true))
	if err != nil {
		return nil, err
	}

	resourceMap := make(map[string]*resources.RemoteResource, len(conns))
	for _, conn := range conns {
		resourceMap[conn.ID] = &resources.RemoteResource{
			ID:         conn.ID,
			ExternalID: conn.ExternalID,
			Data:       conn,
		}
	}

	collection.Set(EventStreamConnectionResourceType, resourceMap)
	return collection, nil
}

// MapRemoteToState turns CLI-managed remote connections into state keyed on
// externalId (rETL rows were already filtered out in LoadResourcesFromRemote).
// Endpoints resolve through the merged cross-provider collection into
// PropertyRefs shaped exactly like the spec side, so the differ compares
// cleanly. A row whose endpoint cannot be resolved to a CLI-managed resource
// cannot be expressed as spec refs and is skipped (mirroring the source
// handler's leniency on tracking-plan lookups).
func (h *Handler) MapRemoteToState(collection *resources.RemoteResources) (*state.State, error) {
	s := state.EmptyState()
	for _, remote := range collection.GetAll(EventStreamConnectionResourceType) {
		conn, ok := remote.Data.(client.Connection)
		if !ok {
			return nil, fmt.Errorf("unable to cast resource to event stream connection")
		}
		sourceURN, err := collection.GetURNByID(source.ResourceType, conn.SourceID)
		if err != nil {
			continue
		}
		destinationURN, err := collection.GetURNByID(destination.DestinationResourceType, conn.DestinationID)
		if err != nil {
			continue
		}
		s.AddResource(mapRemoteToState(&conn, sourceURN, destinationURN))
	}
	return s, nil
}

// mapRemoteToState builds one connection's resource state: the spec-shaped
// endpoint refs plus enabled as Input, the remote identifiers as Output.
func mapRemoteToState(conn *client.Connection, sourceURN, destinationURN string) *state.ResourceState {
	return &state.ResourceState{
		ID:   conn.ExternalID,
		Type: EventStreamConnectionResourceType,
		Input: map[string]any{
			SourceKey:      &resources.PropertyRef{URN: sourceURN, Property: "id"},
			DestinationKey: &resources.PropertyRef{URN: destinationURN, Property: "id"},
			EnabledKey:     conn.IsEnabled,
		},
		Output: *toResourceData(conn),
	}
}

// LoadImportable lists the remote event stream connections not yet managed by
// the CLI (no externalId) and assigns each an identity derived from its
// endpoints' names, e.g. "android-source-to-s3".
func (h *Handler) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.RemoteResources, error) {
	collection := resources.NewRemoteResources()

	conns, sourcesByID, err := h.eventStreamConnections(ctx, client.WithConnectionsHasExternalID(false))
	if err != nil {
		return nil, err
	}

	resourceMap := make(map[string]*resources.RemoteResource, len(conns))
	collection.Set(EventStreamConnectionResourceType, resourceMap)
	if len(conns) == 0 {
		return collection, nil
	}

	destinations, err := h.client.GetDestinations(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting destinations: %w", err)
	}
	destinationsByID := make(map[string]client.Destination, len(destinations))
	for _, d := range destinations {
		destinationsByID[d.ID] = d
	}

	for _, conn := range conns {
		// The source is always present: eventStreamConnections only returns
		// connections whose source is in the map.
		src := sourcesByID[conn.SourceID]
		dst, ok := destinationsByID[conn.DestinationID]
		destinationName := dst.Name
		if !ok {
			// The name only seeds the identity; a destination missing from
			// the list falls back to its remote id.
			destinationName = conn.DestinationID
		}
		externalID, err := idNamer.Name(namer.ScopeName{
			Name:  fmt.Sprintf("%s-to-%s", src.Name, destinationName),
			Scope: EventStreamConnectionResourceType,
		})
		if err != nil {
			return nil, fmt.Errorf("generating externalID for connection %s: %w", conn.ID, err)
		}
		resourceMap[conn.ID] = &resources.RemoteResource{
			ID:         conn.ID,
			ExternalID: externalID,
			Reference:  fmt.Sprintf("#%s:%s", EventStreamConnectionResourceKind, externalID),
			Data: &RemoteConnection{
				Connection:            conn,
				WorkspaceID:           src.WorkspaceID,
				SourceExternalID:      src.ExternalID,
				DestinationExternalID: dst.ExternalID,
			},
		}
	}

	return collection, nil
}

// FormatForExport writes the importable connections as one spec of the
// event-stream-connections kind per run.
func (h *Handler) FormatForExport(
	collection *resources.RemoteResources,
	_ namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, []importmanifest.ImportEntry, error) {
	remotesByID := collection.GetAll(EventStreamConnectionResourceType)
	if len(remotesByID) == 0 {
		return nil, nil, nil
	}

	// One spec file holds every connection: order by assigned id so the
	// emitted list is stable across runs.
	remotes := make([]*resources.RemoteResource, 0, len(remotesByID))
	for _, remote := range remotesByID {
		remotes = append(remotes, remote)
	}
	slices.SortFunc(remotes, func(a, b *resources.RemoteResource) int {
		return cmp.Compare(a.ExternalID, b.ExternalID)
	})

	workspaceMetadata := specs.WorkspaceImportMetadata{
		Resources: make([]specs.ImportIds, 0, len(remotes)),
	}
	var entries []importmanifest.ImportEntry
	items := make([]map[string]any, 0, len(remotes))
	for _, remote := range remotes {
		data, ok := remote.Data.(*RemoteConnection)
		if !ok {
			return nil, nil, fmt.Errorf("unable to cast remote resource to event stream connection")
		}
		if workspaceMetadata.WorkspaceID != "" && workspaceMetadata.WorkspaceID != data.WorkspaceID {
			return nil, nil, fmt.Errorf("cannot export resources from multiple workspaces into a single spec file")
		}
		workspaceMetadata.WorkspaceID = data.WorkspaceID

		urn := resources.URN(remote.ExternalID, EventStreamConnectionResourceType)
		entry := importmanifest.ImportEntry{
			WorkspaceID: data.WorkspaceID,
			URN:         urn,
			RemoteID:    remote.ID,
		}

		// Matched connections (import --merge) adopt an existing local spec:
		// manifest entry only — no spec entry is written for them.
		if remote.MatchedWith != nil {
			entries = append(entries, entry)
			continue
		}

		item, err := toImportItem(remote.ExternalID, data, inputResolver)
		if err != nil {
			// An endpoint resolving to neither an importable nor a CLI-managed
			// resource (e.g. a destination type the CLI has not onboarded)
			// cannot be expressed as spec refs; the connection is left out
			// entirely, mirroring MapRemoteToState's leniency.
			continue
		}
		entries = append(entries, entry)
		workspaceMetadata.Resources = append(workspaceMetadata.Resources, specs.ImportIds{
			URN:      urn,
			RemoteID: remote.ID,
		})
		items = append(items, item)
	}

	if len(items) == 0 {
		return nil, entries, nil
	}

	spec, err := specs.ToImportSpec(
		EventStreamConnectionResourceKind,
		MetadataName,
		workspaceMetadata,
		map[string]any{ConnectionsKey: items},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("creating spec: %w", err)
	}

	return []writer.FormattableEntity{{
		Content:      spec,
		RelativePath: h.importFile,
	}}, entries, nil
}

// toImportItem builds one connection's spec entry, resolving both endpoints
// through the merged collection — imported in the same run or already
// CLI-managed.
func toImportItem(externalID string, data *RemoteConnection, inputResolver resolver.ReferenceResolver) (map[string]any, error) {
	sourceRef, err := endpointRef(inputResolver, source.ResourceType, source.ResourceKind, data.SourceID, data.SourceExternalID)
	if err != nil {
		return nil, fmt.Errorf("resolving source reference: %w", err)
	}
	destinationRef, err := endpointRef(inputResolver, destination.DestinationResourceType, destination.DestinationSpecKind, data.DestinationID, data.DestinationExternalID)
	if err != nil {
		return nil, fmt.Errorf("resolving destination reference: %w", err)
	}
	return map[string]any{
		IDKey:          externalID,
		SourceKey:      sourceRef,
		DestinationKey: destinationRef,
		EnabledKey:     data.IsEnabled,
	}, nil
}

// endpointRef resolves an endpoint's remote id into a spec reference: through
// the import resolver — the endpoint imported in the same run, or managed with
// file metadata on its graph entry — or, for an already-managed endpoint the
// resolver cannot serve (BaseHandler-backed destinations carry no file
// metadata), built from its externalId, which is the endpoint's local resource
// id. The ref shape mirrors what parseSourceRef/parseDestinationRef accept.
func endpointRef(inputResolver resolver.ReferenceResolver, resourceType string, kind string, remoteID string, externalID string) (string, error) {
	ref, err := inputResolver.ResolveToReference(resourceType, remoteID)
	if err == nil {
		return ref, nil
	}
	if externalID == "" {
		return "", err
	}
	return fmt.Sprintf("#%s:%s", kind, externalID), nil
}

// Create creates the connection remotely. By the time it runs the syncer has
// dereferenced the spec refs, so data carries the endpoints' remote ids. The
// CLI identity travels as externalId in the create body; the backend attaches
// it to the row it returns — which may be a revived soft-deleted row for the
// same source–destination pair, so the output always records the returned id.
func (h *Handler) Create(ctx context.Context, id string, data resources.ResourceData) (*resources.ResourceData, error) {
	created, err := h.client.CreateConnection(ctx, &client.Connection{
		ExternalID:    id,
		SourceID:      data[SourceKey].(string),
		DestinationID: data[DestinationKey].(string),
		IsEnabled:     data[EnabledKey].(bool),
	})
	if err != nil {
		return nil, fmt.Errorf("creating event stream connection %q: %w", id, err)
	}
	return toResourceData(created), nil
}

func toResourceData(conn *client.Connection) *resources.ResourceData {
	return &resources.ResourceData{
		IDKey:            conn.ID,
		SourceIDKey:      conn.SourceID,
		DestinationIDKey: conn.DestinationID,
	}
}

// Update changes enabled in place. An endpoint change is a replacement —
// delete then create — because the backend allows only one connection per
// source–destination pair. Recreating a pair that existed before revives the
// soft-deleted row (same remote id), so the create response, not a
// presumed-fresh id, is what lands back in state.
func (h *Handler) Update(ctx context.Context, id string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	remoteID, ok := state[IDKey].(string)
	if !ok || remoteID == "" {
		return nil, fmt.Errorf("connection %q: missing id in state", id)
	}

	if data[SourceKey] != state[SourceIDKey] || data[DestinationKey] != state[DestinationIDKey] {
		if err := h.Delete(ctx, id, state); err != nil {
			return nil, err
		}
		return h.Create(ctx, id, data)
	}

	desired := &client.Connection{
		ID:            remoteID,
		SourceID:      data[SourceKey].(string),
		DestinationID: data[DestinationKey].(string),
		IsEnabled:     data[EnabledKey].(bool),
	}
	if state[EnabledKey] == data[EnabledKey] {
		return toResourceData(desired), nil
	}

	updated, err := h.client.UpdateConnection(ctx, desired)
	if err != nil {
		return nil, fmt.Errorf("updating event stream connection %q: %w", id, err)
	}
	return toResourceData(updated), nil
}

// Delete removes the remote connection only — the endpoints it links are
// their own resources and are never touched from here.
func (h *Handler) Delete(ctx context.Context, id string, state resources.ResourceData) error {
	remoteID, ok := state[IDKey].(string)
	if !ok || remoteID == "" {
		return fmt.Errorf("connection %q: missing id in state", id)
	}
	if err := h.client.DeleteConnection(ctx, remoteID); err != nil {
		return fmt.Errorf("deleting event stream connection %q: %w", id, err)
	}
	return nil
}

// List reports every event stream connection in the workspace — managed or
// not; rows carrying an externalId are the CLI-managed ones. No workspace
// list command is wired to it yet; it completes the handler surface the
// provider dispatches to.
func (h *Handler) List(ctx context.Context, _ lister.Filters) ([]resources.ResourceData, error) {
	conns, _, err := h.eventStreamConnections(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]resources.ResourceData, 0, len(conns))
	for _, conn := range conns {
		resourceData := resources.ResourceData{
			IDKey:            conn.ID,
			SourceIDKey:      conn.SourceID,
			DestinationIDKey: conn.DestinationID,
			EnabledKey:       conn.IsEnabled,
		}
		if conn.ExternalID != "" {
			resourceData[ExternalIDKey] = conn.ExternalID
		}
		result = append(result, resourceData)
	}

	return result, nil
}

// Import adopts an existing remote connection into CLI management: it pushes
// the spec's enabled flag and endpoints via Update (same reconciliation path
// as a regular apply), then sets the external ID last so a failed Update
// never leaves a partially-adopted resource behind.
func (h *Handler) Import(ctx context.Context, id string, data resources.ResourceData, remoteId string) (*resources.ResourceData, error) {
	remote, err := h.client.GetConnection(ctx, remoteId)
	if err != nil {
		return nil, fmt.Errorf("getting event stream connection during import: %w", err)
	}

	// The generic /v2/connections API serves rETL rows too; refuse to adopt a
	// row whose source is not an event stream source (e.g. a stale or
	// hand-edited import manifest) before Update mutates it.
	sources, err := h.sourcesByID(ctx)
	if err != nil {
		return nil, err
	}
	if _, ok := sources[remote.SourceID]; !ok {
		return nil, fmt.Errorf("connection %q: source %q is not an event stream source", remoteId, remote.SourceID)
	}

	existingState := resources.ResourceData{
		IDKey:            remote.ID,
		SourceIDKey:      remote.SourceID,
		DestinationIDKey: remote.DestinationID,
		EnabledKey:       remote.IsEnabled,
	}

	result, err := h.Update(ctx, id, data, existingState)
	if err != nil {
		return nil, fmt.Errorf("updating event stream connection during import: %w", err)
	}

	// An endpoint change during import is a replacement: Update deleted the
	// remote row and created a new one whose create body already carried the
	// externalId, so there is nothing left to stamp it on.
	if (*result)[IDKey] == remoteId {
		if err := h.client.SetConnectionExternalID(ctx, remoteId, id); err != nil {
			return nil, fmt.Errorf("setting external ID for event stream connection during import: %w", err)
		}
	}
	return result, nil
}
