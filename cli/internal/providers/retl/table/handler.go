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
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

// tableSourceTypeFilter is the sourceType query value passed to ListRetlSources.
const tableSourceTypeFilter = string(retlClient.TableSourceType)

// Handler implements the retl provider's resourceHandler interface for
// warehouse table sources.
//
// ponytail: hand-rolled to match its sibling sqlmodel handler and the retl
// provider's bespoke resourceHandler interface. The framework's generic
// handler.BaseHandler (used by accounts, destination, datagraph,
// transformations) would collapse most of this, but retl's interface diverges
// from it — GetResources vs Resources, a narrower List signature, and two
// RETL-only methods (Preview, FetchImportData). Migrating retl onto BaseHandler
// is worth doing; it is a provider-wide change that would touch sqlmodel, so it
// does not belong in a spike.
type Handler struct {
	client    retlClient.RETLStore
	resources map[string]*TableResource
	importDir string
}

func NewHandler(client retlClient.RETLStore, importDir string) *Handler {
	return &Handler{
		client:    client,
		resources: make(map[string]*TableResource),
		importDir: filepath.Join(importDir, ImportPath),
	}
}

func (h *Handler) ParseSpec(_ string, s *specs.Spec) (*specs.ParsedSpec, error) {
	id, ok := s.Spec["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id not found in table source spec")
	}
	return &specs.ParsedSpec{
		URNs: []specs.URNEntry{{
			URN:             resources.URN(id, ResourceType),
			JSONPointerPath: "/spec/id",
		}},
		LegacyResourceType: ResourceType,
	}, nil
}

func (h *Handler) LoadSpec(path string, s *specs.Spec) error {
	spec := &TableSpec{}

	decoderConfig := &mapstructure.DecoderConfig{
		ErrorUnused: true, // reject unknown fields
		Result:      spec,
	}
	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return fmt.Errorf("creating decoder: %w", err)
	}
	if err := decoder.Decode(s.Spec); err != nil {
		return fmt.Errorf("decoding table source spec: %w", err)
	}

	if _, ok := h.resources[spec.ID]; ok {
		return fmt.Errorf("table source with id %s already exists", spec.ID)
	}

	if !isValidSourceDefinition(spec.SourceDefinition) {
		return fmt.Errorf("invalid source_definition %q for table source %s", spec.SourceDefinition, spec.ID)
	}

	// Default Enabled to true when unspecified, matching sqlmodel.
	enabled := true
	if spec.Enabled != nil {
		enabled = *spec.Enabled
	}

	h.resources[spec.ID] = &TableResource{
		ID:               spec.ID,
		DisplayName:      spec.DisplayName,
		AccountID:        spec.AccountID,
		PrimaryKey:       spec.PrimaryKey,
		Schema:           spec.Schema,
		Table:            spec.Table,
		SourceDefinition: string(spec.SourceDefinition),
		Enabled:          enabled,
	}
	return h.loadImportMetadata(s)
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
	for _, workspaceMetadata := range m.Workspaces {
		workspaceId := workspaceMetadata.WorkspaceID
		for _, resourceMetadata := range workspaceMetadata.Resources {
			urn := resourceMetadata.URN
			if urn == "" {
				urn = resources.URN(resourceMetadata.LocalID, ResourceType)
			}
			importMetadata[urn] = &ImportResourceInfo{
				WorkspaceId: workspaceId,
				RemoteId:    resourceMetadata.RemoteID,
			}
		}
	}
	return nil
}

func (h *Handler) GetResources() ([]*resources.Resource, error) {
	result := make([]*resources.Resource, 0, len(h.resources))

	for _, spec := range h.resources {
		data := resources.ResourceData{
			LocalIDKey:          spec.ID,
			DisplayNameKey:      spec.DisplayName,
			AccountIDKey:        spec.AccountID,
			PrimaryKeyKey:       spec.PrimaryKey,
			SchemaKey:           spec.Schema,
			TableKey:            spec.Table,
			SourceDefinitionKey: spec.SourceDefinition,
			EnabledKey:          spec.Enabled,
		}

		var opts []resources.ResourceOpts
		urn := resources.URN(spec.ID, ResourceType)
		if importMeta, ok := importMetadata[urn]; ok {
			opts = []resources.ResourceOpts{
				resources.WithResourceImportMetadata(importMeta.RemoteId, importMeta.WorkspaceId),
			}
		}
		result = append(result, resources.NewResource(
			spec.ID,
			ResourceType,
			data,
			[]string{},
			opts...,
		))
	}

	return result, nil
}

