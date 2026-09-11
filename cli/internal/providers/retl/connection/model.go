package connection

import "github.com/rudderlabs/rudder-iac/cli/internal/resources"

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
)

// ConnectionsSpec mirrors the YAML spec structure: the body is a list of
// connection entries. JSON tags enable the typed rule engine's
// json.Marshal/Unmarshal round-trip and name the JSON Pointer references it
// reports; validate tags drive go-playground/validator checks.
type ConnectionsSpec struct {
	Connections []ConnectionSpec `json:"connections" mapstructure:"connections" validate:"required,dive"`
}

// ConnectionSpec is a single connection entry. Source and destination are
// scalar references to the two endpoints; everything else that is configurable
// lives under config. An omitted enabled means an enabled connection. Config
// needs no validate tag of its own — go-playground ignores required on a
// non-pointer struct and descends into it, so its required members reject an
// omitted config and name the fields it lacks.
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
	Object        *string           `json:"object,omitempty"        mapstructure:"object"`
	CursorColumn  string            `json:"cursor_column,omitempty" mapstructure:"cursor_column"`
	SyncSettings  *SyncSettingsSpec `json:"sync_settings,omitempty" mapstructure:"sync_settings"`
}

// ScheduleSpec is when the connection syncs. Which of the optional fields is
// required, and which is rejected, depends on Type.
type ScheduleSpec struct {
	Type           string `json:"type"                      mapstructure:"type"            validate:"required,oneof=basic cron manual"`
	EveryMinutes   *int   `json:"every_minutes,omitempty"   mapstructure:"every_minutes"`
	CronExpression string `json:"cron_expression,omitempty" mapstructure:"cron_expression"`
}

// EventSpec is the CDP event a JSON mapper connection emits. Name and
// NameColumn are mutually exclusive.
type EventSpec struct {
	Type       string `json:"type"                  mapstructure:"type" validate:"required,oneof=identify track"`
	Name       string `json:"name,omitempty"        mapstructure:"name"`
	NameColumn string `json:"name_column,omitempty" mapstructure:"name_column"`
}

// MappingSpec maps a source column to a destination field or identifier.
type MappingSpec struct {
	From string `json:"from" mapstructure:"from" validate:"required"`
	To   string `json:"to"   mapstructure:"to"   validate:"required"`
}

// ConstantSpec is a user-defined constant added to every synced record.
type ConstantSpec struct {
	Key   string `json:"key"   mapstructure:"key"   validate:"required"`
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
