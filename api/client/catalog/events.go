package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type EventCreate struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	EventType   string  `json:"eventType"`
	CategoryId  *string `json:"categoryId"`
	ProjectId   string  `json:"projectId"`
}

type Event struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	EventType   string    `json:"eventType"`
	CategoryId  *string   `json:"categoryId"`
	WorkspaceId string    `json:"workspaceId"`
	ProjectId   string    `json:"projectId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type EventStore interface {
	CreateEvent(ctx context.Context, input EventCreate) (*Event, error)
	UpdateEvent(ctx context.Context, id string, input *Event) (*Event, error)
	DeleteEvent(ctx context.Context, id string) error
	GetEvent(ctx context.Context, id string) (*Event, error)
	GetEvents(ctx context.Context) ([]*Event, error)
}

func (c *RudderDataCatalog) DeleteEvent(ctx context.Context, id string) error {
	_, err := c.client.Do(ctx, "DELETE", fmt.Sprintf("v2/catalog/events/%s", id), nil)
	if err != nil {
		return fmt.Errorf("sending delete request: %w", err)
	}

	return nil
}

func (c *RudderDataCatalog) CreateEvent(ctx context.Context, input EventCreate) (*Event, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshalling input: %w", err)
	}

	resp, err := c.client.Do(ctx, "POST", "v2/catalog/events", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}

	var event Event
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&event); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &event, nil
}

func (c *RudderDataCatalog) UpdateEvent(ctx context.Context, id string, input *Event) (*Event, error) {
	byt, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshalling input: %w", err)
	}

	resp, err := c.client.Do(ctx, "PUT", fmt.Sprintf("v2/catalog/events/%s", id), bytes.NewReader(byt))
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}

	var event Event
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&event); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &event, nil
}

func (c *RudderDataCatalog) GetEvent(ctx context.Context, id string) (*Event, error) {
	resp, err := c.client.Do(ctx, "GET", fmt.Sprintf("v2/catalog/events/%s", id), nil)
	if err != nil {
		return nil, fmt.Errorf("sending get request: %w", err)
	}

	var event Event
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&event); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &event, nil
}

func (c *RudderDataCatalog) GetEvents(ctx context.Context) ([]*Event, error) {
	return getAllResourcesWithPagination[*Event](ctx, c.client, "v2/catalog/events")
}
