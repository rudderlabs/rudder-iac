package trackingplanconnection

import "context"

type TrackingPlanConnectionStore interface {
	LinkTP(ctx context.Context, trackingPlanId string, sourceId string, config *ConnectionConfig) error
	UpdateTPConnection(ctx context.Context, trackingPlanId string, sourceId string, config *ConnectionConfig) error
	UnlinkTP(ctx context.Context, trackingPlanId string, sourceId string) error
}

type Action string

type StringBool string

const (
	Forward Action = "forward"
	Drop    Action = "drop"
)

const (
	True  StringBool = "true"
	False StringBool = "false"
)

type ConnectionConfig struct {
	Track    *TrackConfig     `json:"track,omitempty"`
	Identify *EventTypeConfig `json:"identify,omitempty"`
	Group    *EventTypeConfig `json:"group,omitempty"`
	Page     *EventTypeConfig `json:"page,omitempty"`
	Screen   *EventTypeConfig `json:"screen,omitempty"`
}

type EventTypeConfig struct {
	PropagateValidationErrors *StringBool   `json:"propagateValidationErrors,omitempty"`
	UnplannedProperties       *Action `json:"unplannedProperties,omitempty"`
	AnyOtherViolation         *Action `json:"anyOtherViolation,omitempty"`
}

type TrackConfig struct {
	*EventTypeConfig
	AllowUnplannedEvents *StringBool `json:"allowUnplannedEvents,omitempty"`
}

type requestBody struct {
	Config *ConnectionConfig `json:"config"`
}
