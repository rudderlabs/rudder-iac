package eventstream

import (
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
)

type EventStreamStore interface {
	source.SourceStore
}

type rudderEventStreamStore struct {
	source.SourceStore
}

func NewRudderEventStreamStore(client *client.Client) EventStreamStore {
	return &rudderEventStreamStore{
		SourceStore: source.NewRudderSourceStore(client),
	}
}