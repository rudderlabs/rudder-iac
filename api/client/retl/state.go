package retl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// ReadState retrieves the complete RETL state
func (r *RudderRETLStore) ReadState(ctx context.Context) (*State, error) {
	data, err := r.client.Do(ctx, "GET", "/v2/cli/retl/state", nil)
	if err != nil {
		return nil, fmt.Errorf("sending read state request: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &state, nil
}

// PutResourceState saves a resource state record
func (r *RudderRETLStore) PutResourceState(ctx context.Context, id string, req PutStateRequest) error {
	if id == "" {
		return fmt.Errorf("resource ID cannot be empty")
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshalling PUT request: %w", err)
	}

	path := fmt.Sprintf("/v2/cli/retl/retl-sources/%s/state", id)
	_, err = r.client.Do(ctx, "PUT", path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("sending put state request: %w", err)
	}

	return nil
}
