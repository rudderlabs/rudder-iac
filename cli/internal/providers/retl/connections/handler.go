package connections

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

var log = logger.New("retl-connection")

// Handler implements the retl provider's resourceHandler interface for RETL
// connections. One spec document yields many resources.
type Handler struct {
	client    retlClient.RETLStore
	resources map[string]*connectionResource
}

func NewHandler(client retlClient.RETLStore) *Handler {
	return &Handler{
		client:    client,
		resources: make(map[string]*connectionResource),
	}
}

// ParseSpec emits one URN per list entry — a plural kind contributes several
// resources from a single document.
func (h *Handler) ParseSpec(_ string, s *specs.Spec) (*specs.ParsedSpec, error) {
	raw, ok := s.Spec[ConnectionsKey].([]any)
	if !ok {
		return nil, fmt.Errorf("connections list not found in retl connections spec")
	}
	entries := make([]specs.URNEntry, 0, len(raw))
	for i, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("connection entry %d is not a mapping", i)
		}
		id, ok := entry["id"].(string)
		if !ok {
			return nil, fmt.Errorf("connection entry %d is missing id", i)
		}
		entries = append(entries, specs.URNEntry{
			URN:             resources.URN(id, ResourceType),
			JSONPointerPath: fmt.Sprintf("/spec/connections/%d/id", i),
		})
	}
	return &specs.ParsedSpec{URNs: entries, LegacyResourceType: ResourceType}, nil
}

func (h *Handler) LoadSpec(_ string, s *specs.Spec) error {
	spec := &ConnectionsSpec{}

	decoderConfig := &mapstructure.DecoderConfig{
		ErrorUnused: true, // reject unknown fields
		Result:      spec,
	}
	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return fmt.Errorf("creating decoder: %w", err)
	}
	if err := decoder.Decode(s.Spec); err != nil {
		return fmt.Errorf("decoding retl connections spec: %w", err)
	}

	for _, c := range spec.Connections {
		res, err := h.loadConnection(c)
		if err != nil {
			return err
		}
		if _, exists := h.resources[res.LocalID]; exists {
			return fmt.Errorf("retl connection with id %s already exists", res.LocalID)
		}
		h.resources[res.LocalID] = res
	}
	return h.loadImportMetadata(s)
}

func (h *Handler) loadConnection(c ConnectionSpec) (*connectionResource, error) {
	sourceRef, err := parseSourceRef(c.Source)
	if err != nil {
		return nil, fmt.Errorf("connection %q: parsing source reference: %w", c.LocalID, err)
	}
	destinationRef, err := parseDestinationRef(c.Destination)
	if err != nil {
		return nil, fmt.Errorf("connection %q: parsing destination reference: %w", c.LocalID, err)
	}

	// Cross-field rules that today live only in the Terraform provider's
	// CustomizeDiff and the webapp's form. Enforced here so the failure is
	// local rather than a 400.
	if c.CursorColumn != "" && c.SyncBehaviour != "upsert" {
		return nil, fmt.Errorf("connection %q: cursor_column requires sync_behaviour: upsert", c.LocalID)
	}
	if c.Schedule.Type == "basic" {
		if c.Schedule.EveryMinutes == nil {
			return nil, fmt.Errorf("connection %q: schedule.every_minutes is required for a basic schedule", c.LocalID)
		}
		if *c.Schedule.EveryMinutes < 5 {
			return nil, fmt.Errorf("connection %q: schedule.every_minutes must be at least 5", c.LocalID)
		}
	}
	if c.Schedule.Type == "cron" && c.Schedule.CronExpression == "" {
		return nil, fmt.Errorf("connection %q: schedule.cron_expression is required for a cron schedule", c.LocalID)
	}
	if c.Event != nil && c.Event.Name != "" && c.Event.NameColumn != "" {
		return nil, fmt.Errorf("connection %q: event.name and event.name_column are mutually exclusive", c.LocalID)
	}

	enabled := true
	if c.Enabled != nil {
		enabled = *c.Enabled
	}

	return &connectionResource{
		LocalID:       c.LocalID,
		Source:        sourceRef,
		Destination:   destinationRef,
		Enabled:       enabled,
		SyncBehaviour: c.SyncBehaviour,
		CursorColumn:  c.CursorColumn,
		Object:        c.Object,
		Schedule:      c.Schedule,
		Event:         c.Event,
		Identifiers:   c.Identifiers,
		Mappings:      c.Mappings,
		Constants:     c.Constants,
		SyncSettings:  c.SyncSettings,
	}, nil
}

