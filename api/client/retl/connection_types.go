package retl

import (
	"encoding/json"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
)

// SyncBehaviour is how records are synced to the destination.
type SyncBehaviour string

const (
	SyncBehaviourUpsert SyncBehaviour = "upsert"
	SyncBehaviourMirror SyncBehaviour = "mirror"
	SyncBehaviourFull   SyncBehaviour = "full"
)

// ScheduleType is the schedule kind for a RETL connection.
type ScheduleType string

const (
	ScheduleTypeBasic  ScheduleType = "basic"
	ScheduleTypeManual ScheduleType = "manual"
	ScheduleTypeCron   ScheduleType = "cron"
)

// EventType is the CDP event type for JSON Mapper flows.
type EventType string

const (
	EventTypeIdentify EventType = "identify"
	EventTypeTrack    EventType = "track"
)

// Schedule defines when a RETL connection syncs.
type Schedule struct {
	Type         ScheduleType `json:"type"`
	EveryMinutes *int         `json:"everyMinutes,omitempty"`
}

// Event represents the CDP event configuration for a JSON Mapper flow.
// Name and NameColumn are mutually exclusive.
type Event struct {
	Type       EventType `json:"type"`
	Name       string    `json:"name,omitempty"`
	NameColumn string    `json:"nameColumn,omitempty"`
}

// Mapping maps a source column to a destination field or identifier.
type Mapping struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Constant represents a user-defined constant added to every synced record.
// Only applicable for JSON Mapper flows.
type Constant struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// SyncLogsConfig controls retention of sync log snapshots.
type SyncLogsConfig struct {
	Enabled            *bool `json:"enabled,omitempty"`
	LogRetentionInDays *int  `json:"logRetentionInDays,omitempty"`
	SnapshotsToRetain  *int  `json:"snapshotsToRetain,omitempty"`
}

// FailedKeysConfig controls retry behaviour for failed keys.
type FailedKeysConfig struct {
	EnableFailedKeysRetry *bool `json:"enableFailedKeysRetry,omitempty"`
}

// SyncSettings bundles operational settings for a connection. When omitted on
// create, the backend applies defaults.
type SyncSettings struct {
	SyncLogsConfig   *SyncLogsConfig   `json:"syncLogsConfig,omitempty"`
	FailedKeysConfig *FailedKeysConfig `json:"failedKeysConfig,omitempty"`
}

// RETLConnection is the full connection resource returned by the API.
// DestinationConfig is json.RawMessage so callers can decode per destination.
type RETLConnection struct {
	ID                string          `json:"id"`
	SourceID          string          `json:"sourceId"`
	DestinationID     string          `json:"destinationId"`
	Enabled           bool            `json:"enabled"`
	ExternalID        string          `json:"externalId,omitempty"`
	Schedule          Schedule        `json:"schedule"`
	SyncSettings      *SyncSettings   `json:"syncSettings,omitempty"`
	SyncBehaviour     SyncBehaviour   `json:"syncBehaviour"`
	Identifiers       []Mapping       `json:"identifiers"`
	Mappings          []Mapping       `json:"mappings,omitempty"`
	Event             *Event          `json:"event,omitempty"`
	Constants         []Constant      `json:"constants,omitempty"`
	CursorColumn      string          `json:"cursorColumn,omitempty"`
	Object            string          `json:"object,omitempty"`
	DestinationConfig json.RawMessage `json:"destinationConfig,omitempty"`
	CreatedAt         *time.Time      `json:"createdAt,omitempty"`
	UpdatedAt         *time.Time      `json:"updatedAt,omitempty"`
}

// CreateRETLConnectionRequest is the request body for POST /v2/retl-connections.
// Flow detection (JSON Mapper / Object mapping / Destination-specific) happens
// server-side based on the destination definition — callers do not specify a flow.
type CreateRETLConnectionRequest struct {
	SourceID          string          `json:"sourceId"`
	DestinationID     string          `json:"destinationId"`
	Enabled           *bool           `json:"enabled,omitempty"`
	ExternalID        string          `json:"externalId,omitempty"`
	Schedule          Schedule        `json:"schedule"`
	SyncSettings      *SyncSettings   `json:"syncSettings,omitempty"`
	SyncBehaviour     SyncBehaviour   `json:"syncBehaviour"`
	Identifiers       []Mapping       `json:"identifiers"`
	Mappings          []Mapping       `json:"mappings,omitempty"`
	Event             *Event          `json:"event,omitempty"`
	Constants         []Constant      `json:"constants,omitempty"`
	CursorColumn      string          `json:"cursorColumn,omitempty"`
	Object            string          `json:"object,omitempty"`
	DestinationConfig json.RawMessage `json:"destinationConfig,omitempty"`
}

// UpdateRETLConnectionRequest is the request body for PUT /v2/retl-connections/:id.
// Only mutable fields are accepted; the API rejects immutable fields at the
// validation layer. Per-flow mutability of Identifiers/Constants/Mappings is
// enforced server-side based on the detected flow.
type UpdateRETLConnectionRequest struct {
	Enabled      *bool         `json:"enabled,omitempty"`
	Schedule     Schedule      `json:"schedule"`
	SyncSettings *SyncSettings `json:"syncSettings,omitempty"`
	// Pointer-to-slice so callers can distinguish "not provided" (nil) from
	// "explicitly empty" (pointer to empty slice) — needed to clear existing
	// values via update.
	Mappings    *[]Mapping  `json:"mappings,omitempty"`
	Constants   *[]Constant `json:"constants,omitempty"`
	Identifiers []Mapping   `json:"identifiers,omitempty"`
}

// ListRETLConnectionsRequest is the request for listing connections.
// Deviates from the ticket's positional signature to surface pagination —
// the API returns a paginated page, and callers need Page/PageSize to iterate.
type ListRETLConnectionsRequest struct {
	SourceID      string
	DestinationID string
	HasExternalID *bool
	Page          int
	PageSize      int
}

// SetRETLConnectionExternalIDRequest is the request for setting a connection's
// external ID.
type SetRETLConnectionExternalIDRequest struct {
	ID         string `json:"id,omitempty"`
	ExternalID string `json:"externalId"`
}

// RETLConnectionsPage is the paginated response from GET /v2/retl-connections.
type RETLConnectionsPage struct {
	Data   []RETLConnection `json:"data"`
	Paging client.Paging    `json:"paging"`
}
