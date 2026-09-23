package connection

import "slices"

// WarehouseMetadata is what the CLI knows about one rETL source warehouse: the
// sync behaviours it accepts, and whether it can back a SQL model source or
// carry sync settings.
type WarehouseMetadata struct {
	SyncBehaviours       []string
	SupportsSQLModel     bool
	SupportsSyncSettings bool
}

// sqlWarehouse is the shape shared by the warehouses that run SQL.
var sqlWarehouse = WarehouseMetadata{
	SyncBehaviours:       []string{"upsert", "mirror", "full"},
	SupportsSQLModel:     true,
	SupportsSyncSettings: true,
}

// warehouseMetadataBySourceDefinition mirrors the options block of
// rudder-integrations-config src/configurations/sources/<name>/db-config.json
// at revision 9fa7b26, which is what the backend reads in
// src/modules/retl/api-gateway/connection-config (constants.ts, assembler.ts,
// connection-constraints.ts).
var warehouseMetadataBySourceDefinition = map[string]WarehouseMetadata{
	"bigquery":   sqlWarehouse,
	"databricks": sqlWarehouse,
	"mysql":      sqlWarehouse,
	"postgres":   sqlWarehouse,
	"redshift":   sqlWarehouse,
	"snowflake":  sqlWarehouse,
	"trino":      sqlWarehouse,

	"s3":   {SyncBehaviours: []string{"upsert"}},
	"sftp": {SyncBehaviours: []string{"upsert"}},

	// clickhouse declares no syncBehaviours upstream, so the backend default
	// applies; it supports neither SQL models nor sync settings.
	"clickhouse": {SyncBehaviours: []string{"upsert", "mirror", "full"}},
}

// SourceWarehouseMetadata returns the metadata for a source definition name. A
// definition missing from the table is unknown, not unsupported: callers must
// skip the checks that need this metadata rather than guess support, because a
// guess turns a backend rejection into a locally passing validation.
func SourceWarehouseMetadata(sourceDefinition string) (WarehouseMetadata, bool) {
	metadata, ok := warehouseMetadataBySourceDefinition[sourceDefinition]
	// The SQL warehouses share one backing array, so hand out a copy: a caller
	// that sorts or appends in place would otherwise rewrite the table for
	// every warehouse at once.
	metadata.SyncBehaviours = slices.Clone(metadata.SyncBehaviours)
	return metadata, ok
}