func (h *Handler) loadImportMetadata(s *specs.Spec) error {
	metadata, err := s.CommonMetadata()
	if err != nil {
		return err
	}
	return h.LoadImportMetadata(metadata.Import)
}

func (h *Handler) LoadImportMetadata(m *specs.WorkspacesImportMetadata) error {
	if m == nil {
		return nil
	}
	for _, ws := range m.Workspaces {
		for _, rm := range ws.Resources {
			urn := rm.URN
			if urn == "" {
				urn = resources.URN(rm.LocalID, ResourceType)
			}
			importMetadata[urn] = &ImportResourceInfo{WorkspaceId: ws.WorkspaceID, RemoteId: rm.RemoteID}
		}
	}
	return nil
}

func (h *Handler) GetResources() ([]*resources.Resource, error) {
	result := make([]*resources.Resource, 0, len(h.resources))
	for _, c := range h.resources {
		data := resources.ResourceData{
			SourceKey:        c.Source,
			DestinationKey:   c.Destination,
			EnabledKey:       c.Enabled,
			SyncBehaviourKey: c.SyncBehaviour,
			CursorColumnKey:  c.CursorColumn,
			ObjectKey:        c.Object,
			ScheduleKey:      c.Schedule,
			EventKey:         c.Event,
			IdentifiersKey:   c.Identifiers,
			MappingsKey:      c.Mappings,
			ConstantsKey:     c.Constants,
			SyncSettingsKey:  c.SyncSettings,
		}
		opts := []resources.ResourceOpts{
			resources.WithResourceFileMetadata(fmt.Sprintf("#%s:%s", ResourceKind, c.LocalID)),
		}
		urn := resources.URN(c.LocalID, ResourceType)
		if meta, ok := importMetadata[urn]; ok {
			opts = []resources.ResourceOpts{
				resources.WithResourceImportMetadata(meta.RemoteId, meta.WorkspaceId),
			}
		}
		result = append(result, resources.NewResource(c.LocalID, ResourceType, data, []string{}, opts...))
	}
	return result, nil
}

func (h *Handler) Create(ctx context.Context, ID string, data resources.ResourceData) (*resources.ResourceData, error) {
	req, err := toCreateRequest(ID, data)
	if err != nil {
		return nil, err
	}
	resp, err := h.client.CreateConnection(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("creating RETL connection: %w", err)
	}

	// The API rejects externalId inline on create — "Fields not allowed for
	// JSON Mapper flow: externalId" (400) — even though
	// CreateRETLConnectionRequest carries the field. It has to be claimed in a
	// second call, unlike RETL sources where ExternalID on create is accepted.
	if err := h.client.SetConnectionExternalId(ctx, &retlClient.SetRETLConnectionExternalIDRequest{
		ID:         resp.ID,
		ExternalID: ID,
	}); err != nil {
		return nil, fmt.Errorf("setting external ID on new RETL connection %s: %w", resp.ID, err)
	}
	resp.ExternalID = ID

	return toResourceData(resp), nil
}

