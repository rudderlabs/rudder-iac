package table

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

type SourceDefinition string

// ResourceType is the type identifier for warehouse table sources.
const (
	ResourceType = "retl-source-table"
	ResourceKind = "retl-source-table"
	MetadataName = "retl-source-table"
	ImportPath   = "tables"

	LocalIDKey          = "local_id"
	DisplayNameKey      = "display_name"
	DescriptionKey      = "description"
	AccountIDKey        = "account_id"
	PrimaryKeyKey       = "primary_key"
	SourceDefinitionKey = "source_definition"
	EnabledKey          = "enabled"
	SchemaKey           = "schema"
	TableKey            = "table"
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

// ponytail: s3 is deliberately absent. An s3 table source carries
// bucket_name/object_prefix instead of schema/table and skips the primary_key
// requirement server-side (config-backend service.ts:70). Modelling both config
// shapes behind one spec kind needs a discriminated union the spike does not
// need. Add SourceDefinitionS3 plus an S3 branch in LoadSpec when s3 lands.
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

func isValidSourceDefinition(sd SourceDefinition) bool {
	v, ok := validSourceDefinitions[sd]
	return ok && v
}

// TableSpec is the YAML shape for a warehouse table source. JSON tags feed the
// typed rule engine's marshal round-trip; validate tags drive
// go-playground/validator.
type TableSpec struct {
	ID               string           `json:"id"                mapstructure:"id"                validate:"required"`
	DisplayName      string           `json:"display_name"      mapstructure:"display_name"      validate:"required"`
	Description      string           `json:"description"       mapstructure:"description"`
	AccountID        string           `json:"account_id"        mapstructure:"account_id"        validate:"required"`
	PrimaryKey       string           `json:"primary_key"       mapstructure:"primary_key"       validate:"required"`
	Schema           string           `json:"schema"            mapstructure:"schema"            validate:"required"`
	Table            string           `json:"table"             mapstructure:"table"             validate:"required"`
	SourceDefinition SourceDefinition `json:"source_definition" mapstructure:"source_definition" validate:"required,oneof=postgres redshift snowflake bigquery mysql databricks trino"`
	Enabled          *bool            `json:"enabled"           mapstructure:"enabled"`
}

// TableResource is a loaded spec ready for API operations.
type TableResource struct {
	ID               string `json:"id"`
	DisplayName      string `json:"display_name"`
	Description      string `json:"description"`
	AccountID        string `json:"account_id"`
	PrimaryKey       string `json:"primary_key"`
	Schema           string `json:"schema"`
	Table            string `json:"table"`
	SourceDefinition string `json:"source_definition"`
	Enabled          bool   `json:"enabled"`
}

func (t *TableResource) FromResourceData(data resources.ResourceData) {
	t.DisplayName = data[DisplayNameKey].(string)
	t.Description = data[DescriptionKey].(string)
	t.AccountID = data[AccountIDKey].(string)
	t.PrimaryKey = data[PrimaryKeyKey].(string)
	t.Schema = data[SchemaKey].(string)
	t.Table = data[TableKey].(string)
	t.SourceDefinition = data[SourceDefinitionKey].(string)
	t.Enabled = data[EnabledKey].(bool)
}

// DiffUpstream reports whether the local resource differs from what the API
// holds. schema/table/primary_key are compared even though a change to them is
// arguably a different source: the API accepts the update, so the CLI forwards
// it rather than inventing a replace the backend does not require.
func (t *TableResource) DiffUpstream(upstream *TableResource) bool {
	if t.DisplayName != upstream.DisplayName {
		return true
	}
	if t.Description != upstream.Description {
		return true
	}
	if t.AccountID != upstream.AccountID {
		return true
	}
	if t.PrimaryKey != upstream.PrimaryKey {
		return true
	}
	if t.Schema != upstream.Schema {
		return true
	}
	if t.Enabled != upstream.Enabled {
		return true
	}
	return t.Table != upstream.Table
}
