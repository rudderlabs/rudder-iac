package table

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/go-viper/mapstructure/v2"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

// tableSourceTypeFilter is the sourceType query value passed to
// ListRetlSources, so this handler never sees SQL model sources.
const tableSourceTypeFilter = string(retlClient.TableSourceType)

type importInfo struct {
	workspaceID string
	remoteID    string
}

// Handler implements the RETL provider's resourceHandler interface for table
// sources. It is hand-rolled like the sqlmodel handler because the provider's
// interface diverges from handler.BaseHandler.
type Handler struct {
	client    retlClient.RETLStore
	resources map[string]*TableSpec
	importDir string
	// importMetadata is keyed by URN. It is per handler rather than package
	// level so that separate providers — and parallel tests — cannot see each
	// other's imports.
	importMetadata map[string]importInfo
}

func NewHandler(client retlClient.RETLStore, importDir string) *Handler {
	return &Handler{
		client:         client,
		resources:      make(map[string]*TableSpec),
		importDir:      filepath.Join(importDir, ImportPath),
		importMetadata: make(map[string]importInfo),
	}
}

// ParseSpec leaves LegacyResourceType empty: retl-source-table is v1-only, so
// import metadata must use urn rather than the legacy local_id.
func (h *Handler) ParseSpec(_ string, s *specs.Spec) (*specs.ParsedSpec, error) {
	id, ok := s.Spec[sqlmodel.IDKey].(string)
	if !ok {
		return nil, fmt.Errorf("id not found in table source spec")
	}
	return &specs.ParsedSpec{
		URNs: []specs.URNEntry{{
			URN:             resources.URN(id, ResourceType),
			JSONPointerPath: "/spec/id",
		}},
	}, nil
}

// LoadSpec trusts the spec's field rules to retl/table/spec-syntax-valid, which
// runs before any spec loads. The decode stays strict so a stray key — a
// description, or a nested config block — is an error, not a dropped setting.
func (h *Handler) LoadSpec(_ string, s *specs.Spec) error {
	// mapstructure leaves keys absent from the spec untouched, so enabled
	// defaults to true.
	spec := &TableSpec{Enabled: true}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused: true,
		Result:      spec,
	})
	if err != nil {
		return fmt.Errorf("creating decoder: %w", err)
	}
	if err := decoder.Decode(s.Spec); err != nil {
		return fmt.Errorf("decoding table source spec: %w", err)
	}
	// The syntax rule already checks the form; this guards data(), which has
	// no way to report a reference it cannot parse.
	if spec.Account != "" {
		if _, err := sqlmodel.ParseAccountRef(spec.Account); err != nil {
			return fmt.Errorf("parsing account reference of table source %s: %w", spec.ID, err)
		}
	}

	if _, ok := h.resources[spec.ID]; ok {
		return fmt.Errorf("table source with id %s already exists", spec.ID)
	}
	h.resources[spec.ID] = spec

	metadata, err := s.CommonMetadata()
	if err != nil {
		return fmt.Errorf("reading metadata for table source %s: %w", spec.ID, err)
	}
	return h.LoadImportMetadata(metadata.Import)
}

// LoadImportMetadata records URN → remote id mappings from inline
// metadata.import or the central import manifest. Entries without a URN are
// skipped: local_id belongs to legacy spec versions this kind never had.
func (h *Handler) LoadImportMetadata(m *specs.WorkspacesImportMetadata) error {
	if m == nil {
		return nil
	}
	for _, workspace := range m.Workspaces {
		for _, resource := range workspace.Resources {
			if resource.URN == "" {
				continue
			}
			h.importMetadata[resource.URN] = importInfo{
				workspaceID: workspace.WorkspaceID,
				remoteID:    resource.RemoteID,
			}
		}
	}
	return nil
}

func (h *Handler) GetResources() ([]*resources.Resource, error) {
	result := make([]*resources.Resource, 0, len(h.resources))
	for _, t := range h.resources {
		data := t.data()
		data[sqlmodel.LocalIDKey] = t.ID

		var opts []resources.ResourceOpts
		if info, ok := h.importMetadata[resources.URN(t.ID, ResourceType)]; ok {
			opts = append(opts, resources.WithResourceImportMetadata(info.remoteID, info.workspaceID))
		}
		result = append(result, resources.NewResource(t.ID, ResourceType, data, []string{}, opts...))
	}
	return result, nil
}

func (h *Handler) Create(ctx context.Context, ID string, data resources.ResourceData) (*resources.ResourceData, error) {
	t := fromData(data)
	source, err := h.client.CreateRetlSource(ctx, &retlClient.RETLSourceCreateRequest{
		Name:                 t.DisplayName,
		Config:               t.config(),
		SourceType:           retlClient.TableSourceType,
		SourceDefinitionName: t.SourceDefinition,
		AccountID:            t.AccountID,
		Enabled:              t.Enabled,
		ExternalID:           ID,
	})
	if err != nil {
		return nil, fmt.Errorf("creating RETL source: %w", err)
	}
	return toOutput(source)
}

