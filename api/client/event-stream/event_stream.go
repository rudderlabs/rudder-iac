package eventstream

import (
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/event-stream/connection"
	"github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
	trackingplanconnection "github.com/rudderlabs/rudder-iac/api/client/event-stream/tracking-plan-connection"
)

type EventStreamStore interface {
	source.SourceStore
	trackingplanconnection.TrackingPlanConnectionStore
	connection.ConnectionStore
}

type rudderEventStreamStore struct {
	source.SourceStore
	trackingplanconnection.TrackingPlanConnectionStore
	connection.ConnectionStore
}

func NewRudderEventStreamStore(client *client.Client) EventStreamStore {
	return &rudderEventStreamStore{
		SourceStore:                 source.NewRudderSourceStore(client),
		TrackingPlanConnectionStore: trackingplanconnection.NewRudderTrackingPlanConnectionStore(client),
		ConnectionStore:             connection.NewRudderConnectionStore(client),
	}
}
