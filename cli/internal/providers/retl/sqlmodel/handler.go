package sqlmodel

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/go-viper/mapstructure/v2"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

// Handler implements the resourceHandler interface for SQL Model resources
type Handler struct {
	client    retlClient.RETLStore
	resources map[string]*SQLModelResource
	importDir string
}

// NewHandler creates a new SQL Model resource handler
func NewHandler(client retlClient.RETLStore, importDir string) *Handler {
	return &Handler{
		client:    client,
		resources: make(map[string]*SQLModelResource),
		importDir: filepath.Join(importDir, ImportPath),
	}
}

func (h *Handler) ParseSpec(_ string, s *specs.Spec) (*specs.ParsedSpec, error) {
	id, ok := s.Spec["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id not found in sql model spec")
	}
	return &specs.ParsedSpec{ExternalIDs: []string{id}}, nil
}

// LoadSpec loads and validates a SQL Model spec
func (h *Handler) LoadSpec(path string, s *specs.Spec) error {
	spec := &SQLModelSpec{}

	// Convert spec map to struct using mapstructure
	if err := mapstructure.Decode(s.Spec, spec); err != nil {
		return fmt.Errorf("converting spec: %w", err)
	}

	if _, ok := h.resources[spec.ID]; ok {
		return fmt.Errorf("sql model with id %s already exists", spec.ID)
	}

	if spec.SQL == nil && spec.File == nil {
		return fmt.Errorf("sql or file must be specified")
	}

	if spec.SQL != nil && spec.File != nil {
		return fmt.Errorf("sql and file cannot be specified together")
	}
	sqlStr := ""
	if spec.SQL != nil {
		sqlStr = *spec.SQL
	} else {
		filePath := *spec.File
		if !filepath.IsAbs(filePath) {
			// If path is relative, resolve it relative to the spec file
			// This properly handles "../" in paths
			specDir := filepath.Dir(path)
			filePath = filepath.Clean(filepath.Join(specDir, filePath))
		}

		sqlContent, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("reading SQL file %s (resolved to %s): %w", *spec.File, filePath, err)
		}
		sqlStr = string(sqlContent)
	}

	// Default Enabled to true if not specified
	enabled := true
	if spec.Enabled != nil {
		enabled = *spec.Enabled
	}

	// Create resource with SQL directly from spec
	h.resources[spec.ID] = &SQLModelResource{
		ID:               spec.ID,
		DisplayName:      spec.DisplayName,
		Description:      spec.Description,
		AccountID:        spec.AccountID,
		PrimaryKey:       spec.PrimaryKey,
		SourceDefinition: string(spec.SourceDefinition),
		Enabled:          enabled,
		SQL:              sqlStr,
	}

	return h.loadImportMetadata(s)
}

func (h *Handler) loadImportMetadata(s *specs.Spec) error {
	metadata, err := s.CommonMetadata()
	if err != nil {
		return err
	}

	if metadata.Import != nil {
		workspaces := metadata.Import.Workspaces
		for _, workspaceMetadata := range workspaces {
			workspaceId := workspaceMetadata.WorkspaceID
			resources := workspaceMetadata.Resources
			for _, resourceMetadata := range resources {
				importMetadata[resourceMetadata.LocalID] = &ImportResourceInfo{
					WorkspaceId: workspaceId,
					RemoteId:    resourceMetadata.RemoteID,
				}
			}
		}
	}

	return nil
}

// Validate validates all loaded SQL Model specs
func (h *Handler) Validate() error {
	for _, spec := range h.resources {
		if err := ValidateSQLModelResource(spec); err != nil {
			return fmt.Errorf("validating sql model spec: %w", err)
		}
	}
	return nil
}

