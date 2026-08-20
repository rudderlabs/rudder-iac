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

// ErrNotImplemented guards the lifecycle surface until DEX-650 wires the
// connections API client; spec parsing and graph construction work today.
var ErrNotImplemented = errors.New("event stream connection apply support is not implemented yet")

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
	//
	// TODO: this is the second copy of the workaround — destination's
	// createTransformationRef (providers/destination/handler.go) stamps
	// Property the same way for the same reason. The fix is to take property
	// as a parameter in handler.CreatePropertyRef so callers cannot forget it;
	// that is a signature change across its existing call sites (datagraph,
	// destination, testutils), so it wants its own PR.
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
	return nil, ErrNotImplemented
}

func (h *Handler) Update(_ context.Context, _ string, _ resources.ResourceData, _ resources.ResourceData) (*resources.ResourceData, error) {
	return nil, ErrNotImplemented
}

func (h *Handler) Delete(_ context.Context, _ string, _ resources.ResourceData) error {
	return ErrNotImplemented
}

func (h *Handler) List(_ context.Context, _ lister.Filters) ([]resources.ResourceData, error) {
	return nil, ErrNotImplemented
}

func (h *Handler) Import(_ context.Context, _ string, _ resources.ResourceData, _ string) (*resources.ResourceData, error) {
	return nil, ErrNotImplemented
}
