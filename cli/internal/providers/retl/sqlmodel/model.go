package sqlmodel

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

type SourceDefinition string

// ResourceType is the type identifier for SQL Model resources
const (
	ResourceType = "retl-source-sql-model"
	ResourceKind = "retl-source-sql-model"
	MetadataName = "retl-source-sql-model"
	ImportPath   = "sql-models"

	LocalIDKey          = "local_id"
	DisplayNameKey      = "display_name"
	DescriptionKey      = "description"
	AccountIDKey        = "account_id"
	PrimaryKeyKey       = "primary_key"
	SourceDefinitionKey = "source_definition"
	EnabledKey          = "enabled"
	SQLKey              = "sql"
	FileKey             = "file"
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

type ImportResourceInfo struct {
	WorkspaceId string
	RemoteId    string
}

var importMetadata = map[string]*ImportResourceInfo{}

// isValidSourceDefinition checks if the given source definition is valid
func isValidSourceDefinition(sd SourceDefinition) bool {
	v, ok := validSourceDefinitions[sd]
	return ok && v
}

// SQLModelSpec represents the YAML specification for a SQL Model resource.
// JSON tags enable the typed rule engine's json.Marshal/Unmarshal round-trip;
// validate tags drive go-playground/validator checks.
type SQLModelSpec struct {
	ID               string           `json:"id"                mapstructure:"id"                validate:"required"`
	DisplayName      string           `json:"display_name"      mapstructure:"display_name"      validate:"required"`
	Description      string           `json:"description"       mapstructure:"description"`
	File             *string          `json:"file"              mapstructure:"file"`
	SQL              *string          `json:"sql"               mapstructure:"sql"               validate:"required_without=File,excluded_with=File"`
	AccountID        string           `json:"account_id"        mapstructure:"account_id"        validate:"required"`
	PrimaryKey       string           `json:"primary_key"       mapstructure:"primary_key"       validate:"required"`
	SourceDefinition SourceDefinition `json:"source_definition" mapstructure:"source_definition" validate:"required,oneof=postgres redshift snowflake bigquery mysql databricks trino"`
	Enabled          *bool            `json:"enabled"           mapstructure:"enabled"`
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

func (s *SQLModelResource) FromResourceData(data resources.ResourceData) {
	s.DisplayName = data[DisplayNameKey].(string)
	s.Description = data[DescriptionKey].(string)
	s.SQL = data[SQLKey].(string)
	s.AccountID = data[AccountIDKey].(string)
	s.PrimaryKey = data[PrimaryKeyKey].(string)
	s.SourceDefinition = data[SourceDefinitionKey].(string)
	s.Enabled = data[EnabledKey].(bool)
}

func (s *SQLModelResource) DiffUpstream(upstream *SQLModelResource) bool {
	if s.DisplayName != upstream.DisplayName {
		return true
	}
	if s.Description != upstream.Description {
		return true
	}
	if s.AccountID != upstream.AccountID {
		return true
	}
	if s.PrimaryKey != upstream.PrimaryKey {
		return true
	}
	if s.Enabled != upstream.Enabled {
		return true
	}
	return s.SQL != upstream.SQL
}