// GetResources returns all SQL Model resources
func (h *Handler) GetResources() ([]*resources.Resource, error) {
	result := make([]*resources.Resource, 0, len(h.resources))

	for _, spec := range h.resources {
		// Convert spec to resource data
		data := resources.ResourceData{
			LocalIDKey:          spec.ID,
			DisplayNameKey:      spec.DisplayName,
			DescriptionKey:      spec.Description,
			AccountIDKey:        spec.AccountID,
			PrimaryKeyKey:       spec.PrimaryKey,
			SourceDefinitionKey: spec.SourceDefinition,
			EnabledKey:          spec.Enabled,
			SQLKey:              spec.SQL,
		}

		var opts []resources.ResourceOpts
		if importMetadata, ok := importMetadata[spec.ID]; ok {
			opts = []resources.ResourceOpts{
				resources.WithResourceImportMetadata(importMetadata.RemoteId, importMetadata.WorkspaceId),
			}
		}
		resource := resources.NewResource(
			spec.ID,
			ResourceType,
			data,
			[]string{}, // No dependencies for now
			opts...,
		)
		result = append(result, resource)
	}

	return result, nil
}

// Create creates a new SQL Model resource
func (h *Handler) Create(ctx context.Context, ID string, data resources.ResourceData) (*resources.ResourceData, error) {
	source := &retlClient.RETLSourceCreateRequest{
		Name:                 data[DisplayNameKey].(string),
		Config:               toRETLSQLModelConfig(data),
		SourceType:           retlClient.ModelSourceType,
		SourceDefinitionName: data[SourceDefinitionKey].(string),
		AccountID:            data[AccountIDKey].(string),
		Enabled:              data[EnabledKey].(bool),
		ExternalID:           ID,
	}

	// Call API to create RETL source
	resp, err := h.client.CreateRetlSource(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("creating RETL source: %w", err)
	}

	return toResourceData(resp), nil
}

// Update updates an existing SQL Model resource
func (h *Handler) Update(ctx context.Context, ID string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	// Get source_id from state - needed for API call
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
		Config:    toRETLSQLModelConfig(data),
		IsEnabled: data[EnabledKey].(bool),
		AccountID: data[AccountIDKey].(string),
	}

	// Call API to update RETL source
	resp, err := h.client.UpdateRetlSource(ctx, sourceID, source)
	if err != nil {
		return nil, fmt.Errorf("updating RETL source: %w", err)
	}

	return toResourceData(resp), nil
}

// Delete deletes an existing SQL Model resource
func (h *Handler) Delete(ctx context.Context, ID string, state resources.ResourceData) error {
	// Get source_id from state - needed for API call
	sourceID, ok := state[IDKey].(string)
	if !ok {
		return fmt.Errorf("missing %s in resource state", IDKey)
	}

	// Call API to delete RETL source
	if err := h.client.DeleteRetlSource(ctx, sourceID); err != nil {
		return fmt.Errorf("deleting RETL source: %w", err)
	}

	return nil
}

