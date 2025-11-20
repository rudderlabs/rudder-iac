package eventstream

import (
	esClient "github.com/rudderlabs/rudder-iac/api/client/event-stream"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	sourceHandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
)

const importDir = "event-stream"

func New(client esClient.EventStreamStore) *provider.BaseProvider {
	return provider.NewBaseProvider(
		"event-stream",
		[]provider.Handler{sourceHandler.NewHandler(client, importDir)},
	)
}
