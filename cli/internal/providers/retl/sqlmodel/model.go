package sqlmodel

import (
	"fmt"
)

type SourceDefinition string

// ResourceType is the type identifier for SQL Model resources
const (
	ResourceType = "retl-source-sql-model"

	LocalIDKey          = "local_id"
	DisplayNameKey      = "display_name"
	DescriptionKey      = "description"
	AccountIDKey        = "account_id"
	PrimaryKeyKey       = "primary_key"
	SourceDefinitionKey = "source_definition"
	EnabledKey          = "enabled"
	SQLKey              = "sql"
	IDKey               = "id"
	SourceTypeKey       = "source_type"
	CreatedAtKey        = "createdAt"
	UpdatedAtKey        = "updatedAt"

	SourceDefinitionPostgres   SourceDefinition = "postgres"
	SourceDefinitionRedshift   SourceDefinition = "redshift"
	SourceDefinitionSnowflake  SourceDefinition = "snowflake"
	SourceDefinitionBigQuery   SourceDefinition = "bigquery"
	SourceDefinitionMySQL      SourceDefinition = "mysql"
	SourceDefinitionDatabricks SourceDefinition = "databricks"
	SourceDefinitionTrino      SourceDefinition = "trino"
)

// validSourceDefinitions contains all valid source definition values
var validSourceDefinitions = map[SourceDefinition]bool{
	SourceDefinitionPostgres:   true,
	SourceDefinitionRedshift:   true,
	SourceDefinitionSnowflake:  true,
	SourceDefinitionBigQuery:   true,
	SourceDefinitionMySQL:      true,
	SourceDefinitionDatabricks: true,
	SourceDefinitionTrino:      true,
}

// isValidSourceDefinition checks if the given source definition is valid
func isValidSourceDefinition(sd SourceDefinition) bool {
	v, ok := validSourceDefinitions[sd]
	return ok && v
}

// SQLModelSpec represents the YAML specification for a SQL Model resource
type SQLModelSpec struct {
	ID               string           `mapstructure:"id"`
	DisplayName      string           `mapstructure:"display_name"`
	Description      string           `mapstructure:"description"`
	File             *string          `mapstructure:"file"`
	SQL              *string          `mapstructure:"sql"`
	AccountID        string           `mapstructure:"account_id"`
	PrimaryKey       string           `mapstructure:"primary_key"`
	SourceDefinition SourceDefinition `mapstructure:"source_definition"`
	Enabled          bool             `mapstructure:"enabled"`
}

// SQLModelResource represents a processed SQL Model resource ready for API operations
type SQLModelResource struct {
	ID               string `json:"id"`
	DisplayName      string `json:"display_name"`
	Description      string `json:"description"`
	SQL              string `json:"sql"`
	AccountID        string `json:"account_id"`
	PrimaryKey       string `json:"primary_key"`
	SourceDefinition string `json:"source_definition"`
	Enabled          bool   `json:"enabled"`
}

// ValidateSQLModelResource validates a SQL Model resource
func ValidateSQLModelResource(spec *SQLModelResource) error {
	if spec.ID == "" {
		return fmt.Errorf("id is required")
	}
	if spec.DisplayName == "" {
		return fmt.Errorf("display_name is required")
	}
	if spec.AccountID == "" {
		return fmt.Errorf("account_id is required")
	}
	if spec.PrimaryKey == "" {
		return fmt.Errorf("primary_key is required")
	}
	if spec.SourceDefinition == "" {
		return fmt.Errorf("source_definition is required")
	}
	if !isValidSourceDefinition(SourceDefinition(spec.SourceDefinition)) {
		return fmt.Errorf("source_definition '%s' is invalid, must be one of: %v", spec.SourceDefinition, validSourceDefinitions)
	}
	if spec.SQL == "" {
		return fmt.Errorf("sql is required")
	}
	return nil
}
