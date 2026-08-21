package connection

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/api/client"
	esClient "github.com/rudderlabs/rudder-iac/api/client/event-stream"
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

// ErrNotImplemented guards the list and import surfaces until DEX-654; the
// apply lifecycle (create/update/delete + remote state) is implemented.
var ErrNotImplemented = errors.New("event stream connection support is not implemented yet")

type Handler struct {
	resources map[string]*connectionResource
	client    esClient.EventStreamStore
}

func NewHandler(client esClient.EventStreamStore) *Handler {
	return &Handler{
		resources: make(map[string]*connectionResource),
		client:    client,
	}
}

// ParseSpec collects one URN per connection entry — the spec body is a list,
// unlike the single-resource event stream source spec.
func (h *Handler) ParseSpec(_ string, s *specs.Spec) (*specs.ParsedSpec, error) {
	raw, ok := s.Spec["connections"].([]any)
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
		LocalID:     c.LocalID,
		Source:      sourceRef,
		Destination: destinationRef,
		Enabled:     enabled,
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
		r := resources.NewResource(
			c.LocalID,
			EventStreamConnectionResourceType,
			data,
			[]string{},
			resources.WithResourceFileMetadata(fmt.Sprintf("#%s:%s", EventStreamConnectionResourceKind, c.LocalID)),
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

// ScalarRefRegex matches a well-formed scalar reference "#<kind>:<id>". It is
// the single definition of the connection reference format: refID parses with
// it below, and the connection spec syntax rule matches against it, so parsing
// and validation cannot drift apart. The id side deliberately accepts any
// non-empty single-line value — endpoint local ids carry no charset
// restriction, but neither the kind nor the id may span multiple lines.
var ScalarRefRegex = regexp.MustCompile(`^#([a-zA-Z0-9_-]+):(.+)$`)

// refID extracts <id> from a scalar "#<kind>:<id>" reference.
func refID(ref string, kind string) (string, error) {
	matches := ScalarRefRegex.FindStringSubmatch(strings.TrimSpace(ref))
	if matches == nil || matches[1] != kind {
		return "", fmt.Errorf("invalid reference %q: expected format #%s:<id>", ref, kind)
	}
	return matches[2], nil
}

// LoadImportMetadata is a no-op: import support for connections lands with
// DEX-654.
func (h *Handler) LoadImportMetadata(_ *specs.WorkspacesImportMetadata) error {
	return nil
}

// LoadResourcesFromRemote lists the remote connections that carry an
// externalId — the CLI-managed ones. The generic connections list also
// returns rETL rows; only connections whose source is an event stream source
// are kept.
func (h *Handler) LoadResourcesFromRemote(ctx context.Context) (*resources.RemoteResources, error) {
	collection := resources.NewRemoteResources()

	sources, err := h.client.GetSources(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting event stream sources: %w", err)
	}
	eventStreamSourceIDs := make(map[string]struct{}, len(sources))
	for _, s := range sources {
		eventStreamSourceIDs[s.ID] = struct{}{}
	}

	resourceMap := make(map[string]*resources.RemoteResource)
	page, err := h.client.ListConnections(ctx, client.WithConnectionsHasExternalID(true))
	for page != nil && err == nil {
		for _, conn := range page.Connections {
			if _, ok := eventStreamSourceIDs[conn.SourceID]; !ok {
				continue
			}
			resourceMap[conn.ID] = &resources.RemoteResource{
				ID:         conn.ID,
				ExternalID: conn.ExternalID,
				Data:       conn,
			}
		}
		page, err = h.client.NextConnections(ctx, page.Paging)
	}
	if err != nil {
		return nil, fmt.Errorf("listing event stream connections: %w", err)
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

func (h *Handler) LoadImportable(_ context.Context, _ namer.Namer) (*resources.RemoteResources, error) {
	return resources.NewRemoteResources(), nil
}

func (h *Handler) FormatForExport(
	_ *resources.RemoteResources,
	_ namer.Namer,
	_ resolver.ReferenceResolver,
) ([]writer.FormattableEntity, []importmanifest.ImportEntry, error) {
	return nil, nil, nil
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

func (h *Handler) List(_ context.Context, _ lister.Filters) ([]resources.ResourceData, error) {
	return nil, ErrNotImplemented
}

func (h *Handler) Import(_ context.Context, _ string, _ resources.ResourceData, _ string) (*resources.ResourceData, error) {
	return nil, ErrNotImplemented
}
