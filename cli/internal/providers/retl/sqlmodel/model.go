package sqlmodel

import "fmt"

// ResourceType is the type identifier for SQL Model resources
const (
	ResourceType = "retl-source-sql-model"

	LocalIDKey              = "local_id"
	DisplayNameKey          = "display_name"
	DescriptionKey          = "description"
	AccountIDKey            = "account_id"
	PrimaryKeyKey           = "primary_key"
	SourceDefinitionNameKey = "source_definition_name"
	EnabledKey              = "enabled"
	SQLKey                  = "sql"
	IDKey                   = "id"
	SourceTypeKey           = "source_type"
	CreatedAtKey            = "created_at"
	UpdatedAtKey            = "updated_at"
)

// SQLModelSpec represents the YAML specification for a SQL Model resource
type SQLModelSpec struct {
	ID                   string  `mapstructure:"id"`
	DisplayName          string  `mapstructure:"display_name"`
	Description          string  `mapstructure:"description"`
	File                 *string `mapstructure:"file"`
	SQL                  *string `mapstructure:"sql"`
	AccountID            string  `mapstructure:"account_id"`
	PrimaryKey           string  `mapstructure:"primary_key"`
	SourceDefinitionName string  `mapstructure:"source_definition_name"`
	Enabled              bool    `mapstructure:"enabled"`
}

// SQLModelResource represents a processed SQL Model resource ready for API operations
type SQLModelResource struct {
	ID                   string `json:"id"`
	DisplayName          string `json:"display_name"`
	Description          string `json:"description"`
	SQL                  string `json:"sql"`
	AccountID            string `json:"account_id"`
	PrimaryKey           string `json:"primary_key"`
	SourceDefinitionName string `json:"source_definition_name"`
	Enabled              bool   `json:"enabled"`
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
