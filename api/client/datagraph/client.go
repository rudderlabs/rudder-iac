package datagraph

import "github.com/rudderlabs/rudder-iac/api/client"

const (
	dataGraphsBasePath = "/v2/data-graphs"
)

// DataGraphClient is the interface for Data Graph operations
type DataGraphClient interface {
	DataGraphStore
	ModelStore
}

// rudderDataGraphClient implements the DataGraphStore interface
type rudderDataGraphClient struct {
	client *client.Client
}

// NewRudderDataGraphClient creates a new DataGraphStore implementation
func NewRudderDataGraphClient(c *client.Client) DataGraphClient {
	return &rudderDataGraphClient{
		client: c,
	}
}
