package source

import (
	"github.com/rudderlabs/rudder-iac/api/client"
)

type CreateSourceRequest struct {
	ExternalID string `json:"externalId"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Enabled    bool   `json:"enabled"`
}

type UpdateSourceRequest struct {
	Name    string `json:"name,omitempty"`
	Enabled bool   `json:"enabled"`
}

type CreateUpdateSourceResponse struct {
	ID         string `json:"id"`
	ExternalID string `json:"externalId"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Enabled    bool   `json:"enabled"`
}

type EventStreamSource struct {
	ID           string        `json:"id"`
	ExternalID   string        `json:"externalId"`
	Name         string        `json:"name"`
	Type         string        `json:"type"`
	Enabled      bool          `json:"enabled"`
	TrackingPlan *TrackingPlan `json:"trackingPlan"`
}

// The response shape is different for the GET API compared
// to the APIs present in the tracking-plan-connection package
type TrackingPlanConfig struct {
	Track    *TrackConfig     `json:"track,omitempty"`
	Identify *EventTypeConfig `json:"identify,omitempty"`
	Group    *EventTypeConfig `json:"group,omitempty"`
	Page     *EventTypeConfig `json:"page,omitempty"`
	Screen   *EventTypeConfig `json:"screen,omitempty"`
}

type TrackConfig struct {
	*EventTypeConfig
	DropUnplannedEvents *bool `json:"dropUnplannedEvents,omitempty"`
}

type EventTypeConfig struct {
	PropagateViolations *bool   `json:"propagateViolations,omitempty"`
	DropUnplannedProperties       *bool `json:"dropUnplannedProperties,omitempty"`
	DropOtherViolations         *bool `json:"dropOtherViolations,omitempty"`
}


type TrackingPlan struct {
	ID     string                               `json:"id"`
	Config *TrackingPlanConfig `json:"config"`
}

type eventStreamSourcesPage struct {
	client.APIPage
	Sources []EventStreamSource `json:"data"`
}
