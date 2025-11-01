package eventstream

import (
	esClient "github.com/rudderlabs/rudder-iac/api/client/event-stream"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	sourceHandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
)

const importDir = "event-stream"

type Provider struct {
	*provider.BaseProvider
}

func New(client esClient.EventStreamStore) *Provider {
	p := &Provider{
		BaseProvider: provider.NewBaseProvider(
			"event-stream",
			map[string]provider.Handler{
				sourceHandler.ResourceType: sourceHandler.NewHandler(client, importDir),
			}, map[string]string{
				"event-stream-source": sourceHandler.ResourceType,
			},
		),
	}
	return p
}
