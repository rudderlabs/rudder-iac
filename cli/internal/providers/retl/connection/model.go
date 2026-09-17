package connection

import (
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

const (
	// columnNamePattern is the grammar a warehouse column reference has to
	// follow: a letter or underscore, then word characters, "$" or "." — the
	// same shape the backend accepts for identifiers, mappings, the cursor
	// column and the event name column.
	columnNamePattern = `^[A-Za-z_][\w$.]*$`
	columnNameTag     = "retl_column_name"

	// objectNamePattern rejects an empty or whitespace-padded object: the
	// backend matches a declared object verbatim.
	objectNamePattern = `(?s)^\S(.*\S)?$`
	objectNameTag     = "retl_object_name"
)

// Registering here rather than in the rules package keeps the pattern available
// wherever the spec is validated: the tag lives on this package's types, so its
// registrar has to be reachable from this package alone.
func init() {
	funcs.NewPattern(columnNameTag, columnNamePattern, "must be a column name matching "+columnNamePattern)
	funcs.NewPattern(objectNameTag, objectNamePattern, "must not be empty or have leading or trailing whitespace")
}

const (
	ResourceType = "retl-connection"
	ResourceKind = "retl-connections"
	MetadataName = "retl-connections"

	// ImportPath is the spec file all importable connections are written to,
	// relative to the provider's import directory — export emits one spec of
	// the retl-connections kind per run.
	ImportPath = "connections.yaml"

	// ConnectionsKey names the spec body; the rest are graph input keys. Source
	// and destination hold PropertyRefs the syncer dereferences to remote ids
	// before the lifecycle runs, and config holds the canonical config map.
	ConnectionsKey = "connections"
	SourceKey      = "source"
	DestinationKey = "destination"
	EnabledKey     = "enabled"
	ConfigKey      = "config"

	// Output-side keys: the remote identifiers the lifecycle stores in state.
	// They intentionally repeat the event stream connection names so that the
	// project-wide connection topology scan, which reads event stream
	// connections only today, can fold rETL connections in without a second
	// set of keys.
	IDKey            = "id"
	SourceIDKey      = "sourceId"
	DestinationIDKey = "destinationId"

	// ExternalIDKey marks CLI-managed rows in List output.
	ExternalIDKey = "externalId"
)

// RemoteConnection carries a remote connection through import and export
// together with the identity the connection row itself does not hold: the
// workspace and the source kind come from the rETL source, the endpoints'
// display names are what an imported connection is named after, and the
// endpoints' externalIds — empty when an endpoint is not CLI-managed — are the
// endpoints' local resource ids. Both endpoint catalogs are read once per
// operation, so carrying the result here is what keeps naming, export and
// matching free of further API calls.
type RemoteConnection struct {
	retlClient.RETLConnection
	WorkspaceID           string
	SourceKind            SourceKind
	SourceName            string
	SourceExternalID      string
	DestinationName       string
	DestinationExternalID string
}

// ConnectionsSpec mirrors the YAML spec structure: the body is a list of
// connection entries. JSON tags enable the typed rule engine's
// json.Marshal/Unmarshal round-trip and name the JSON Pointer references it
// reports; validate tags drive go-playground/validator checks.
type ConnectionsSpec struct {
	Connections []ConnectionSpec `json:"connections" mapstructure:"connections" validate:"required,dive"`
}

// ConnectionSpec is a single connection entry. Source and destination are
// scalar references to the two endpoints; everything else that is configurable
// lives under config. An omitted enabled means an enabled connection.
type ConnectionSpec struct {
	LocalID     string     `json:"id"                mapstructure:"id"          validate:"required"`
	Source      string     `json:"source"            mapstructure:"source"      validate:"required"`
	Destination string     `json:"destination"       mapstructure:"destination" validate:"required"`
	Enabled     *bool      `json:"enabled,omitempty" mapstructure:"enabled"`
	Config      ConfigSpec `json:"config"            mapstructure:"config"`
}

// ConfigSpec is the per-connection rETL configuration. There is no flow field:
// the backend derives the flow from the destination and the presence of object,
// and ClassifyFlow mirrors that. No raw map absorbs unknown keys, so a misspelt
// field is an error rather than a silently dropped setting. Object is a pointer
// so validation can tell an omitted object (a JSON mapper connection) from an
// explicitly empty one, which is a spec error.
type ConfigSpec struct {
	SyncBehaviour string            `json:"sync_behaviour"          mapstructure:"sync_behaviour" validate:"required,oneof=upsert mirror full"`
	Schedule      ScheduleSpec      `json:"schedule"                mapstructure:"schedule"`
	Identifiers   []MappingSpec     `json:"identifiers"             mapstructure:"identifiers"    validate:"required,min=1,dive"`
	Mappings      []MappingSpec     `json:"mappings,omitempty"      mapstructure:"mappings"       validate:"dive"`
	Constants     []ConstantSpec    `json:"constants,omitempty"     mapstructure:"constants"      validate:"dive"`
	Event         *EventSpec        `json:"event,omitempty"         mapstructure:"event"`
	Object        *string           `json:"object,omitempty"        mapstructure:"object"         validate:"omitempty,pattern=retl_object_name"`
	CursorColumn  string            `json:"cursor_column,omitempty" mapstructure:"cursor_column"  validate:"excluded_unless=SyncBehaviour upsert,omitempty,pattern=retl_column_name"`
	SyncSettings  *SyncSettingsSpec `json:"sync_settings,omitempty" mapstructure:"sync_settings"`
}

// ScheduleSpec is when the connection syncs. Which of the optional fields is
// required, and which is rejected, depends on Type. The five-minute floor on
// every_minutes is the same one the cron rule applies to cron expressions
// through minGapMinutes (retl/rules/connection/cron.go).
type ScheduleSpec struct {
	Type           string `json:"type"                      mapstructure:"type"            validate:"required,oneof=basic cron manual"`
	EveryMinutes   *int   `json:"every_minutes,omitempty"   mapstructure:"every_minutes"   validate:"required_if=Type basic,excluded_if=Type cron,excluded_if=Type manual,omitempty,gte=5"`
	CronExpression string `json:"cron_expression,omitempty" mapstructure:"cron_expression" validate:"required_if=Type cron,excluded_if=Type basic,excluded_if=Type manual"`
}

// EventSpec is the CDP event a JSON mapper connection emits. Name and
// NameColumn are mutually exclusive.
type EventSpec struct {
	Type       string `json:"type"                  mapstructure:"type"        validate:"required,oneof=identify track"`
	Name       string `json:"name,omitempty"        mapstructure:"name"        validate:"excluded_with=NameColumn"`
	NameColumn string `json:"name_column,omitempty" mapstructure:"name_column" validate:"omitempty,pattern=retl_column_name"`
}

// MappingSpec maps a source column to a destination field or identifier.
type MappingSpec struct {
	From string `json:"from" mapstructure:"from" validate:"required,pattern=retl_column_name"`
	To   string `json:"to"   mapstructure:"to"   validate:"required"`
}

// ConstantSpec is a user-defined constant added to every synced record. The
// backend writes context.mappedToDestination itself
// (config-backend src/modules/retl/api-gateway/connection-config/constants.ts),
// so a user constant claiming that key would be overwritten rather than
// delivered.
type ConstantSpec struct {
	Key   string `json:"key"   mapstructure:"key"   validate:"required,ne=context.mappedToDestination"`
	Value string `json:"value" mapstructure:"value" validate:"required"`
}

// SyncSettingsSpec bundles the operational settings of a connection. Every
// field is optional: what the spec omits keeps the backend default.
type SyncSettingsSpec struct {
	SyncLogs   *SyncLogsSpec   `json:"sync_logs,omitempty"   mapstructure:"sync_logs"`
	FailedKeys *FailedKeysSpec `json:"failed_keys,omitempty" mapstructure:"failed_keys"`
}

// SyncLogsSpec controls retention of sync log snapshots. The pointers keep an
// explicit false or zero distinguishable from an omitted field.
type SyncLogsSpec struct {
	Enabled           *bool `json:"enabled,omitempty"             mapstructure:"enabled"`
	RetentionDays     *int  `json:"retention_days,omitempty"      mapstructure:"retention_days"`
	SnapshotsToRetain *int  `json:"snapshots_to_retain,omitempty" mapstructure:"snapshots_to_retain"`
}

// FailedKeysSpec controls retry behaviour for failed keys.
type FailedKeysSpec struct {
	Retry *bool `json:"retry,omitempty" mapstructure:"retry"`
}

// connectionResource is the graph-side representation of one connection: the
// two endpoint references, the resolved enabled flag and the canonical config
// the converters produce. The PropertyRefs give the resource graph its
// dependency edges on both endpoints.
type connectionResource struct {
	LocalID        string
	Source         *resources.PropertyRef
	Destination    *resources.PropertyRef
	Enabled        bool
	Config         map[string]any
	ImportMetadata map[string]*WorkspaceRemoteIDMapping
}

// WorkspaceRemoteIDMapping is one connection's import identity. Each
// connection owns its mapping — it hangs off connectionResource rather than a
// package-level map — so a second loaded project cannot inherit the first
// one's metadata.
type WorkspaceRemoteIDMapping struct {
	WorkspaceID string
	RemoteID    string
}
