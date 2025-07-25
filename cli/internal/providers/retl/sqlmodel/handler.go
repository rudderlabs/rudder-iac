package sqlmodel

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-viper/mapstructure/v2"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

// Handler implements the resourceHandler interface for SQL Model resources
type Handler struct {
	client    retlClient.RETLStore
	resources map[string]*SQLModelResource
}

// NewHandler creates a new SQL Model resource handler
func NewHandler(client retlClient.RETLStore) *Handler {
	return &Handler{
		client:    client,
		resources: make(map[string]*SQLModelResource),
	}
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
			specDir := filepath.Dir(path)
			filePath = filepath.Join(specDir, filePath)
		}

		sqlContent, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("reading SQL file %s: %w", *spec.File, err)
		}
		sqlStr = string(sqlContent)
	}

	// Create resource with SQL directly from spec
	h.resources[spec.ID] = &SQLModelResource{
		ID:               spec.ID,
		DisplayName:      spec.DisplayName,
		Description:      spec.Description,
		AccountID:        spec.AccountID,
		PrimaryKey:       spec.PrimaryKey,
		SourceDefinition: string(spec.SourceDefinition),
		Enabled:          spec.Enabled,
		SQL:              sqlStr,
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

		// Create resource with SQL Model resource type
		resource := resources.NewResource(
			spec.ID,
			ResourceType,
			data,
			[]string{}, // No dependencies for now
		)

		result = append(result, resource)
	}

	return result, nil
}

// Create creates a new SQL Model resource
func (h *Handler) Create(ctx context.Context, ID string, data resources.ResourceData) (*resources.ResourceData, error) {
	if enabled, ok := data[EnabledKey].(bool); ok && !enabled {
		return nil, fmt.Errorf("cannot create disabled sql model")
	}

	source := &retlClient.RETLSourceCreateRequest{
		Name:                 data[DisplayNameKey].(string),
		Config:               toRETLSQLModelConfig(data),
		SourceType:           retlClient.ModelSourceType,
		SourceDefinitionName: data[SourceDefinitionKey].(string),
		AccountID:            data[AccountIDKey].(string),
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
