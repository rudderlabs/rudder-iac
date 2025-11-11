package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type CustomTypeCreate struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Config      map[string]interface{} `json:"config"`
	Properties  []CustomTypeProperty   `json:"properties,omitempty"`
	Variants    Variants               `json:"variants,omitempty"`
	ExternalId  string                 `json:"externalId"`
}

type CustomTypeUpdate struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Config      map[string]interface{} `json:"config"`
	Properties  []CustomTypeProperty   `json:"properties,omitempty"`
	Variants    Variants               `json:"variants,omitempty"`
}

type CustomType struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Version         int                    `json:"version"`
	Description     string                 `json:"description"`
	Type            string                 `json:"type"`
	DataType        string                 `json:"dataType"`
	WorkspaceId     string                 `json:"workspaceId"`
	ExternalID      string                 `json:"externalId,omitempty"`
	Config          map[string]interface{} `json:"config"`
	Rules           map[string]interface{} `json:"rules"`
	Properties      []CustomTypeProperty   `json:"properties"`
	ItemDefinitions []any                  `json:"itemDefinitions"`
	Variants        Variants               `json:"variants,omitempty"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
	CreatedBy       string                 `json:"createdBy"`
	UpdatedBy       string                 `json:"updatedBy"`
}

type CustomTypeProperty struct {
	ID       string `json:"id"`
	Required bool   `json:"required"`
}

type CustomTypeStore interface {
	CreateCustomType(ctx context.Context, input CustomTypeCreate) (*CustomType, error)
	UpdateCustomType(ctx context.Context, id string, input *CustomTypeUpdate) (*CustomType, error)
	DeleteCustomType(ctx context.Context, id string) error
	GetCustomType(ctx context.Context, id string) (*CustomType, error)
	GetCustomTypes(ctx context.Context, options ListOptions) ([]*CustomType, error)
	SetCustomTypeExternalId(ctx context.Context, id string, externalId string) error
}

func (c *RudderDataCatalog) DeleteCustomType(ctx context.Context, id string) error {
	_, err := c.client.Do(ctx, "DELETE", fmt.Sprintf("v2/catalog/custom-types/%s", id), nil)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	return nil
}

func (c *RudderDataCatalog) UpdateCustomType(ctx context.Context, id string, new *CustomTypeUpdate) (*CustomType, error) {
	byt, err := json.Marshal(new)
	if err != nil {
		return nil, fmt.Errorf("marshalling input: %w", err)
	}

	resp, err := c.client.Do(ctx, "PUT", fmt.Sprintf("v2/catalog/custom-types/%s", id), bytes.NewReader(byt))
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}

	var customType CustomType
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&customType); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &customType, nil
}

func (c *RudderDataCatalog) CreateCustomType(ctx context.Context, input CustomTypeCreate) (*CustomType, error) {
	byt, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshalling input: %w", err)
	}

	resp, err := c.client.Do(ctx, "POST", "v2/catalog/custom-types", bytes.NewReader(byt))
	if err != nil {
		return nil, fmt.Errorf("executing http request: %w", err)
	}

	customType := CustomType{} // Create a holder for response object
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&customType); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &customType, nil
}

func (c *RudderDataCatalog) GetCustomType(ctx context.Context, id string) (*CustomType, error) {
	resp, err := c.client.Do(ctx, "GET", fmt.Sprintf("v2/catalog/custom-types/%s", id), nil)
	if err != nil {
		return nil, fmt.Errorf("sending get request: %w", err)
	}

	var customType CustomType
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&customType); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &customType, nil
}

func (c *RudderDataCatalog) GetCustomTypes(ctx context.Context, options ListOptions) ([]*CustomType, error) {
	return getAllResourcesPaginated[*CustomType](ctx, c.client, fmt.Sprintf("v2/catalog/custom-types%s", options.ToQuery()))
}

func (c *RudderDataCatalog) SetCustomTypeExternalId(ctx context.Context, id string, externalId string) error {
	payload := map[string]string{
		"externalId": externalId,
	}

	byt, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshalling payload: %w", err)
	}

	_, err = c.client.Do(ctx, "PUT", fmt.Sprintf("v2/catalog/custom-types/%s/external-id", id), bytes.NewReader(byt))
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}

	return nil
}