// Update forwards mutable changes. source, destination, sync_behaviour and
// cursor_column are immutable — Terraform marks all four ForceNew, and
// identifiers deliberately mutable. Rejecting here rather than silently
// producing a no-op keeps the CLI honest about what needs a replace.
func (h *Handler) Update(ctx context.Context, ID string, data resources.ResourceData, st resources.ResourceData) (*resources.ResourceData, error) {
	connectionID, ok := st[IDKey].(string)
	if !ok {
		return nil, fmt.Errorf("missing %s in resource state", IDKey)
	}

	for _, immutable := range []struct {
		key   string
		label string
	}{
		{SyncBehaviourKey, "sync_behaviour"},
		{CursorColumnKey, "cursor_column"},
	} {
		newVal, _ := data[immutable.key].(string)
		oldVal, hadOld := st[immutable.key].(string)
		if hadOld && newVal != oldVal {
			return nil, fmt.Errorf("%s cannot be changed on an existing connection; the connection must be replaced", immutable.label)
		}
	}

	req, err := toUpdateRequest(data)
	if err != nil {
		return nil, err
	}
	resp, err := h.client.UpdateConnection(ctx, connectionID, req)
	if err != nil {
		return nil, fmt.Errorf("updating RETL connection: %w", err)
	}
	return toResourceData(resp), nil
}

func (h *Handler) Delete(ctx context.Context, ID string, st resources.ResourceData) error {
	connectionID, ok := st[IDKey].(string)
	if !ok {
		return fmt.Errorf("missing %s in resource state", IDKey)
	}
	if err := h.client.DeleteConnection(ctx, connectionID); err != nil {
		return fmt.Errorf("deleting RETL connection: %w", err)
	}
	return nil
}

func (h *Handler) List(ctx context.Context, hasExternalId *bool) ([]resources.ResourceData, error) {
	page, err := h.client.ListConnections(ctx, &retlClient.ListRETLConnectionsRequest{HasExternalID: hasExternalId})
	if err != nil {
		return nil, fmt.Errorf("listing RETL connections: %w", err)
	}
	out := make([]resources.ResourceData, 0, len(page.Data))
	for i := range page.Data {
		out = append(out, *toResourceData(&page.Data[i]))
	}
	return out, nil
}

func (h *Handler) Import(ctx context.Context, ID string, data resources.ResourceData, remoteId string) (*resources.ResourceData, error) {
	existing, err := h.client.GetConnection(ctx, remoteId)
	if err != nil {
		return nil, fmt.Errorf("getting RETL connection: %w", err)
	}
	if err := h.client.SetConnectionExternalId(ctx, &retlClient.SetRETLConnectionExternalIDRequest{
		ID:         remoteId,
		ExternalID: ID,
	}); err != nil {
		return nil, fmt.Errorf("setting external ID for RETL connection: %w", err)
	}
	return toResourceData(existing), nil
}

func (h *Handler) LoadResourcesFromRemote(ctx context.Context) (*resources.RemoteResources, error) {
	collection := resources.NewRemoteResources()
	page, err := h.client.ListConnections(ctx, &retlClient.ListRETLConnectionsRequest{})
	if err != nil {
		return nil, fmt.Errorf("listing RETL connections: %w", err)
	}
	m := make(map[string]*resources.RemoteResource)
	for _, c := range page.Data {
		if c.ExternalID == "" {
			continue
		}
		conn := c
		m[c.ID] = &resources.RemoteResource{ID: c.ID, ExternalID: c.ExternalID, Data: conn}
	}
	collection.Set(ResourceType, m)
	return collection, nil
}

func (h *Handler) MapRemoteToState(collection *resources.RemoteResources) (*state.State, error) {
	st := state.EmptyState()
	for _, r := range collection.GetAll(ResourceType) {
		conn, ok := r.Data.(retlClient.RETLConnection)
		if !ok {
			return nil, fmt.Errorf("unable to cast resource to retl connection")
		}

		// Endpoint URNs come from the sibling collections. A connection whose
		// source or destination is not CLI-managed is skipped rather than
		// failed — the same stance event-stream takes.
		sourceURN, err := sourceURNFor(collection, conn.SourceID)
		if err != nil {
			log.Warn("skipping connection whose source is not managed by the CLI",
				"connection", conn.ExternalID, "sourceId", conn.SourceID)
			continue
		}
		destinationURN, err := collection.GetURNByID(destination.DestinationResourceType, conn.DestinationID)
		if err != nil {
			log.Warn("skipping connection whose destination is not managed by the CLI",
				"connection", conn.ExternalID, "destinationId", conn.DestinationID)
			continue
		}

		st.AddResource(&state.ResourceState{
			Type:   ResourceType,
			ID:     conn.ExternalID,
			Input:  toSpecShapedInput(&conn, sourceURN, destinationURN),
			Output: *toResourceData(&conn),
		})
	}
	return st, nil
}

