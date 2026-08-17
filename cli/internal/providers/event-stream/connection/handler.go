package connection

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-viper/mapstructure/v2"
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

// errNotImplemented guards the lifecycle surface until DEX-650 wires the
// connections API client; spec parsing and graph construction work today.
var errNotImplemented = errors.New("event stream connection apply support is not implemented yet")

type Handler struct {
	resources map[string]*connectionResource
}

func NewHandler() *Handler {
	return &Handler{
		resources: make(map[string]*connectionResource),
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
			URN:             resources.URN(id, ResourceType),
			JSONPointerPath: fmt.Sprintf("/spec/connections/%d/id", i),
		})
	}
	return &specs.ParsedSpec{URNs: entries}, nil
}

func (h *Handler) LoadSpec(_ string, s *specs.Spec) error {
	spec := &ConnectionsSpec{}
	// Use strict decoding to reject unknown fields — a config key on a
	// connection must fail at parse time: connections are pure links and
	// connection settings live on the destination spec.
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
	// A nil slice means the connections key was absent: fail loudly instead of
	// letting the file silently contribute nothing. An explicit empty list
	// stays valid so removing the last connection does not invalidate the spec.
	if spec.Connections == nil {
		return fmt.Errorf("connections not found in event stream connections spec")
	}
	for i := range spec.Connections {
		resource, err := h.loadConnection(&spec.Connections[i], i)
		if err != nil {
			return err
		}
		// When we are at this point, we expect the spec along with the
		// localID to be valid and unique (see project/duplicate-urn rule)
		h.resources[resource.LocalID] = resource
	}
	return nil
}

func (h *Handler) loadConnection(c *ConnectionSpec, index int) (*connectionResource, error) {
	if c.LocalID == "" {
		return nil, fmt.Errorf("id is required for connection at index %d", index)
	}
	if c.Source == "" {
		return nil, fmt.Errorf("source is required for connection %q", c.LocalID)
	}
	if c.Destination == "" {
		return nil, fmt.Errorf("destination is required for connection %q", c.LocalID)
	}

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

// GetResources attaches each connection as raw data so the syncer's
// reflection-based dereference resolves the typed destination ref and the
// graph derives dependency edges on both endpoints.
func (h *Handler) GetResources() ([]*resources.Resource, error) {
	result := make([]*resources.Resource, 0, len(h.resources))
	for _, c := range h.resources {
		r := resources.NewResource(
			c.LocalID,
			ResourceType,
			resources.ResourceData{},
			[]string{},
			resources.WithResourceFileMetadata(fmt.Sprintf("#%s:%s", ResourceType, c.LocalID)),
			resources.WithRawData(c),
		)
		result = append(result, r)
	}
	return result, nil
}

// parseSourceRef parses a scalar "#event-stream-source:<id>" reference. The
// ref resolves via the source's legacy state output map, which carries the
// remote id under "id".
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

// LoadImportMetadata is a no-op: import support for connections lands with
// DEX-654.
func (h *Handler) LoadImportMetadata(_ *specs.WorkspacesImportMetadata) error {
	return nil
}

// LoadResourcesFromRemote returns an empty collection until DEX-650 wires the
// connections API client — no remote connections are managed yet, and apply
// must keep working for the other event stream resource types.
func (h *Handler) LoadResourcesFromRemote(_ context.Context) (*resources.RemoteResources, error) {
	return resources.NewRemoteResources(), nil
}

func (h *Handler) MapRemoteToState(_ *resources.RemoteResources) (*state.State, error) {
	return state.EmptyState(), nil
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

func (h *Handler) Create(_ context.Context, _ string, _ resources.ResourceData) (*resources.ResourceData, error) {
	return nil, errNotImplemented
}

func (h *Handler) Update(_ context.Context, _ string, _ resources.ResourceData, _ resources.ResourceData) (*resources.ResourceData, error) {
	return nil, errNotImplemented
}

func (h *Handler) Delete(_ context.Context, _ string, _ resources.ResourceData) error {
	return errNotImplemented
}

func (h *Handler) List(_ context.Context, _ lister.Filters) ([]resources.ResourceData, error) {
	return nil, errNotImplemented
}

func (h *Handler) Import(_ context.Context, _ string, _ resources.ResourceData, _ string) (*resources.ResourceData, error) {
	return nil, errNotImplemented
}
