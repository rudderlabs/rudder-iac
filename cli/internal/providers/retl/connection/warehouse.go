package connection

import (
	"slices"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
)

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

// AcceptedSyncBehaviours is the set of sync behaviours a connection between a
// source definition and a destination may declare: what both endpoints accept,
// intersected with what a spec can express on the given flow. It is the one
// derivation of that set — the V-R2 validation rule and the import eligibility
// check both read it, so a row import writes can never fail validate on this
// field, and widening the spec contract narrows the import skip by itself.
//
// An empty result means the two endpoints share no expressible behaviour, which
// callers report as such rather than falling back to a permissive set: that
// would let a connection the backend refuses pass. ok is false when the source
// definition's capabilities are unknown, in which case there is nothing to
// intersect and the caller has to skip the check rather than guess.
func AcceptedSyncBehaviours(sourceDefinition string, destinationAccepted []string, flow Flow) ([]string, bool) {
	warehouse, known := SourceWarehouseMetadata(sourceDefinition)
	if !known {
		return nil, false
	}
	// SourceWarehouseMetadata hands back a clone, so filtering in place is safe.
	return slices.DeleteFunc(warehouse.SyncBehaviours, func(behaviour string) bool {
		return !specSupportsSyncBehaviour(behaviour, flow) || !slices.Contains(destinationAccepted, behaviour)
	}), true
}

// specSupportsSyncBehaviour reports whether ConfigSpec.SyncBehaviour can carry
// the behaviour on the given flow. It is the Go side of that field's `oneof`
// tag, flow by flow: the tag is the union over the flows, so widening one has
// to widen the other.
//
// Mirroring needs the destination's field set declared up front, which is what
// the vdm-next destinations do through their own config. The JSON mapper and
// object mapping declare no such set, so a spec cannot carry mirror there
// however the two endpoints feel about it.
//
// On object mapping that is stricter than the backend, deliberately. The
// backend refuses mirror on the JSON mapper alone (FLOWS_DISALLOWING_MIRROR in
// config-backend retl/api-gateway/connection-config/assembler.ts), so an
// object-mapping connection is accepted and stored as mirror. DEX-1046 found
// that what looked like a silent downgrade to upsert was the webapp displaying
// it as Upsert, not the backend rewriting it. The sync still has no declared
// field set to mirror onto, so refusing it here is better than passing a
// connection that cannot do what its spec says.
func specSupportsSyncBehaviour(behaviour string, flow Flow) bool {
	if behaviour == string(retlClient.SyncBehaviourMirror) {
		return flow == FlowVisualMapperV2
	}
	return true
}
