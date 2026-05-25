package columnmetadata

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
)

func requireDataGraphFlag() error {
	cfg := config.GetConfig()
	if !cfg.ExperimentalFlags.DataGraph {
		return fmt.Errorf("data-graphs commands require the experimental flag 'dataGraph' to be enabled in your configuration")
	}
	return nil
}

func newDataGraphClient(c *client.Client) dgClient.DataGraphClient {
	return dgClient.NewRudderDataGraphClient(c)
}
