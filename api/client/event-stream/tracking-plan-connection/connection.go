package trackingplanconnection

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
)

const prefix = "/v2/catalog"

type rudderTrackingPlanConnectionStore struct {
	client *client.Client
}

func NewRudderTrackingPlanConnectionStore(client *client.Client) TrackingPlanConnectionStore {
	return &rudderTrackingPlanConnectionStore{
		client: client,
	}
}

func (r *rudderTrackingPlanConnectionStore) LinkTP(ctx context.Context, trackingPlanId string, sourceId string, config *ConnectionConfig) error {
	req := &requestBody{
		Config: config,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshalling connection config: %w", err)
	}

	path := fmt.Sprintf("%s/tracking-plans/%s/sources/%s", prefix, trackingPlanId, sourceId)
	_, err = r.client.Do(ctx, "POST", path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("linking tracking plan to source: %w", err)
	}
	return nil
}

func (r *rudderTrackingPlanConnectionStore) UpdateTPConnection(ctx context.Context, trackingPlanId string, sourceId string, config *ConnectionConfig) error {
	req := &requestBody{
		Config: config,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshalling connection config: %w", err)
	}

	path := fmt.Sprintf("%s/tracking-plans/%s/sources/%s", prefix, trackingPlanId, sourceId)
	_, err = r.client.Do(ctx, "PUT", path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("updating tracking plan connection: %w", err)
	}
	return nil
}

func (r *rudderTrackingPlanConnectionStore) UnlinkTP(ctx context.Context, trackingPlanId string, sourceId string) error {
	path := fmt.Sprintf("%s/tracking-plans/%s/sources/%s", prefix, trackingPlanId, sourceId)
	_, err := r.client.Do(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("unlinking tracking plan from source: %w", err)
	}
	return nil
}