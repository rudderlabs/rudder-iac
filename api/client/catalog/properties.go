package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type PropertyCreate struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Config      map[string]interface{} `json:"propConfig,omitempty"`
}

type Property struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	WorkspaceId string                 `json:"workspaceId"`
	Config      map[string]interface{} `json:"propConfig"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	CreatedBy   string                 `json:"createdBy"`
	UpdatedBy   string                 `json:"updatedBy"`
}

type PropertyStore interface {
	CreateProperty(ctx context.Context, input PropertyCreate) (*Property, error)
	UpdateProperty(ctx context.Context, id string, input *Property) (*Property, error)
	DeleteProperty(ctx context.Context, id string) error
}

func (c *RudderDataCatalog) DeleteProperty(ctx context.Context, id string) error {
	_, err := c.client.Do(ctx, "DELETE", fmt.Sprintf("catalog/properties/%s", id), nil)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	return nil
}

func (c *RudderDataCatalog) UpdateProperty(ctx context.Context, id string, new *Property) (*Property, error) {
	byt, err := json.Marshal(new)
	if err != nil {
		return nil, fmt.Errorf("marshalling input: %w", err)
	}

	resp, err := c.client.Do(ctx, "PUT", fmt.Sprintf("catalog/properties/%s", id), bytes.NewReader(byt))
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}

	var property Property
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&property); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &property, nil
}

func (c *RudderDataCatalog) CreateProperty(ctx context.Context, input PropertyCreate) (*Property, error) {
	byt, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshalling input: %w", err)
	}

	resp, err := c.client.Do(ctx, "POST", "catalog/properties", bytes.NewReader(byt))
	if err != nil {
		return nil, fmt.Errorf("executing http request: %w", err)
	}

	property := Property{} // Create a holder for response object
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&property); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &property, nil
}
