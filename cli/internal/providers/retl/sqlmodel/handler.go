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
	client retlClient.RETLStore
	specs  map[string]*SQLModelResource
}

// NewHandler creates a new SQL Model resource handler
func NewHandler(client retlClient.RETLStore) *Handler {
	return &Handler{
		client: client,
		specs:  make(map[string]*SQLModelResource),
	}
}

// LoadSpec loads and validates a SQL Model spec
func (h *Handler) LoadSpec(path string, s *specs.Spec) error {
	spec := &SQLModelSpec{}

	// Convert spec map to struct using mapstructure
	if err := mapstructure.Decode(s.Spec, spec); err != nil {
		return fmt.Errorf("converting spec: %w", err)
	}

	if _, ok := h.specs[spec.ID]; ok {
		return fmt.Errorf("sql model with id %s already exists", spec.ID)
	}

	if spec.SQL == nil && spec.File == nil {
		return fmt.Errorf("sql or file must be specified")
	}

	if spec.SQL != nil && spec.File != nil {
		return fmt.Errorf("sql and file cannot be specified together")
	}

	// If file path is specified, load SQL from file
	if spec.File != nil && spec.SQL == nil {
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

		sqlStr := string(sqlContent)
		spec.SQL = &sqlStr
	}

	h.specs[spec.ID] = &SQLModelResource{
		ID:                   spec.ID,
		DisplayName:          spec.DisplayName,
		Description:          spec.Description,
		AccountID:            spec.AccountID,
		PrimaryKey:           spec.PrimaryKey,
		SourceDefinitionName: spec.SourceDefinitionName,
		Enabled:              spec.Enabled,
		SQL:                  *spec.SQL,
	}
	return nil
}

// Validate validates all loaded SQL Model specs
func (h *Handler) Validate() error {
	for _, spec := range h.specs {
		if err := ValidateSQLModelResource(spec); err != nil {
			return fmt.Errorf("validating sql model spec: %w", err)
		}
	}
	return nil
}

// GetResources returns all SQL Model resources
func (h *Handler) GetResources() ([]*resources.Resource, error) {
	result := make([]*resources.Resource, 0, len(h.specs))

	for _, spec := range h.specs {
		// Convert spec to resource data
		data := resources.ResourceData{
			LocalIDKey:              spec.ID,
			DisplayNameKey:          spec.DisplayName,
			DescriptionKey:          spec.Description,
			AccountIDKey:            spec.AccountID,
			PrimaryKeyKey:           spec.PrimaryKey,
			SourceDefinitionNameKey: spec.SourceDefinitionName,
			EnabledKey:              spec.Enabled,
			SQLKey:                  spec.SQL,
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
	source := &retlClient.RETLSourceCreateRequest{
		Name: data[DisplayNameKey].(string),
		Config: retlClient.RETLSQLModelConfig{
			PrimaryKey:  data[PrimaryKeyKey].(string),
			Sql:         data[SQLKey].(string),
			Description: data[DescriptionKey].(string),
		},
		SourceType:           ModelSourceType,
		SourceDefinitionName: data[SourceDefinitionNameKey].(string),
		AccountID:            data[AccountIDKey].(string),
	}

	// Call API to create RETL source
	resp, err := h.client.CreateRetlSource(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("creating RETL source: %w", err)
	}

	// Convert API response to resource data
	result := resources.ResourceData{
		LocalIDKey:              ID,
		DisplayNameKey:          resp.Name,
		DescriptionKey:          resp.Config.Description,
		AccountIDKey:            resp.AccountID,
		PrimaryKeyKey:           resp.Config.PrimaryKey,
		SQLKey:                  resp.Config.Sql,
		SourceIDKey:             resp.ID, // Store the remote source ID
		SourceTypeKey:           resp.SourceType,
		EnabledKey:              resp.IsEnabled,
		SourceDefinitionNameKey: resp.SourceDefinitionName,
	}

	if resp.CreatedAt != nil {
		result[CreatedAtKey] = resp.CreatedAt
	}

	if resp.UpdatedAt != nil {
		result[UpdatedAtKey] = resp.UpdatedAt
	}

	return &result, nil
}

// Update updates an existing SQL Model resource
func (h *Handler) Update(ctx context.Context, ID string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	// Get source_id from state - needed for API call
	sourceID, ok := state[SourceIDKey].(string)
	if !ok {
		return nil, fmt.Errorf("missing source_id in resource state")
	}

	source := &retlClient.RETLSourceUpdateRequest{
		Name: data[DisplayNameKey].(string),
		Config: retlClient.RETLSQLModelConfig{
			PrimaryKey:  data[PrimaryKeyKey].(string),
			Sql:         data[SQLKey].(string),
			Description: data[DescriptionKey].(string),
		},
		IsEnabled: data[EnabledKey].(bool),
		AccountID: data[AccountIDKey].(string),
	}

	// Call API to update RETL source
	resp, err := h.client.UpdateRetlSource(ctx, sourceID, source)
	if err != nil {
		return nil, fmt.Errorf("updating RETL source: %w", err)
	}

	// Convert API response to resource data
	result := resources.ResourceData{
		LocalIDKey:              ID,
		DisplayNameKey:          resp.Name,
		DescriptionKey:          resp.Config.Description,
		AccountIDKey:            resp.AccountID,
		PrimaryKeyKey:           resp.Config.PrimaryKey,
		SQLKey:                  resp.Config.Sql,
		SourceIDKey:             resp.ID,
		SourceTypeKey:           resp.SourceType,
		EnabledKey:              resp.IsEnabled,
		SourceDefinitionNameKey: resp.SourceDefinitionName,
	}

	if resp.CreatedAt != nil {
		result[CreatedAtKey] = resp.CreatedAt
	}

	if resp.UpdatedAt != nil {
		result[UpdatedAtKey] = resp.UpdatedAt
	}

	return &result, nil
}

// Delete deletes an existing SQL Model resource
func (h *Handler) Delete(ctx context.Context, ID string, state resources.ResourceData) error {
	// Get source_id from state - needed for API call
	sourceID, ok := state["source_id"].(string)
	if !ok {
		return fmt.Errorf("missing source_id in resource state")
	}

	// Call API to delete RETL source
	if err := h.client.DeleteRetlSource(ctx, sourceID); err != nil {
		return fmt.Errorf("deleting RETL source: %w", err)
	}

	return nil
}