// Update rejects a source_definition change instead of sending it: the update
// request has no field for it, so the API would keep the old value and report
// success.
func (h *Handler) Update(ctx context.Context, ID string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	sourceID, ok := state[sqlmodel.IDKey].(string)
	if !ok || sourceID == "" {
		return nil, fmt.Errorf("missing %s in resource state", sqlmodel.IDKey)
	}

	var (
		desired    = fromData(data)
		current, _ = state[sqlmodel.SourceDefinitionKey].(string)
	)
	if desired.SourceDefinition != current {
		return nil, fmt.Errorf("updating table source %s: source_definition cannot be changed from %q to %q", ID, current, desired.SourceDefinition)
	}
	return h.update(ctx, sourceID, desired)
}

func (h *Handler) update(ctx context.Context, sourceID string, t TableSpec) (*resources.ResourceData, error) {
	source, err := h.client.UpdateRetlSource(ctx, sourceID, &retlClient.RETLSourceUpdateRequest{
		Name:      t.DisplayName,
		Config:    t.config(),
		IsEnabled: t.Enabled,
		AccountID: t.AccountID,
	})
	if err != nil {
		return nil, fmt.Errorf("updating RETL source: %w", err)
	}
	return toOutput(source)
}

func (h *Handler) Delete(ctx context.Context, ID string, state resources.ResourceData) error {
	sourceID, ok := state[sqlmodel.IDKey].(string)
	if !ok || sourceID == "" {
		return fmt.Errorf("missing %s in resource state", sqlmodel.IDKey)
	}
	if err := h.client.DeleteRetlSource(ctx, sourceID); err != nil {
		return fmt.Errorf("deleting RETL source: %w", err)
	}
	return nil
}

func (h *Handler) List(ctx context.Context, hasExternalID *bool) ([]resources.ResourceData, error) {
	sources, err := h.client.ListRetlSources(ctx, retlClient.WithSourceType(tableSourceTypeFilter), retlClient.WithHasExternalId(hasExternalID))
	if err != nil {
		return nil, fmt.Errorf("listing RETL sources: %w", err)
	}

	var result []resources.ResourceData
	for _, source := range sources.Data {
		t, err := fromRemote(&source)
		if err != nil {
			return nil, err
		}
		result = append(result, resources.ResourceData{
			sqlmodel.IDKey:               source.ID,
			sqlmodel.ExternalIDKey:       source.ExternalID,
			"name":                       source.Name,
			sqlmodel.AccountIDKey:        source.AccountID,
			sqlmodel.SourceDefinitionKey: source.SourceDefinitionName,
			sqlmodel.CreatedAtKey:        source.CreatedAt,
			sqlmodel.UpdatedAtKey:        source.UpdatedAt,
			"config":                     map[string]any(t.configData()),
		})
	}
	return result, nil
}

// Import claims an existing remote source for the local spec by setting its
// external id, then updates it if the spec differs.
func (h *Handler) Import(ctx context.Context, ID string, data resources.ResourceData, remoteID string) (*resources.ResourceData, error) {
	source, err := h.client.GetRetlSource(ctx, remoteID)
	if err != nil {
		return nil, fmt.Errorf("getting RETL source: %w", err)
	}
	if source.SourceType != retlClient.TableSourceType {
		return nil, fmt.Errorf("importing RETL source %s: source type is %q, not %q", remoteID, source.SourceType, retlClient.TableSourceType)
	}

	remote, err := fromRemote(source)
	if err != nil {
		return nil, err
	}
	local := fromData(data)
	// Checked before claiming the source, so a mismatch leaves it untouched.
	if local.SourceDefinition != remote.SourceDefinition {
		return nil, fmt.Errorf("importing RETL source %s: source_definition is %q remotely and %q locally, and cannot be changed", remoteID, remote.SourceDefinition, local.SourceDefinition)
	}

	if err := h.client.SetExternalId(ctx, remoteID, ID); err != nil {
		return nil, fmt.Errorf("setting external ID for RETL source: %w", err)
	}

	// The local id is not part of the remote source, so it does not count as
	// a difference.
	local.ID = ""
	if local == remote {
		return toOutput(source)
	}
	updated, err := h.update(ctx, remoteID, local)
	if err != nil {
		return nil, fmt.Errorf("importing RETL source: %w", err)
	}
	return updated, nil
}

func (h *Handler) LoadResourcesFromRemote(ctx context.Context) (*resources.RemoteResources, error) {
	hasExternalID := true
	sources, err := h.client.ListRetlSources(ctx, retlClient.WithSourceType(tableSourceTypeFilter), retlClient.WithHasExternalId(&hasExternalID))
	if err != nil {
		return nil, fmt.Errorf("listing RETL sources: %w", err)
	}

	resourceMap := make(map[string]*resources.RemoteResource, len(sources.Data))
	for _, source := range sources.Data {
		resourceMap[source.ID] = &resources.RemoteResource{
			ID:         source.ID,
			ExternalID: source.ExternalID,
			Data:       source,
		}
	}
	collection := resources.NewRemoteResources()
	collection.Set(ResourceType, resourceMap)
	return collection, nil
}

