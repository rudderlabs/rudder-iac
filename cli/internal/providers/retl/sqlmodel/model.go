package sqlmodel

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sourcekeys"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// SourceDefinition aliases sourcekeys.Definition so existing signatures keep
// working; new kinds should use sourcekeys.Definition directly.
type SourceDefinition = sourcekeys.Definition

// ResourceType is the type identifier for SQL Model resources
const (
	ResourceType = "retl-source-sql-model"
	ResourceKind = "retl-source-sql-model"
	MetadataName = "retl-source-sql-model"
	ImportPath   = "sql-models"

	// SQL-model's own payload keys. Everything else a source carries is
	// shared vocabulary and lives in the sourcekeys package.
	DescriptionKey = "description"
	SQLKey         = "sql"
	FileKey        = "file"

	// Aliases for the shared keys, kept so the references already spread
	// through this package and its tests keep compiling. New code should
	// read them from the sourcekeys package.
	LocalIDKey          = sourcekeys.LocalIDKey
	DisplayNameKey      = sourcekeys.DisplayNameKey
	AccountIDKey        = sourcekeys.AccountIDKey
	PrimaryKeyKey       = sourcekeys.PrimaryKeyKey
	SourceDefinitionKey = sourcekeys.SourceDefinitionKey
	EnabledKey          = sourcekeys.EnabledKey
	IDKey               = sourcekeys.IDKey
	SourceTypeKey       = sourcekeys.SourceTypeKey
	CreatedAtKey        = sourcekeys.CreatedAtKey
	UpdatedAtKey        = sourcekeys.UpdatedAtKey

	SourceDefinitionPostgres   = sourcekeys.DefinitionPostgres
	SourceDefinitionRedshift   = sourcekeys.DefinitionRedshift
	SourceDefinitionSnowflake  = sourcekeys.DefinitionSnowflake
	SourceDefinitionBigQuery   = sourcekeys.DefinitionBigQuery
	SourceDefinitionMySQL      = sourcekeys.DefinitionMySQL
	SourceDefinitionDatabricks = sourcekeys.DefinitionDatabricks
	SourceDefinitionTrino      = sourcekeys.DefinitionTrino
)

// validSourceDefinitions contains all valid source definition values. SQL
// models run a query, so they accept the warehouse definitions and not s3.
var validSourceDefinitions = func() map[SourceDefinition]bool {
	m := make(map[SourceDefinition]bool, len(sourcekeys.WarehouseDefinitions))
	for _, d := range sourcekeys.WarehouseDefinitions {
		m[d] = true
	}
	return m
}()

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
	AccountID        string           `json:"account_id"        mapstructure:"account_id"        validate:"required_without=Account,excluded_with=Account"`
	Account          string           `json:"account"           mapstructure:"account"           validate:"omitempty,pattern=account_ref"`
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
	// AccountLocalID is the local id of the account the spec references, parsed
	// out of its "#account:<id>" form, and empty when the spec sets account_id.
	// Only loaded specs carry it: remote sources and dereferenced data hold the
	// resolved AccountID. Named for the parsed id because table.TableSpec's
	// Account field holds the raw reference instead.
	AccountLocalID string `json:"account"`
}

// accountValue is the resource's graph value under AccountIDKey: the raw id, or
// a reference that resolves to it.
func (s *SQLModelResource) accountValue() any {
	if s.AccountLocalID != "" {
		return AccountRef(s.AccountLocalID)
	}
	return s.AccountID
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