// sourceURNFor resolves a connection's source against either RETL source kind.
func sourceURNFor(collection *resources.RemoteResources, sourceID string) (string, error) {
	for _, rt := range []string{table.ResourceType, sqlmodel.ResourceType} {
		if urn, err := collection.GetURNByID(rt, sourceID); err == nil {
			return urn, nil
		}
	}
	return "", fmt.Errorf("source %s not found in any managed RETL source collection", sourceID)
}

// Preview is meaningless for a connection.
func (h *Handler) Preview(ctx context.Context, ID string, data resources.ResourceData, limit int) ([]map[string]any, error) {
	return nil, fmt.Errorf("preview is not supported for %s resources", ResourceType)
}

// ponytail: import out of scope for the spike. LoadImportable and
// FormatForExport return empty because the provider fans both across every
// handler — an erroring stub breaks import and export for sibling kinds.
func (h *Handler) FetchImportData(ctx context.Context, args specs.ImportIds) (writer.FormattableEntity, error) {
	return writer.FormattableEntity{}, fmt.Errorf("import is not yet implemented for %s", ResourceType)
}

func (h *Handler) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.RemoteResources, error) {
	return resources.NewRemoteResources(), nil
}

func (h *Handler) FormatForExport(
	collection *resources.RemoteResources,
	idNamer namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, []importmanifest.ImportEntry, error) {
	return nil, nil, nil
}

// parseSourceRef accepts a reference to either RETL source kind, so a
// connection can activate a table or a SQL model.
func parseSourceRef(ref string) (*resources.PropertyRef, error) {
	for kind, resourceType := range map[string]string{
		table.ResourceKind:    table.ResourceType,
		sqlmodel.ResourceKind: sqlmodel.ResourceType,
	} {
		if id, ok := refID(ref, kind); ok {
			return &resources.PropertyRef{URN: resources.URN(id, resourceType), Property: "id"}, nil
		}
	}
	return nil, fmt.Errorf("expected a reference of the form #%s:<id> or #%s:<id>, got %q",
		table.ResourceKind, sqlmodel.ResourceKind, ref)
}

// parseDestinationRef resolves "#destination:<id>" through the destination
// provider's typed state.
//
// A naive &resources.PropertyRef{URN, Property: "id"} does NOT work here, and
// fails only at apply time with "destination is missing or of unexpected type
// <nil>". RETL source refs get away with it because those handlers put a
// literal "id" key in their output ResourceData; a destination's state is a
// typed DestinationState struct, so it needs a resolver. Mirrors
// event-stream's parseDestinationRef.
func parseDestinationRef(ref string) (*resources.PropertyRef, error) {
	id, ok := refID(ref, destination.DestinationSpecKind)
	if !ok {
		return nil, fmt.Errorf("expected a reference of the form #%s:<id>, got %q", destination.DestinationSpecKind, ref)
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
	// Stamp "id" so the differ's comparePropertyRefs sees a stable shape on
	// both the spec and state sides — same workaround event-stream carries.
	propertyRef.Property = "id"
	return propertyRef, nil
}

func refID(ref, kind string) (string, bool) {
	prefix := "#" + kind + ":"
	if !strings.HasPrefix(ref, prefix) {
		return "", false
	}
	id := strings.TrimPrefix(ref, prefix)
	if id == "" {
		return "", false
	}
	return id, true
}
