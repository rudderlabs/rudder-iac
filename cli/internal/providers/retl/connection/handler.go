package connection

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"sync"

	"github.com/go-viper/mapstructure/v2"
	apiClient "github.com/rudderlabs/rudder-iac/api/client"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	esConnection "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/connection"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/samber/lo"
)

var log = logger.New("retl-connection")

const listPageSize = 100

// Handler manages rETL connections the way the event stream connection handler
// does: one graph resource per spec entry, an endpoint change handled as a
// delete-then-create replacement, and an immutable config change rejected with
// the remedy rather than silently not applied.
type Handler struct {
	client     retlClient.RETLStore
	registry   *definitions.Registry
	resources  map[string]*connectionResource
	importFile string

	// The endpoint catalogs, fetched once and shared by every operation that
	// judges a remote row. The syncer runs one Import per connection, possibly
	// concurrently, so the fetch is guarded rather than repeated per row.
	endpointsMu      sync.Mutex
	sourcesByID      map[string]retlClient.RETLSource
	destinationsByID map[string]apiClient.Destination

	// enabledSourceTypes is the set of source kinds this run registered; nil
	// means all of SourceKinds. Set once by the provider, before any operation.
	enabledSourceTypes map[string]bool
}

// EnableSourceKinds narrows the source kinds remote rows may use to those whose
// resource type is registered. The provider calls it after every option, so a
// connection on a kind whose flag is off is skipped with that flag named,
// rather than exported as a reference this CLI cannot load.
func (h *Handler) EnableSourceKinds(resourceTypes ...string) {
	h.enabledSourceTypes = make(map[string]bool, len(resourceTypes))
	for _, rt := range resourceTypes {
		h.enabledSourceTypes[rt] = true
	}
}

func (h *Handler) sourceKindEnabled(sk SourceKind) bool {
	return h.enabledSourceTypes == nil || h.enabledSourceTypes[sk.ResourceType]
}

// NewHandler takes the destination registry because a remote row names its
// destination by upstream API type only; whether the CLI can express the
// connection at all depends on the definition registered for that type.
func NewHandler(client retlClient.RETLStore, importDir string, registry *definitions.Registry) *Handler {
	return &Handler{
		client:     client,
		registry:   registry,
		resources:  make(map[string]*connectionResource),
		importFile: filepath.Join(importDir, ImportPath),
	}
}

func (h *Handler) SpecSchema() any {
	return ConnectionsSpec{}
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

	if err := h.assertCreatable(ctx, request); err != nil {
		return nil, fmt.Errorf("vetting rETL connection %q: %w", id, err)
	}

	created, err := h.client.CreateConnection(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("creating rETL connection %q: %w", id, err)
	}
	// The one skip reason a create cannot be vetted for up front: the server
	// may store a config the spec cannot express (destination-specific settings
	// filled in on its side). The create response is flattened by the same
	// mapper as GET, so this is exactly what MapRemoteToState will see — and a
	// row it skips would be re-created on every apply. Undo it instead.
	if _, err := configFromRemote(created); err != nil {
		if delErr := h.client.DeleteConnection(ctx, created.ID); delErr != nil {
			return nil, fmt.Errorf("rETL connection %q was created as %q but cannot be read back (%w), and deleting it failed: %w", id, created.ID, err, delErr)
		}
		return nil, fmt.Errorf("rETL connection %q was created but cannot be read back, so it was deleted again: %w", id, err)
	}
	return toResourceData(created), nil
}