func (h *Handler) List(ctx context.Context, hasExternalId *bool) ([]resources.ResourceData, error) {
	sources, err := h.client.ListRetlSources(ctx, hasExternalId)
	if err != nil {
		return nil, fmt.Errorf("listing RETL sources: %w", err)
	}
	re := regexp.MustCompile(`\s+`)
	var resourceData []resources.ResourceData
	for _, source := range sources.Data {
		// Replace newlines with spaces and collapse multiple spaces into one
		sql := re.ReplaceAllString(source.Config.Sql, " ")
		resourceData = append(resourceData, resources.ResourceData{
			IDKey:               source.ID,
			"name":              source.Name,
			AccountIDKey:        source.AccountID,
			SourceDefinitionKey: source.SourceDefinitionName,
			CreatedAtKey:        source.CreatedAt,
			UpdatedAtKey:        source.UpdatedAt,
			"config": map[string]interface{}{
				PrimaryKeyKey:  source.Config.PrimaryKey,
				SQLKey:         sql,
				DescriptionKey: source.Config.Description,
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

	err = h.client.SetExternalId(ctx, remoteId, ID)
	if err != nil {
		return nil, fmt.Errorf("setting external ID for RETL source: %w", err)
	}

	existingState := &SQLModelResource{}
	existingState.FromResourceData(*toResourceData(existingSource))

	currentState := &SQLModelResource{}
	currentState.FromResourceData(data)

	changed := currentState.DiffUpstream(existingState)
	result := toResourceData(existingSource)
	if changed {
		updatedData, err := h.updateCall(ctx, remoteId, data)
		if err != nil {
			return nil, fmt.Errorf("updating RETL source: %w", err)
		}
		result = updatedData
	}
	return result, nil
}

func (h *Handler) FetchImportData(ctx context.Context, args specs.ImportIds) (writer.FormattableEntity, error) {
	if args.LocalID == "" {
		return writer.FormattableEntity{}, fmt.Errorf("local id is required")
	}
	if args.RemoteID == "" {
		return writer.FormattableEntity{}, fmt.Errorf("remote id is required")
	}

	// First, get all sources to find the one we want to import
	source, err := h.client.GetRetlSource(ctx, args.RemoteID)
	if err != nil {
		return writer.FormattableEntity{}, fmt.Errorf("getting RETL source for import: %w", err)
	}
	// Validate that this is a SQL model source
	if source.SourceType != retlClient.ModelSourceType {
		return writer.FormattableEntity{}, fmt.Errorf("source %s is not a SQL model (type: %s)", args.RemoteID, source.SourceType)
	}

	// Create the base resource data structure for the imported source
	importedData := resources.ResourceData{
		IDKey:               args.LocalID,
		DisplayNameKey:      source.Name,
		DescriptionKey:      source.Config.Description,
		AccountIDKey:        source.AccountID,
		PrimaryKeyKey:       source.Config.PrimaryKey,
		SourceDefinitionKey: source.SourceDefinitionName,
		EnabledKey:          source.IsEnabled,
		SQLKey:              source.Config.Sql,
	}

	importMetadata := specs.Metadata{
		Name: args.LocalID,
		Import: &specs.WorkspacesImportMetadata{
			Workspaces: []specs.WorkspaceImportMetadata{
				{
					WorkspaceID: source.WorkspaceID,
					Resources: []specs.ImportIds{
						{
							LocalID:  args.LocalID,
							RemoteID: args.RemoteID,
						},
					},
				},
			},
		},
	}

	// Convert metadata to map for spec
	metadataMap, err := importMetadata.ToMap()
	if err != nil {
		return writer.FormattableEntity{}, fmt.Errorf("converting metadata to map: %w", err)
	}

	// Create the spec with the imported data
	spec := &specs.Spec{
		Version:  specs.SpecVersion,
		Kind:     ResourceType,
		Metadata: metadataMap,
		Spec:     importedData,
	}

	// Return as FormattableEntity
	return writer.FormattableEntity{
		Content:      spec,
		RelativePath: fmt.Sprintf("%s.yaml", args.LocalID),
	}, nil
}

func (h *Handler) LoadResourcesFromRemote(ctx context.Context) (*resources.RemoteResources, error) {
	collection := resources.NewRemoteResources()
	hasExternalID := true
	sources, err := h.client.ListRetlSources(ctx, &hasExternalID)
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
	sqlModelResources := collection.GetAll(ResourceType)
	for _, resource := range sqlModelResources {
		source, ok := resource.Data.(retlClient.RETLSource)
		if !ok {
			return nil, fmt.Errorf("unable to cast resource to retl source")
		}
		input := resources.ResourceData{
			DisplayNameKey:      source.Name,
			DescriptionKey:      source.Config.Description,
			AccountIDKey:        source.AccountID,
			PrimaryKeyKey:       source.Config.PrimaryKey,
			SQLKey:              source.Config.Sql,
			EnabledKey:          source.IsEnabled,
			SourceDefinitionKey: source.SourceDefinitionName,
			LocalIDKey:          source.ExternalID,
		}
		output := toResourceData(&source)
		s.AddResource(&state.ResourceState{
			Type:   ResourceType,
			ID:     source.ExternalID,
			Input:  input,
			Output: *output,
		})
	}
	return s, nil
}

func (h *Handler) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.RemoteResources, error) {
	collection := resources.NewRemoteResources()
	hasExternalID := false
	sources, err := h.client.ListRetlSources(ctx, &hasExternalID)
	if err != nil {
		return nil, fmt.Errorf("listing RETL sources: %w", err)
	}
	resourceMap := make(map[string]*resources.RemoteResource)
	for _, source := range sources.Data {
		externalID, err := idNamer.Name(namer.ScopeName{
			Name:  source.Name,
			Scope: ResourceType,
		})
		if err != nil {
			return nil, fmt.Errorf("generating externalID for source %s: %w", source.Name, err)
		}
		resourceMap[source.ID] = &resources.RemoteResource{
			ID:         source.ID,
			ExternalID: externalID,
			Data:       &source,
			Reference: fmt.Sprintf("#/%s/%s/%s",
				ResourceKind,
				MetadataName,
				externalID,
			),
		}
	}
	collection.Set(ResourceType, resourceMap)
	return collection, nil
}

func (h *Handler) FormatForExport(collection *resources.RemoteResources, idNamer namer.Namer, inputResolver resolver.ReferenceResolver) ([]writer.FormattableEntity, error) {
	sources := collection.GetAll(ResourceType)
	if len(sources) == 0 {
		return nil, nil
	}
	workspaceMetadata := specs.WorkspaceImportMetadata{
		Resources: make([]specs.ImportIds, 0),
	}
	var result []writer.FormattableEntity
	for _, source := range sources {
		sourceData, ok := source.Data.(*retlClient.RETLSource)
		if !ok {
			return nil, fmt.Errorf("unable to cast resource to retl source")
		}
		workspaceMetadata.WorkspaceID = sourceData.WorkspaceID
		workspaceMetadata.Resources = []specs.ImportIds{
			{
				LocalID:  source.ExternalID,
				RemoteID: source.ID,
			},
		}

		metadata := specs.Metadata{
			Name: source.ExternalID,
			Import: &specs.WorkspacesImportMetadata{
				Workspaces: []specs.WorkspaceImportMetadata{workspaceMetadata},
			},
		}

		metadataMap, err := metadata.ToMap()
		if err != nil {
			return nil, err
		}

		spec := &specs.Spec{
			Version:  specs.SpecVersion,
			Kind:     ResourceKind,
			Metadata: metadataMap,
			Spec: map[string]interface{}{
				DisplayNameKey:      sourceData.Name,
				DescriptionKey:      sourceData.Config.Description,
				AccountIDKey:        sourceData.AccountID,
				PrimaryKeyKey:       sourceData.Config.PrimaryKey,
				SQLKey:              sourceData.Config.Sql,
				SourceDefinitionKey: sourceData.SourceDefinitionName,
				EnabledKey:          sourceData.IsEnabled,
				IDKey:               source.ExternalID,
			},
		}
		result = append(result, writer.FormattableEntity{
			Content:      spec,
			RelativePath: filepath.Join(h.importDir, fmt.Sprintf("%s.yaml", source.ExternalID)),
		})
	}
	return result, nil
}

func toResourceData(source *retlClient.RETLSource) *resources.ResourceData {
	result := resources.ResourceData{
		DisplayNameKey:      source.Name,
		DescriptionKey:      source.Config.Description,
		AccountIDKey:        source.AccountID,
		PrimaryKeyKey:       source.Config.PrimaryKey,
		SQLKey:              source.Config.Sql,
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
	return &result
}

func toRETLSQLModelConfig(data resources.ResourceData) retlClient.RETLSQLModelConfig {
	description, ok := data[DescriptionKey].(string)
	if !ok {
		description = data[DisplayNameKey].(string)
	}
	return retlClient.RETLSQLModelConfig{
		PrimaryKey:  data[PrimaryKeyKey].(string),
		Sql:         data[SQLKey].(string),
		Description: description,
	}
}