func (h *Handler) MapRemoteToState(collection *resources.RemoteResources) (*state.State, error) {
	s := state.EmptyState()
	for _, resource := range collection.GetAll(ResourceType) {
		source, ok := resource.Data.(retlClient.RETLSource)
		if !ok {
			return nil, fmt.Errorf("unable to cast resource to retl source")
		}
		remote, err := fromRemote(&source)
		if err != nil {
			return nil, err
		}
		output, err := toOutput(&source)
		if err != nil {
			return nil, err
		}

		local, ok := h.resources[source.ExternalID]
		input := remote.data()
		input[sqlmodel.LocalIDKey] = source.ExternalID
		input[sqlmodel.AccountIDKey] = sqlmodel.AccountInput(source.AccountID, ok && local.Account == "", collection)
		s.AddResource(&state.ResourceState{
			Type:   ResourceType,
			ID:     source.ExternalID,
			Input:  input,
			Output: *output,
		})
	}
	return s, nil
}

// Preview is not supported: a table source has no query to run, and the
// webapp's table flow has no preview step either.
func (h *Handler) Preview(_ context.Context, _ string, _ resources.ResourceData, _ int) ([]map[string]any, error) {
	return nil, fmt.Errorf("preview is not supported for %s resources", ResourceType)
}

// FetchImportData backs the single-source `import retl-source` command, which
// only handles SQL models. Table sources are imported through `import
// workspace` (LoadImportable and FormatForExport).
func (h *Handler) FetchImportData(_ context.Context, _ specs.ImportIds) (writer.FormattableEntity, error) {
	return writer.FormattableEntity{}, fmt.Errorf("single-source import is not supported for %s resources", ResourceType)
}

func (h *Handler) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.RemoteResources, error) {
	hasExternalID := false
	sources, err := h.client.ListRetlSources(ctx, retlClient.WithSourceType(tableSourceTypeFilter), retlClient.WithHasExternalId(&hasExternalID))
	if err != nil {
		return nil, fmt.Errorf("listing RETL sources: %w", err)
	}

	resourceMap := make(map[string]*resources.RemoteResource, len(sources.Data))
	for _, source := range sources.Data {
		externalID, err := idNamer.Name(namer.ScopeName{
			Name:  source.Name,
			Scope: ResourceType,
		})
		if err != nil {
			return nil, fmt.Errorf("generating external ID for source %s: %w", source.Name, err)
		}
		resourceMap[source.ID] = &resources.RemoteResource{
			ID:         source.ID,
			ExternalID: externalID,
			Data:       &source,
			Reference:  fmt.Sprintf("#%s:%s", ResourceKind, externalID),
		}
	}
	collection := resources.NewRemoteResources()
	collection.Set(ResourceType, resourceMap)
	return collection, nil
}

func (h *Handler) FormatForExport(
	collection *resources.RemoteResources,
	_ namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, []importmanifest.ImportEntry, error) {
	sources := collection.GetAll(ResourceType)
	if len(sources) == 0 {
		return nil, nil, nil
	}

	var (
		entities []writer.FormattableEntity
		entries  []importmanifest.ImportEntry
	)
	for _, source := range sources {
		data, ok := source.Data.(*retlClient.RETLSource)
		if !ok {
			return nil, nil, fmt.Errorf("unable to cast resource to retl source")
		}
		remote, err := fromRemote(data)
		if err != nil {
			return nil, nil, err
		}

		urn := resources.URN(source.ExternalID, ResourceType)
		entries = append(entries, importmanifest.ImportEntry{
			WorkspaceID: data.WorkspaceID,
			URN:         urn,
			RemoteID:    source.ID,
		})

		// A source matched to an existing local spec (import --merge) only
		// needs its manifest entry; the spec already exists.
		if source.MatchedWith != nil {
			continue
		}

		metadata := &specs.Metadata{
			Name: source.ExternalID,
			Import: &specs.WorkspacesImportMetadata{
				Workspaces: []specs.WorkspaceImportMetadata{{
					WorkspaceID: data.WorkspaceID,
					Resources:   []specs.ImportIds{{URN: urn, RemoteID: source.ID}},
				}},
			},
		}
		metadataMap, err := metadata.ToMap()
		if err != nil {
			return nil, nil, fmt.Errorf("converting metadata for table source %s: %w", source.ExternalID, err)
		}

		fields := remote.specFields(source.ExternalID)
		accountKey, account := sqlmodel.ExportAccount(data.AccountID, inputResolver)
		fields[accountKey] = account

		entities = append(entities, writer.FormattableEntity{
			Content: &specs.Spec{
				Version:  specs.SpecVersionV1,
				Kind:     ResourceKind,
				Metadata: metadataMap,
				Spec:     fields,
			},
			RelativePath: filepath.Join(h.importDir, fmt.Sprintf("%s.yaml", source.ExternalID)),
		})
	}
	return entities, entries, nil
}
