package source

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
)

const prefix = "/v2/event-stream-sources"

type SourceStore interface {
	Create(ctx context.Context, source *CreateSourceRequest) (*EventStreamSource, error)
	Update(ctx context.Context, sourceId string, source *UpdateSourceRequest) (*EventStreamSource, error)
	Delete(ctx context.Context, sourceId string) error
	GetSources(ctx context.Context) ([]EventStreamSource, error)
}

type rudderSourceStore struct {
	client *client.Client
}

func NewRudderSourceStore(client *client.Client) SourceStore {
	store := &rudderSourceStore{
		client: client,
	}
	return store
}

func (r *rudderSourceStore) Create(ctx context.Context, source *CreateSourceRequest) (*EventStreamSource, error) {
	data, err := json.Marshal(source)
	if err != nil {
		return nil, fmt.Errorf("marshalling create source request: %w", err)
	}
	resp, err := r.client.Do(ctx, "POST", prefix, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating event stream source: %w", err)
	}
	var result EventStreamSource
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling create source response: %w", err)
	}
	return &result, nil
}

func (r *rudderSourceStore) Update(ctx context.Context, sourceId string, source *UpdateSourceRequest) (*EventStreamSource, error) {
	data, err := json.Marshal(source)
	if err != nil {
		return nil, fmt.Errorf("marshalling update source request: %w", err)
	}
	path := fmt.Sprintf("%s/%s", prefix, sourceId)
	resp, err := r.client.Do(ctx, "PUT", path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("updating event stream source: %w", err)
	}

	var result EventStreamSource
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling update source response: %w", err)
	}
	return &result, nil
}

func (r *rudderSourceStore) Delete(ctx context.Context, sourceId string) error {
	path := fmt.Sprintf("%s/%s", prefix, sourceId)
	_, err := r.client.Do(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("deleting event stream source: %w", err)
	}
	return nil
}

func (r *rudderSourceStore) GetSources(ctx context.Context) ([]EventStreamSource, error) {
	return client.GetAllResourcesWithPagination[EventStreamSource](ctx, r.client, prefix)
}
