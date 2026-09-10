package connections

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

const (
	ResourceType = "retl-connection"
	ResourceKind = "retl-connections"
	MetadataName = "retl-connections"
	ImportPath   = "connections.yaml"

	ConnectionsKey       = "connections"
	SourceKey            = "source"
	DestinationKey       = "destination"
	EnabledKey           = "enabled"
	SyncBehaviourKey     = "sync_behaviour"
	CursorColumnKey      = "cursor_column"
	ObjectKey            = "object"
	ScheduleKey          = "schedule"
	EventKey             = "event"
	IdentifiersKey       = "identifiers"
	MappingsKey          = "mappings"
	ConstantsKey         = "constants"
	SyncSettingsKey      = "sync_settings"
	DestinationConfigKey = "destination_config"

	// Output-side keys held in state.
	IDKey            = "id"
	SourceIDKey      = "sourceId"
	DestinationIDKey = "destinationId"
	ExternalIDKey    = "externalId"
)

// ConnectionsSpec mirrors the YAML: the body is a list, following
// event-stream-connections. A warehouse-activation project usually declares
// many connections over few sources, so the plural form keeps them reviewable
// in one file.
type ConnectionsSpec struct {
	Connections []ConnectionSpec `json:"connections" mapstructure:"connections" validate:"required,dive"`
}

// ConnectionSpec is one connection entry.
//
// Flow is not a field: the backend derives it from the destination definition
// (JSON Mapper / Object Mapping / destination-specific). The spec expresses the
// distinction structurally — `event` for JSON Mapper, `object` for Object
// Mapping — exactly as the Terraform provider does.
type ConnectionSpec struct {
	LocalID       string            `json:"id"             mapstructure:"id"             validate:"required"`
	Source        string            `json:"source"         mapstructure:"source"         validate:"required"`
	Destination   string            `json:"destination"    mapstructure:"destination"    validate:"required"`
	Enabled       *bool             `json:"enabled"        mapstructure:"enabled"`
	SyncBehaviour string            `json:"sync_behaviour" mapstructure:"sync_behaviour" validate:"required,oneof=upsert mirror full"`
	CursorColumn  string            `json:"cursor_column"  mapstructure:"cursor_column"`
	Object        string            `json:"object"         mapstructure:"object"`
	Schedule      ScheduleSpec      `json:"schedule"       mapstructure:"schedule"       validate:"required"`
	Event         *EventSpec        `json:"event"          mapstructure:"event"`
	Identifiers   []MappingSpec     `json:"identifiers"    mapstructure:"identifiers"    validate:"required,min=1,dive"`
	Mappings      []MappingSpec     `json:"mappings"       mapstructure:"mappings"       validate:"dive"`
	Constants     []ConstantSpec    `json:"constants"      mapstructure:"constants"      validate:"dive"`
	SyncSettings  *SyncSettingsSpec `json:"sync_settings"  mapstructure:"sync_settings"`
	// DestinationConfig carries destination-specific flow fields. Schema-less at
	// spec level by design — the destination registry validates it. Customer.io
	// requires "object" in here rather than top-level, which the API enforces
	// with "destinationConfig: missing required property 'object'".
	DestinationConfig map[string]any `json:"destination_config" mapstructure:"destination_config"`
}

type ScheduleSpec struct {
	Type           string `json:"type"            mapstructure:"type"            validate:"required,oneof=basic cron manual"`
	EveryMinutes   *int   `json:"every_minutes"   mapstructure:"every_minutes"`
	CronExpression string `json:"cron_expression" mapstructure:"cron_expression"`
}

// EventSpec applies to JSON Mapper flows only. Name and NameColumn are mutually
// exclusive — a fixed event name, or one read per row from a column.
type EventSpec struct {
	Type       string `json:"type"        mapstructure:"type"        validate:"required,oneof=identify track"`
	Name       string `json:"name"        mapstructure:"name"`
	NameColumn string `json:"name_column" mapstructure:"name_column"`
}

type MappingSpec struct {
	From string `json:"from" mapstructure:"from" validate:"required"`
	To   string `json:"to"   mapstructure:"to"   validate:"required"`
}

type ConstantSpec struct {
	Key   string `json:"key"   mapstructure:"key"   validate:"required"`
	Value string `json:"value" mapstructure:"value" validate:"required"`
}

// SyncSettingsSpec and its members are pointers throughout so an omitted
// section serialises away rather than being sent as a zero value. That is the
// CLI half of the merge-semantics discipline: a declarative client must send
// only what the user declared, or it will reset config it never mentioned.
type SyncSettingsSpec struct {
	SyncLogs   *SyncLogsSpec   `json:"sync_logs"   mapstructure:"sync_logs"`
	FailedKeys *FailedKeysSpec `json:"failed_keys" mapstructure:"failed_keys"`
}

type SyncLogsSpec struct {
	Enabled           *bool `json:"enabled"             mapstructure:"enabled"`
	RetentionDays     *int  `json:"retention_days"      mapstructure:"retention_days"`
	SnapshotsToRetain *int  `json:"snapshots_to_retain" mapstructure:"snapshots_to_retain"`
}

type FailedKeysSpec struct {
	Retry *bool `json:"retry" mapstructure:"retry"`
}

// connectionResource is the graph-side representation. The two endpoint
// PropertyRefs give the resource graph its dependency edges; the syncer
// dereferences them to remote ids before calling the lifecycle.
type connectionResource struct {
	LocalID           string
	Source            *resources.PropertyRef
	Destination       *resources.PropertyRef
	Enabled           bool
	SyncBehaviour     string
	CursorColumn      string
	Object            string
	Schedule          ScheduleSpec
	Event             *EventSpec
	Identifiers       []MappingSpec
	Mappings          []MappingSpec
	Constants         []ConstantSpec
	SyncSettings      *SyncSettingsSpec
	DestinationConfig map[string]any
}

type ImportResourceInfo struct {
	WorkspaceId string
	RemoteId    string
}

var importMetadata = map[string]*ImportResourceInfo{}
