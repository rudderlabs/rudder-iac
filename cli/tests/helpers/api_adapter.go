package helpers

import (
	"context"
	"encoding/json"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
)

var _ UpstreamStateReader = &APIClientAdapter{}

// APIClientAdapter wraps the catalog.DataCatalog client
// and implements the UpstreamStateReader interface.
type APIClientAdapter struct {
	client catalog.DataCatalog
}

// NewAPIClientAdapter creates a new APIClientAdapter instance
func NewAPIClientAdapter(client catalog.DataCatalog) *APIClientAdapter {
	return &APIClientAdapter{
		client: client,
	}
}

func (a *APIClientAdapter) RawState(ctx context.Context) (map[string]any, error) {
	state, err := a.client.ReadState(ctx)
	if err != nil {
		return nil, err
	}

	stateBytes, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}

	var stateMap map[string]any
	if err := json.Unmarshal(stateBytes, &stateMap); err != nil {
		return nil, err
	}

	return stateMap, nil
}
