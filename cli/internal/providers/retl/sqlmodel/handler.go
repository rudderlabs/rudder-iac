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
	specs  []*SQLModelResource
}

// NewHandler creates a new SQL Model resource handler
func NewHandler(client retlClient.RETLStore) *Handler {
	return &Handler{
		client: client,
		specs:  []*SQLModelResource{},
	}
}

// LoadSpec loads and validates a SQL Model spec
func (h *Handler) LoadSpec(path string, s *specs.Spec) error {
	spec := &SQLModelSpec{}

	// Convert spec map to struct using mapstructure
	if err := mapstructure.Decode(s.Spec, spec); err != nil {
		return fmt.Errorf("converting spec: %w", err)
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

	h.specs = append(h.specs, &SQLModelResource{
		ID:                   spec.ID,
		DisplayName:          spec.DisplayName,
		Description:          spec.Description,
		AccountID:            spec.AccountID,
		PrimaryKey:           spec.PrimaryKey,
		SourceDefinitionName: spec.SourceDefinitionName,
		Enabled:              spec.Enabled,
		SQL:                  *spec.SQL,
	})
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
			"id":                     spec.ID,
			"display_name":           spec.DisplayName,
			"description":            spec.Description,
			"account_id":             spec.AccountID,
			"primary_key":            spec.PrimaryKey,
			"source_definition_name": spec.SourceDefinitionName,
			"enabled":                spec.Enabled,
			"sql":                    spec.SQL,
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
	// Convert resource data to RETL source
	sqlContent := ""
	if sql, ok := data["sql"].(string); ok {
		sqlContent = sql
	}

	source := &retlClient.RETLSourceCreateRequest{
		Name: data["display_name"].(string),
		Config: retlClient.RETLSourceConfig{
			PrimaryKey:  data["primary_key"].(string),
			Sql:         sqlContent,
			Description: data["description"].(string),
		},
		SourceType:           "model",
		SourceDefinitionName: data["source_definition_name"].(string),
		AccountID:            data["account_id"].(string),
	}

	// Call API to create RETL source
	resp, err := h.client.CreateRetlSource(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("creating RETL source: %w", err)
	}

	// Convert API response to resource data
	result := resources.ResourceData{
		"id":                     ID,
		"display_name":           resp.Name,
		"description":            resp.Config.Description,
		"account_id":             resp.AccountID,
		"primary_key":            resp.Config.PrimaryKey,
		"sql":                    resp.Config.Sql,
		"source_id":              resp.ID, // Store the remote source ID
		"source_type":            resp.SourceType,
		"enabled":                resp.IsEnabled,
		"source_definition_name": resp.SourceDefinitionName,
	}

	if resp.CreatedAt != nil {
		result["created_at"] = resp.CreatedAt
	}

	return &result, nil
}

// Update updates an existing SQL Model resource
func (h *Handler) Update(ctx context.Context, ID string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	// Get source_id from state - needed for API call
	sourceID, ok := state["source_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing source_id in resource state")
	}

	// Convert resource data to RETL source
	sqlContent := ""
	if sql, ok := data["sql"].(string); ok {
		sqlContent = sql
	}

	source := &retlClient.RETLSourceUpdateRequest{
		SourceID: sourceID,
		Name:     data["display_name"].(string),
		Config: retlClient.RETLSourceConfig{
			PrimaryKey:  data["primary_key"].(string),
			Sql:         sqlContent,
			Description: data["description"].(string),
		},
		IsEnabled: data["enabled"].(bool),
		AccountID: data["account_id"].(string),
	}

	// Call API to update RETL source
	resp, err := h.client.UpdateRetlSource(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("updating RETL source: %w", err)
	}

	// Convert API response to resource data
	result := resources.ResourceData{
		"id":                     ID,
		"display_name":           resp.Name,
		"description":            resp.Config.Description,
		"account_id":             resp.AccountID,
		"primary_key":            resp.Config.PrimaryKey,
		"sql":                    resp.Config.Sql,
		"source_id":              resp.ID,
		"source_type":            resp.SourceType,
		"enabled":                resp.IsEnabled,
		"source_definition_name": resp.SourceDefinitionName,
	}

	if resp.CreatedAt != nil {
		result["created_at"] = resp.CreatedAt
	}

	if resp.UpdatedAt != nil {
		result["updated_at"] = resp.UpdatedAt
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
