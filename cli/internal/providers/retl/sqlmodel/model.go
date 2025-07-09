package sqlmodel

import "fmt"

// ResourceType is the type identifier for SQL Model resources
const ResourceType = "retl-source-sql-model"

// SQLModelSpec represents the YAML specification for a SQL Model resource
type SQLModelSpec struct {
	ID                   string  `json:"id" mapstructure:"id"`
	DisplayName          string  `json:"display_name" mapstructure:"display_name"`
	Description          string  `json:"description" mapstructure:"description"`
	File                 *string `json:"file" mapstructure:"file"`
	SQL                  *string `json:"sql" mapstructure:"sql"`
	AccountID            string  `json:"account_id" mapstructure:"account_id"`
	PrimaryKey           string  `json:"primary_key" mapstructure:"primary_key"`
	SourceDefinitionName string  `json:"source_definition_name" mapstructure:"source_definition_name"`
	Enabled              bool    `json:"enabled" mapstructure:"enabled"`
}

// SQLModelResource represents a processed SQL Model resource ready for API operations
type SQLModelResource struct {
	ID                   string `json:"id" mapstructure:"id"`
	DisplayName          string `json:"display_name" mapstructure:"display_name"`
	Description          string `json:"description" mapstructure:"description"`
	SQL                  string `json:"sql" mapstructure:"sql"`
	AccountID            string `json:"account_id" mapstructure:"account_id"`
	PrimaryKey           string `json:"primary_key" mapstructure:"primary_key"`
	SourceDefinitionName string `json:"source_definition_name" mapstructure:"source_definition_name"`
	Enabled              bool   `json:"enabled" mapstructure:"enabled"`
}

// ValidateSQLModelResource validates a SQL Model resource
func ValidateSQLModelResource(spec *SQLModelResource) error {
	if spec.ID == "" {
		return fmt.Errorf("id is required")
	}
	if spec.DisplayName == "" {
		return fmt.Errorf("display_name is required")
	}
	if spec.Description == "" {
		return fmt.Errorf("description is required")
	}
	if spec.AccountID == "" {
		return fmt.Errorf("account_id is required")
	}
	if spec.PrimaryKey == "" {
		return fmt.Errorf("primary_key is required")
	}
	if spec.SourceDefinitionName == "" {
		return fmt.Errorf("source_definition_name is required")
	}
	if spec.SQL == "" {
		return fmt.Errorf("sql is required")
	}
	return nil
}