func (h *Handler) Create(ctx context.Context, ID string, data resources.ResourceData) (*resources.ResourceData, error) {
	source := &retlClient.RETLSourceCreateRequest{
		Name:                 data[DisplayNameKey].(string),
		Config:               toRETLTableConfig(data),
		SourceType:           retlClient.TableSourceType,
		SourceDefinitionName: data[SourceDefinitionKey].(string),
		AccountID:            data[AccountIDKey].(string),
		Enabled:              data[EnabledKey].(bool),
		ExternalID:           ID,
	}

	resp, err := h.client.CreateRetlSource(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("creating RETL source: %w", err)
	}

	return toResourceData(resp)
}

func (h *Handler) Update(ctx context.Context, ID string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	sourceID, ok := state[IDKey].(string)
	if !ok {
		return nil, fmt.Errorf("missing %s in resource state", IDKey)
	}

	if data[SourceDefinitionKey] != nil && data[SourceDefinitionKey].(string) != state[SourceDefinitionKey].(string) {
		return nil, fmt.Errorf("source definition name cannot be changed")
	}

	return h.updateCall(ctx, sourceID, data)
}

func (h *Handler) updateCall(ctx context.Context, sourceID string, data resources.ResourceData) (*resources.ResourceData, error) {
	source := &retlClient.RETLSourceUpdateRequest{
		Name:      data[DisplayNameKey].(string),
		Config:    toRETLTableConfig(data),
		IsEnabled: data[EnabledKey].(bool),
		AccountID: data[AccountIDKey].(string),
	}

	resp, err := h.client.UpdateRetlSource(ctx, sourceID, source)
	if err != nil {
		return nil, fmt.Errorf("updating RETL source: %w", err)
	}

	return toResourceData(resp)
}

func (h *Handler) Delete(ctx context.Context, ID string, state resources.ResourceData) error {
	sourceID, ok := state[IDKey].(string)
	if !ok {
		return fmt.Errorf("missing %s in resource state", IDKey)
	}

	if err := h.client.DeleteRetlSource(ctx, sourceID); err != nil {
		return fmt.Errorf("deleting RETL source: %w", err)
	}

	return nil
}

func (h *Handler) List(ctx context.Context, hasExternalId *bool) ([]resources.ResourceData, error) {
	sources, err := h.client.ListRetlSources(ctx, retlClient.WithSourceType(tableSourceTypeFilter), retlClient.WithHasExternalId(hasExternalId))
	if err != nil {
		return nil, fmt.Errorf("listing RETL sources: %w", err)
	}

	var resourceData []resources.ResourceData
	for _, source := range sources.Data {
		cfg, err := retlClient.DecodeConfig[retlClient.RETLTableConfig](source.Config)
		if err != nil {
			return nil, fmt.Errorf("decoding table config for source %s: %w", source.ID, err)
		}
		resourceData = append(resourceData, resources.ResourceData{
			IDKey:               source.ID,
			"name":              source.Name,
			AccountIDKey:        source.AccountID,
			SourceDefinitionKey: source.SourceDefinitionName,
			CreatedAtKey:        source.CreatedAt,
			UpdatedAtKey:        source.UpdatedAt,
			"config": map[string]interface{}{
				PrimaryKeyKey: cfg.PrimaryKey,
				SchemaKey:     cfg.Schema,
				TableKey:      cfg.Table,
			},
		})
	}

	return resourceData, nil
}

func (h *Handler) Import(ctx context.Context, ID string, data resources.ResourceData, remoteId string) (*resources.ResourceData, error) {
	existingSource, err := h.client.GetRetlSource(ctx, remoteId)
	if err != nil {
		return nil, fmt.Errorf("getting RETL source: %w", err)
	}

	if err := h.client.SetExternalId(ctx, remoteId, ID); err != nil {
		return nil, fmt.Errorf("setting external ID for RETL source: %w", err)
	}

	existing, err := toResourceData(existingSource)
	if err != nil {
		return nil, err
	}

	existingState := &TableResource{}
	existingState.FromResourceData(*existing)

	currentState := &TableResource{}
	currentState.FromResourceData(data)

	if currentState.DiffUpstream(existingState) {
		updated, err := h.updateCall(ctx, remoteId, data)
		if err != nil {
			return nil, fmt.Errorf("importing RETL source: %w", err)
		}
		return updated, nil
	}
	return existing, nil
}