// assertCreatable refuses a create the read path would not be able to map back.
func (h *Handler) assertCreatable(ctx context.Context, request *retlClient.CreateRETLConnectionRequest) error {
	dst, err := h.destination(ctx, request.DestinationID)
	if err != nil {
		return err
	}
	return h.destinationUsable(dst, request.Object)
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
	sourceID, destinationID, replace, err := replacementNeeded(data, state)
	if err != nil {
		return nil, fmt.Errorf("connection %q: %w", id, err)
	}

	if replace {
		// Vet the whole create before the delete: anything the create would
		// refuse must not cost the live connection.
		request, err := toCreateRequest(data)
		if err != nil {
			return nil, fmt.Errorf("connection %q: %w", id, err)
		}
		if err := h.assertCreatable(ctx, request); err != nil {
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

	if enabled == state[EnabledKey] && reflect.DeepEqual(desired, stored) {
		return toResourceData(&retlClient.RETLConnection{
			ID:            remoteID,
			SourceID:      sourceID,
			DestinationID: destinationID,
		}), nil
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

// replacementNeeded reports whether the desired entry moves the connection to a
// different source–destination pair, the one change a PUT cannot carry and the
// handler applies as a delete followed by a create. Import asks the same
// question, because a replacement leaves it holding a row Create made rather
// than the row it set out to adopt.
//
// The two endpoint ids it validated come back with the verdict so callers can
// use them without asserting on the map a second time.
func replacementNeeded(data, state resources.ResourceData) (sourceID, destinationID string, replace bool, err error) {
	sourceID, err = endpointIDFromData(data, SourceKey)
	if err != nil {
		return "", "", false, err
	}
	destinationID, err = endpointIDFromData(data, DestinationKey)
	if err != nil {
		return "", "", false, err
	}
	return sourceID, destinationID, sourceID != state[SourceIDKey] || destinationID != state[DestinationIDKey], nil
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

// listAll walks the whole connections list. paging.next is an internal
// /apigateway URL the CLI must not call, so it is read as nothing more than
// "there is another page" and the public page number is advanced instead. That
// also means a short — or even empty — page while next is set is not the end of
// the list; only an empty next is.
//
// The walk terminates by construction: the API emits next exactly while
// page*pageSize stays below total, and page strictly increases, so it stops
// after ceil(total/pageSize) requests.
func (h *Handler) listAll(ctx context.Context, hasExternalID *bool) ([]retlClient.RETLConnection, error) {
	var all []retlClient.RETLConnection
	for page := 1; ; page++ {
		result, err := h.client.ListConnections(ctx, &retlClient.ListRETLConnectionsRequest{
			HasExternalID: hasExternalID,
			Page:          page,
			PageSize:      listPageSize,
		})
		if err != nil {
			return nil, fmt.Errorf("listing rETL connections (page %d): %w", page, err)
		}
		if result == nil {
			return nil, fmt.Errorf("listing rETL connections (page %d): empty response", page)
		}
		all = append(all, result.Data...)

		if result.Paging.Next == "" {
			return all, nil
		}
	}
}

// endpoints reads the two catalogs a connection row cannot be judged without:
// GetConnection reports endpoint ids only, so the source kind, the destination
// definition and both names come from the source and destination lists.
//
// The first successful read is kept for the read path. Every lookup there is
// for an endpoint a remote connection row already references, so it existed
// before the run started and a stale snapshot cannot hide it. The write path
// can name a destination this apply just created, so destination refetches on
// a miss and replaces the destination map. A failed read is not kept,
// so a transient error does not poison the handler. A built map is never
// written to, only ever replaced wholesale (see destination), so a reader that
// already holds one keeps a consistent snapshot across the syncer's concurrent
// imports.
func (h *Handler) endpoints(ctx context.Context) (map[string]retlClient.RETLSource, map[string]apiClient.Destination, error) {
	h.endpointsMu.Lock()
	defer h.endpointsMu.Unlock()

	if h.sourcesByID != nil {
		return h.sourcesByID, h.destinationsByID, nil
	}

	sources, err := h.client.ListRetlSources(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("listing rETL sources: %w", err)
	}
	destinations, err := h.client.GetDestinations(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("listing destinations: %w", err)
	}

	sourcesByID := make(map[string]retlClient.RETLSource, len(sources.Data))
	for _, source := range sources.Data {
		sourcesByID[source.ID] = source
	}
	destinationsByID := destinationIndex(destinations)

	h.sourcesByID, h.destinationsByID = sourcesByID, destinationsByID
	return sourcesByID, destinationsByID, nil
}

func destinationIndex(destinations []apiClient.Destination) map[string]apiClient.Destination {
	byID := make(map[string]apiClient.Destination, len(destinations))
	for _, d := range destinations {
		byID[d.ID] = d
	}
	return byID
}

// destination finds a workspace destination for the write path. A miss is
// refetched before it is believed: the cached catalog is a snapshot taken
// before the apply, so a create can legitimately name a destination this same
// apply just built. Still missing after that is fatal, because the API accepts
// such a create and the read path then skips the row forever.
func (h *Handler) destination(ctx context.Context, id string) (apiClient.Destination, error) {
	h.endpointsMu.Lock()
	defer h.endpointsMu.Unlock()

	if dst, ok := h.destinationsByID[id]; ok {
		return dst, nil
	}
	// Refetched under the lock, so concurrent creates naming the same new
	// destination cost one call; coalesce per id if apply ever goes wide.
	destinations, err := h.client.GetDestinations(ctx)
	if err != nil {
		return apiClient.Destination{}, fmt.Errorf("listing destinations: %w", err)
	}
	h.destinationsByID = destinationIndex(destinations)
	dst, ok := h.destinationsByID[id]
	if !ok {
		return apiClient.Destination{}, fmt.Errorf("destination %q was not found in this workspace", id)
	}
	return dst, nil
}

// destinationUsable reports why a destination cannot carry a rETL connection,
// or nil when it can. Read and write both consult it, so a create cannot write
// a row the read path would skip. It covers the destination rules only; the
// source-side reasons remoteConnection skips for stay read-path only.
func (h *Handler) destinationUsable(dst apiClient.Destination, object string) error {
	registered, err := h.registry.GetByAPIType(dst.Type, dst.Version)
	if err != nil {
		// Usually a definition registered only behind a flag this run has off —
		// bingads_offline_conversions and customerio_audience sit behind
		// unverifiedDestinations — so say so rather than surface the lookup.
		return fmt.Errorf("destination type %q version %d is not registered in this CLI; if it is an unverified destination, set RUDDERSTACK_X_UNVERIFIED_DESTINATIONS=true: %w", dst.Type, dst.Version, err)
	}
	if !slices.Contains(registered.SupportedSourceTypes(), common.SourceTypeWarehouse) {
		return fmt.Errorf("destination type %q does not accept warehouse sources", dst.Type)
	}
	if _, err := ClassifyFlow(dst.Type, registered.SupportsVisualMapper(), lo.EmptyableToPtr(object)); err != nil {
		return err
	}
	return nil
}

// remoteConnection checks one remote row against everything the spec contract
// can express and, when it passes, returns it with the endpoint metadata export
// and matching need. The error names the reason, so a bulk caller can log an
// actionable skip and a direct import can refuse before it mutates anything.
func (h *Handler) remoteConnection(
	conn retlClient.RETLConnection,
	sources map[string]retlClient.RETLSource,
	destinations map[string]apiClient.Destination,
) (*RemoteConnection, error) {
	source, ok := sources[conn.SourceID]
	if !ok {
		return nil, fmt.Errorf("connection %q: source %q is not a rETL source in this workspace", conn.ID, conn.SourceID)
	}
	sourceKind, ok := SourceKindBySourceType(source.SourceType)
	if !ok {
		return nil, fmt.Errorf("connection %q: rETL source type %q is not supported", conn.ID, source.SourceType)
	}
	if !h.sourceKindEnabled(sourceKind) {
		return nil, fmt.Errorf("connection %q: its source is a %s, which is not enabled (set %s=true)", conn.ID, sourceKind.Kind, sourceKind.Flag)
	}
	// The workspace is what an import entry is filed under; without one there
	// is no valid import metadata to write for the row.
	if source.WorkspaceID == "" {
		return nil, fmt.Errorf("connection %q: rETL source %q reports no workspace", conn.ID, conn.SourceID)
	}
	dst, ok := destinations[conn.DestinationID]
	if !ok {
		return nil, fmt.Errorf("connection %q: destination %q was not found in this workspace", conn.ID, conn.DestinationID)
	}
	if err := h.destinationUsable(dst, conn.Object); err != nil {
		return nil, fmt.Errorf("connection %q: %w", conn.ID, err)
	}
	// configFromRemote names the connection in its own errors, so wrapping here
	// would only repeat the prefix the other arms add.
	config, err := configFromRemote(&conn)
	if err != nil {
		return nil, err
	}

	return &RemoteConnection{
		RETLConnection:        conn,
		Config:                config,
		WorkspaceID:           source.WorkspaceID,
		SourceKind:            sourceKind,
		SourceName:            source.Name,
		SourceExternalID:      source.ExternalID,
		DestinationName:       dst.Name,
		DestinationExternalID: dst.ExternalID,
	}, nil
}

// logSkip reports a remote row the CLI cannot express. Nothing is changed
// remotely, so the message has to say both that the row survives untouched and
// what would bring it under management.
func logSkip(conn retlClient.RETLConnection, err error) {
	log.Warn("skipping rETL connection the CLI cannot manage: it stays in the workspace unchanged; importing its source and destination, or a CLI version that supports it, may be needed",
		"connection", conn.ID, "externalId", conn.ExternalID, "sourceId", conn.SourceID,
		"destinationId", conn.DestinationID, "reason", err.Error())
}

// eligible lists the connections on the given side of the managed filter and
// keeps the ones the spec contract can express, logging an actionable skip for
// every row it drops. The endpoint catalogs are only read when there is a row
// to judge, so an empty workspace never reaches for them.
func (h *Handler) eligible(ctx context.Context, hasExternalID *bool) ([]*RemoteConnection, error) {
	conns, err := h.listAll(ctx, hasExternalID)
	if err != nil {
		return nil, err
	}
	if len(conns) == 0 {
		return nil, nil
	}

	sources, destinations, err := h.endpoints(ctx)
	if err != nil {
		return nil, err
	}
	remotes := make([]*RemoteConnection, 0, len(conns))
	for _, conn := range conns {
		remote, err := h.remoteConnection(conn, sources, destinations)
		if err != nil {
			logSkip(conn, err)
			continue
		}
		remotes = append(remotes, remote)
	}
	return remotes, nil
}

// LoadResourcesFromRemote lists the CLI-managed connections — those carrying an
// externalId — and keeps the ones the spec contract can express. The payload is
// the same *RemoteConnection LoadImportable stores, so every consumer of this
// resource type reads one type: MapRemoteToState resolves the endpoints out of
// the merged cross-provider collection rather than out of a second fetch, and
// the config it needs is already rebuilt on the value.
func (h *Handler) LoadResourcesFromRemote(ctx context.Context) (*resources.RemoteResources, error) {
	remotes, err := h.eligible(ctx, lo.ToPtr(true))
	if err != nil {
		return nil, err
	}

	resourceMap := make(map[string]*resources.RemoteResource, len(remotes))
	for _, remote := range remotes {
		resourceMap[remote.ID] = &resources.RemoteResource{
			ID:         remote.ID,
			ExternalID: remote.ExternalID,
			Data:       remote,
		}
	}
	collection := resources.NewRemoteResources()
	collection.Set(ResourceType, resourceMap)
	return collection, nil
}

// MapRemoteToState turns the managed remote connections into state keyed on
// externalId. Endpoints resolve through the merged cross-provider collection
// into PropertyRefs shaped exactly like the spec side, so the differ compares
// cleanly. A row whose endpoint is not CLI-managed cannot be expressed as spec
// refs and is skipped with a warning, mirroring the event stream handler; any
// other lookup or conversion failure is fatal rather than silent state loss.
func (h *Handler) MapRemoteToState(collection *resources.RemoteResources) (*state.State, error) {
	s := state.EmptyState()
	for _, remote := range collection.GetAll(ResourceType) {
		managed, ok := remote.Data.(*RemoteConnection)
		if !ok {
			return nil, fmt.Errorf("unable to cast resource to rETL connection")
		}
		conn := managed.RETLConnection

		sourceURN, err := resolveSourceURN(collection, conn.SourceID)
		switch {
		case errors.Is(err, resources.ErrRemoteResourceNotFound),
			errors.Is(err, resources.ErrRemoteResourceExternalIdNotFound):
			logSkip(conn, fmt.Errorf("source %q is not managed by the CLI", conn.SourceID))
			continue
		case err != nil:
			return nil, fmt.Errorf("resolving source urn for connection %q: %w", conn.ExternalID, err)
		}

		destinationURN, err := collection.GetURNByID(destination.DestinationResourceType, conn.DestinationID)
		switch {
		case errors.Is(err, resources.ErrRemoteResourceNotFound),
			errors.Is(err, resources.ErrRemoteResourceExternalIdNotFound):
			logSkip(conn, fmt.Errorf("destination %q is not managed by the CLI", conn.DestinationID))
			continue
		case err != nil:
			return nil, fmt.Errorf("resolving destination urn for connection %q: %w", conn.ExternalID, err)
		}

		config, err := configToMap(managed.Config)
		if err != nil {
			return nil, fmt.Errorf("reading remote connection %q: %w", conn.ExternalID, err)
		}

		s.AddResource(&state.ResourceState{
			ID:   conn.ExternalID,
			Type: ResourceType,
			Input: map[string]any{
				SourceKey:      &resources.PropertyRef{URN: sourceURN, Property: "id"},
				DestinationKey: &resources.PropertyRef{URN: destinationURN, Property: "id"},
				EnabledKey:     conn.Enabled,
				ConfigKey:      config,
			},
			Output: *toResourceData(&conn),
		})
	}
	return s, nil
}

// resolveSourceURN looks the source up under every rETL source kind: the
// connection row names an id, and which kind's collection holds it is exactly
// what SourceKinds enumerates.
//
// Every failure reports as a plain miss. GetURNByID separates an id no
// collection holds from one whose resource carries no externalId, but
// MapRemoteToState skips the row on either, so keeping them apart here could
// not change what the caller does. Returning the miss after the loop also
// covers an empty SourceKinds, which would otherwise yield an empty URN and no
// error at all.
func resolveSourceURN(collection *resources.RemoteResources, sourceID string) (string, error) {
	for _, sourceKind := range SourceKinds {
		urn, err := collection.GetURNByID(sourceKind.ResourceType, sourceID)
		if err == nil {
			return urn, nil
		}
	}
	return "", resources.ErrRemoteResourceNotFound
}

// LoadImportable lists the connections not yet managed by the CLI and names
// each after its endpoints, e.g. "users-to-webhook". Both endpoints are known
// to exist: a row missing either is not importable in the first place.
func (h *Handler) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.RemoteResources, error) {
	remotes, err := h.eligible(ctx, lo.ToPtr(false))
	if err != nil {
		return nil, err
	}

	resourceMap := make(map[string]*resources.RemoteResource, len(remotes))
	for _, remote := range remotes {
		externalID, err := idNamer.Name(namer.ScopeName{
			Name:  fmt.Sprintf("%s-to-%s", remote.SourceName, remote.DestinationName),
			Scope: ResourceType,
		})
		if err != nil {
			return nil, fmt.Errorf("generating externalID for connection %s: %w", remote.ID, err)
		}
		resourceMap[remote.ID] = &resources.RemoteResource{
			ID:         remote.ID,
			ExternalID: externalID,
			Reference:  fmt.Sprintf("#%s:%s", ResourceKind, externalID),
			Data:       remote,
		}
	}
	collection := resources.NewRemoteResources()
	collection.Set(ResourceType, resourceMap)
	return collection, nil
}

// FormatForExport writes the importable connections as one spec of the
// retl-connections kind per run.
func (h *Handler) FormatForExport(
	collection *resources.RemoteResources,
	_ namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, []importmanifest.ImportEntry, error) {
	remotesByID := collection.GetAll(ResourceType)
	if len(remotesByID) == 0 {
		return nil, nil, nil
	}

	// One spec file holds every connection: order by assigned id so the emitted
	// list is stable across runs.
	remotes := make([]*resources.RemoteResource, 0, len(remotesByID))
	for _, remote := range remotesByID {
		remotes = append(remotes, remote)
	}
	slices.SortFunc(remotes, func(a, b *resources.RemoteResource) int {
		return cmp.Compare(a.ExternalID, b.ExternalID)
	})

	var (
		// The id doubles as the "not set yet" sentinel: remoteConnection
		// already refuses a row whose source reports no workspace, so an empty
		// one cannot reach here.
		workspaceMetadata = specs.WorkspaceImportMetadata{
			Resources: make([]specs.ImportIds, 0, len(remotes)),
		}
		entries []importmanifest.ImportEntry
	)
	items := make([]map[string]any, 0, len(remotes))
	for _, remote := range remotes {
		data, ok := remote.Data.(*RemoteConnection)
		if !ok {
			return nil, nil, fmt.Errorf("unable to cast remote resource to rETL connection")
		}
		// Matched connections (import --merge) adopt an existing local spec:
		// manifest entry only, no spec entry. Every other row has to survive
		// toImportItem first, because a row skipped there contributes nothing —
		// the export's workspace included.
		var item map[string]any
		if remote.MatchedWith == nil {
			built, err := toImportItem(remote.ExternalID, data, inputResolver)
			if err != nil {
				logSkip(data.RETLConnection, err)
				continue
			}
			item = built
		}

		if workspaceMetadata.WorkspaceID != "" && workspaceMetadata.WorkspaceID != data.WorkspaceID {
			return nil, nil, fmt.Errorf("cannot export resources from multiple workspaces into a single spec file")
		}
		workspaceMetadata.WorkspaceID = data.WorkspaceID

		urn := resources.URN(remote.ExternalID, ResourceType)
		entries = append(entries, importmanifest.ImportEntry{
			WorkspaceID: data.WorkspaceID,
			URN:         urn,
			RemoteID:    remote.ID,
		})
		if item == nil {
			continue
		}
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
		ResourceKind,
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

// toImportItem builds one connection's spec entry: both endpoint refs resolved
// through the merged collection — imported in the same run or already managed —
// and the config in the canonical shape a spec would produce.
func toImportItem(externalID string, data *RemoteConnection, inputResolver resolver.ReferenceResolver) (map[string]any, error) {
	sourceRef, err := esConnection.EndpointRef(inputResolver, data.SourceKind.ResourceType, data.SourceKind.Kind, data.SourceID, data.SourceExternalID)
	if err != nil {
		return nil, fmt.Errorf("resolving source reference: %w", err)
	}
	destinationRef, err := esConnection.EndpointRef(inputResolver, destination.DestinationResourceType, destination.DestinationSpecKind, data.DestinationID, data.DestinationExternalID)
	if err != nil {
		return nil, fmt.Errorf("resolving destination reference: %w", err)
	}
	config, err := configToMap(data.Config)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		IDKey:          externalID,
		SourceKey:      sourceRef,
		DestinationKey: destinationRef,
		EnabledKey:     data.Enabled,
		ConfigKey:      config,
	}, nil
}

// List reports the workspace's rETL connections, honouring the provider's
// hasExternalId filter; rows carrying an externalId are the CLI-managed ones.
//
// Deliberately unfiltered by the eligibility rules the other remote-reading
// paths apply: this answers "what exists in the workspace", not "what can the
// CLI express". A connection whose destination has no registered definition, or
// whose flow is unsupported, is listed here even though apply skips it.
func (h *Handler) List(ctx context.Context, hasExternalID *bool) ([]resources.ResourceData, error) {
	conns, err := h.listAll(ctx, hasExternalID)
	if err != nil {
		return nil, err
	}

	result := make([]resources.ResourceData, 0, len(conns))
	for _, conn := range conns {
		result = append(result, resources.ResourceData{
			IDKey:            conn.ID,
			SourceIDKey:      conn.SourceID,
			DestinationIDKey: conn.DestinationID,
			EnabledKey:       conn.Enabled,
			ExternalIDKey:    conn.ExternalID,
		})
	}
	return result, nil
}

// Import adopts an existing remote connection into CLI management: it refuses
// anything the CLI cannot express or does not own before touching it, pushes
// the spec through the same Update a regular apply runs, and claims the
// identity last, so a failed reconciliation leaves nothing half-adopted.
func (h *Handler) Import(ctx context.Context, id string, data resources.ResourceData, remoteID string) (*resources.ResourceData, error) {
	remote, err := h.client.GetConnection(ctx, remoteID)
	if err != nil {
		return nil, fmt.Errorf("getting rETL connection during import: %w", err)
	}
	if remote == nil {
		return nil, fmt.Errorf("getting rETL connection during import: connection %q came back empty", remoteID)
	}
	if remote.ExternalID != "" && remote.ExternalID != id {
		return nil, fmt.Errorf("connection %q is already managed by the CLI as %q", remoteID, remote.ExternalID)
	}

	sources, destinations, err := h.endpoints(ctx)
	if err != nil {
		return nil, err
	}
	eligible, err := h.remoteConnection(*remote, sources, destinations)
	if err != nil {
		return nil, fmt.Errorf("importing rETL connection: %w", err)
	}

	config, err := configToMap(eligible.Config)
	if err != nil {
		return nil, fmt.Errorf("importing rETL connection %q: %w", remoteID, err)
	}
	existing := resources.ResourceData{
		IDKey:            remote.ID,
		SourceIDKey:      remote.SourceID,
		DestinationIDKey: remote.DestinationID,
		EnabledKey:       remote.Enabled,
		ConfigKey:        config,
	}

	_, _, replace, err := replacementNeeded(data, existing)
	if err != nil {
		return nil, fmt.Errorf("connection %q: %w", id, err)
	}
	result, err := h.Update(ctx, id, data, existing)
	if err != nil {
		return nil, fmt.Errorf("updating rETL connection during import: %w", err)
	}
	claimed, ok := (*result)[IDKey].(string)
	if !ok || claimed == "" {
		return nil, fmt.Errorf("importing rETL connection %q: reconciliation returned no connection id", id)
	}

	// Which path ran decides what is left to claim, not the returned id. A
	// replacement built its row through Create, which carries the externalId in
	// the body, so that row is already this connection's.
	if replace {
		return result, nil
	}

	// Re-importing a row the CLI already owns has nothing left to claim.
	if remote.ExternalID == id {
		return result, nil
	}

	// Adoption is the one path with something left to claim: the row predates
	// this apply, so no create body carried the externalId, and a PUT cannot —
	// UpdateRETLConnectionRequest has no such field. A failure here leaves the
	// connection reconciled but unowned; re-running the import adopts it.
	if err := h.client.SetConnectionExternalId(ctx, &retlClient.SetRETLConnectionExternalIDRequest{
		ID: claimed, ExternalID: id,
	}); err != nil {
		return nil, fmt.Errorf("setting external ID for rETL connection during import: %w", err)
	}
	return result, nil
}
