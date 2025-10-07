package source

import (
	"github.com/rudderlabs/rudder-iac/api/client"
	trackingplanClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/tracking-plan-connection"
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

type TrackingPlan struct {
	ID     string                               `json:"id"`
	Config *trackingplanClient.ConnectionConfig `json:"config"`
}

type eventStreamSourcesPage struct {
	client.APIPage
	Sources []EventStreamSource `json:"data"`
}
