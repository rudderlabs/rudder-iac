package connection

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

const (
	SourceKey      = "source"
	DestinationKey = "destination"
	EnabledKey     = "enabled"

	// Output-side keys: the remote identifiers the lifecycle stores in state.
	// SourceIDKey/DestinationIDKey hold the resolved endpoint ids so Update can
	// detect an endpoint change (a replacement) without re-dereferencing refs.
	IDKey            = "id"
	SourceIDKey      = "sourceId"
	DestinationIDKey = "destinationId"

	EventStreamConnectionResourceType = "event-stream-connection"
	EventStreamConnectionResourceKind = "event-stream-connections"
)

// ConnectionsSpec mirrors the YAML spec structure: the body is a list of
// connection entries. JSON tags enable the typed rule engine's
// json.Marshal/Unmarshal round-trip; validate tags drive
// go-playground/validator checks.
type ConnectionsSpec struct {
	Connections []ConnectionSpec `json:"connections" mapstructure:"connections" validate:"required,dive"`
}

// ConnectionSpec is a single connection entry. There is intentionally no
// config field — event stream connections are pure links; connection settings live on the
// destination spec. Unknown keys (config included) fail at decode time via
// strict decoding.
type ConnectionSpec struct {
	LocalID     string `json:"id"          mapstructure:"id"          validate:"required"`
	Source      string `json:"source"      mapstructure:"source"      validate:"required"`
	Destination string `json:"destination" mapstructure:"destination" validate:"required"`
	Enabled     *bool  `json:"enabled"     mapstructure:"enabled"`
}

// connectionResource is the graph-side representation of one connection: the
// two endpoint references plus the resolved enabled flag. The PropertyRefs
// give the resource graph its dependency edges on both endpoints.
type connectionResource struct {
	LocalID     string
	Source      *resources.PropertyRef
	Destination *resources.PropertyRef
	Enabled     bool
}