func (h *Handler) LoadResourcesFromRemote(ctx context.Context) (*resources.RemoteResources, error) {
	collection := resources.NewRemoteResources()
	hasExternalID := true
	sources, err := h.client.ListRetlSources(ctx, retlClient.WithSourceType(tableSourceTypeFilter), retlClient.WithHasExternalId(&hasExternalID))
	if err != nil {
		return nil, fmt.Errorf("listing RETL sources: %w", err)
	}
	resourceMap := make(map[string]*resources.RemoteResource)
	for _, source := range sources.Data {
		resourceMap[source.ID] = &resources.RemoteResource{
			ID:         source.ID,
			ExternalID: source.ExternalID,
			Data:       source,
		}
	}
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
		cfg, err := retlClient.DecodeConfig[retlClient.RETLTableConfig](source.Config)
		if err != nil {
			return nil, fmt.Errorf("decoding table config for source %s: %w", source.ID, err)
		}
		output, err := toResourceData(&source)
		if err != nil {
			return nil, err
		}
		input := resources.ResourceData{
			DisplayNameKey:      source.Name,
			AccountIDKey:        source.AccountID,
			PrimaryKeyKey:       cfg.PrimaryKey,
			SchemaKey:           cfg.Schema,
			TableKey:            cfg.Table,
			EnabledKey:          source.IsEnabled,
			SourceDefinitionKey: source.SourceDefinitionName,
			LocalIDKey:          source.ExternalID,
		}
		s.AddResource(&state.ResourceState{
			Type:   ResourceType,
			ID:     source.ExternalID,
			Input:  input,
			Output: *output,
		})
	}
	return s, nil
}

// Preview is not supported for table sources.
//
// ponytail: sqlmodel implements Preview because a SQL model has a query to run.
// A table source has no query — the webapp's table flow has no preview step
// either (SelectSource -> SetupSource -> SourceDetails -> Review). Returning an
// error rather than silently succeeding keeps the distinction explicit.
func (h *Handler) Preview(ctx context.Context, ID string, data resources.ResourceData, limit int) ([]map[string]any, error) {
	return nil, fmt.Errorf("preview is not supported for %s resources", ResourceType)
}

// Import is out of scope for the spike.
//
// ponytail: LoadImportable and FormatForExport return empty rather than an
// error. The provider fans both out across every registered handler, so an
// erroring stub here breaks import and export for sqlmodel too — proven by
// TestProviderFormatForExport. Contributing nothing keeps table invisible to
// import while leaving its sibling working. FetchImportData is addressed by
// resource id rather than fanned out, so it can fail loudly.
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

func toResourceData(source *retlClient.RETLSource) (*resources.ResourceData, error) {
	cfg, err := retlClient.DecodeConfig[retlClient.RETLTableConfig](source.Config)
	if err != nil {
		return nil, fmt.Errorf("decoding table config for source %s: %w", source.ID, err)
	}
	result := resources.ResourceData{
		DisplayNameKey:      source.Name,
		AccountIDKey:        source.AccountID,
		PrimaryKeyKey:       cfg.PrimaryKey,
		SchemaKey:           cfg.Schema,
		TableKey:            cfg.Table,
		IDKey:               source.ID,
		SourceTypeKey:       source.SourceType,
		EnabledKey:          source.IsEnabled,
		SourceDefinitionKey: source.SourceDefinitionName,
	}
	if source.CreatedAt != nil {
		result[CreatedAtKey] = source.CreatedAt
	}
	if source.UpdatedAt != nil {
		result[UpdatedAtKey] = source.UpdatedAt
	}
	return &result, nil
}

func toRETLTableConfig(data resources.ResourceData) retlClient.RETLTableConfig {
	return retlClient.RETLTableConfig{
		PrimaryKey: data[PrimaryKeyKey].(string),
		Schema:     data[SchemaKey].(string),
		Table:      data[TableKey].(string),
	}
}
