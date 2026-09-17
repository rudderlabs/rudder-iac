// Package sourcekeys holds the vocabulary every rETL source kind shares.
//
// A rETL source is one row upstream whatever its kind: sql-model, table, and
// (next) audience all carry the same identity, account, warehouse and enabled
// fields, and differ only in how their payload is described — a SQL string, a
// schema/table pair, a bucket, a filter spec. Connections reference any of them
// through the same graph keys.
//
// Those shared names used to live in the sqlmodel package because it was the
// only kind. Every later kind then imported sqlmodel for constants that are not
// about SQL models, and connection/sourcekinds.go had to alias them to keep a
// rename from silently breaking. This package is that shared middle, so a new
// kind depends on the vocabulary rather than on its oldest sibling.
package sourcekeys

// Graph data keys. Every source handler publishes these, so connection
// validation and the shared rules can read a source without knowing its kind.
//
// PrimaryKeyKey is present for every kind and empty where the kind has no
// primary key — file-based sources among them — so a consumer reads one shape.
const (
	IDKey               = "id"
	LocalIDKey          = "local_id"
	DisplayNameKey      = "display_name"
	AccountIDKey        = "account_id"
	SourceDefinitionKey = "source_definition"
	PrimaryKeyKey       = "primary_key"
	EnabledKey          = "enabled"
	SourceTypeKey       = "source_type"
	CreatedAtKey        = "createdAt"
	UpdatedAtKey        = "updatedAt"
)

// Definition is the warehouse (or bucket) a source reads from, named as the
// control plane names it in sourceDefinitionName.
type Definition string

const (
	DefinitionPostgres   Definition = "postgres"
	DefinitionRedshift   Definition = "redshift"
	DefinitionSnowflake  Definition = "snowflake"
	DefinitionBigQuery   Definition = "bigquery"
	DefinitionMySQL      Definition = "mysql"
	DefinitionDatabricks Definition = "databricks"
	DefinitionTrino      Definition = "trino"
	DefinitionS3         Definition = "s3"
)

// WarehouseDefinitions are the query-backed definitions: every kind that runs
// SQL accepts exactly these. s3 is deliberately absent — it is file-backed, so
// it carries a bucket instead of a schema and table, and no primary key.
var WarehouseDefinitions = []Definition{
	DefinitionPostgres,
	DefinitionRedshift,
	DefinitionSnowflake,
	DefinitionBigQuery,
	DefinitionMySQL,
	DefinitionDatabricks,
	DefinitionTrino,
}

// TableDefinitions are the definitions a table source accepts: the warehouses
// plus s3.
var TableDefinitions = append(append([]Definition{}, WarehouseDefinitions...), DefinitionS3)

// OneOf renders definitions for a validate:"oneof=..." struct tag. Struct tags
// cannot interpolate a constant, so a kind's tag is still written by hand — but
// its test can assert the tag against this, which is what keeps the two from
// drifting.
func OneOf(definitions []Definition) string {
	out := make([]byte, 0, len(definitions)*10)
	for i, d := range definitions {
		if i > 0 {
			out = append(out, ' ')
		}
		out = append(out, d...)
	}
	return string(out)
}
