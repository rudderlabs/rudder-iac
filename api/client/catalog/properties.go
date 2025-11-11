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
	ExternalId  string                 `json:"externalId"`
}

type PropertyUpdate struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Config      map[string]interface{} `json:"propConfig"`
}

type Property struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Type             string                 `json:"type"`
	WorkspaceId      string                 `json:"workspaceId"`
	DefinitionId     string                 `json:"definitionId"`
	ItemDefinitionId string                 `json:"itemDefinitionId"`
	ExternalID       string                 `json:"externalId,omitempty"`
	Config           map[string]interface{} `json:"propConfig"`
	CreatedAt        time.Time              `json:"createdAt"`
	UpdatedAt        time.Time              `json:"updatedAt"`
	CreatedBy        string                 `json:"createdBy"`
	UpdatedBy        string                 `json:"updatedBy"`
}

type PropertyStore interface {
	CreateProperty(ctx context.Context, input PropertyCreate) (*Property, error)
	UpdateProperty(ctx context.Context, id string, input *PropertyUpdate) (*Property, error)
	DeleteProperty(ctx context.Context, id string) error
	GetProperty(ctx context.Context, id string) (*Property, error)
	GetProperties(ctx context.Context, options ListOptions) ([]*Property, error)
	SetPropertyExternalId(ctx context.Context, id string, externalId string) error
}

func (c *RudderDataCatalog) DeleteProperty(ctx context.Context, id string) error {
	_, err := c.client.Do(ctx, "DELETE", fmt.Sprintf("v2/catalog/properties/%s", id), nil)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	return nil
}

func (c *RudderDataCatalog) UpdateProperty(ctx context.Context, id string, new *PropertyUpdate) (*Property, error) {
	byt, err := json.Marshal(new)
	if err != nil {
		return nil, fmt.Errorf("marshalling input: %w", err)
	}

	resp, err := c.client.Do(ctx, "PUT", fmt.Sprintf("v2/catalog/properties/%s", id), bytes.NewReader(byt))
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

	resp, err := c.client.Do(ctx, "POST", "v2/catalog/properties", bytes.NewReader(byt))
	if err != nil {
		return nil, fmt.Errorf("executing http request: %w", err)
	}

	property := Property{} // Create a holder for response object
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&property); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &property, nil
}

func (c *RudderDataCatalog) GetProperty(ctx context.Context, id string) (*Property, error) {
	resp, err := c.client.Do(ctx, "GET", fmt.Sprintf("v2/catalog/properties/%s", id), nil)
	if err != nil {
		return nil, fmt.Errorf("sending get request: %w", err)
	}

	var property Property
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&property); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &property, nil
}

func (c *RudderDataCatalog) GetProperties(ctx context.Context, options ListOptions) ([]*Property, error) {
	return getAllResourcesPaginated[*Property](ctx, c.client, fmt.Sprintf("v2/catalog/properties%s", options.ToQuery()))
}

func (c *RudderDataCatalog) SetPropertyExternalId(ctx context.Context, id string, externalId string) error {
	payload := map[string]string{
		"externalId": externalId,
	}

	byt, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshalling payload: %w", err)
	}

	_, err = c.client.Do(ctx, "PUT", fmt.Sprintf("v2/catalog/properties/%s/external-id", id), bytes.NewReader(byt))
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}

	return nil
}
