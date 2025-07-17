package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type CategoryCreate struct {
	Name string `json:"name"`
}

type CategoryUpdate struct {
	Name string `json:"name"`
}

type Category struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	WorkspaceID string    `json:"workspaceId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CategoryStore interface {
	CreateCategory(ctx context.Context, input CategoryCreate) (*Category, error)
	UpdateCategory(ctx context.Context, id string, input CategoryUpdate) (*Category, error)
	DeleteCategory(ctx context.Context, id string) error
	GetCategory(ctx context.Context, id string) (*Category, error)
}

func (c *RudderDataCatalog) CreateCategory(ctx context.Context, input CategoryCreate) (*Category, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshalling input: %w", err)
	}

	resp, err := c.client.Do(ctx, "POST", "v2/catalog/categories", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}

	var category Category
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&category); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &category, nil
}

func (c *RudderDataCatalog) UpdateCategory(ctx context.Context, id string, input CategoryUpdate) (*Category, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshalling input: %w", err)
	}

	resp, err := c.client.Do(ctx, "PUT", fmt.Sprintf("v2/catalog/categories/%s", id), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}

	var category Category
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&category); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &category, nil
}

func (c *RudderDataCatalog) DeleteCategory(ctx context.Context, id string) error {
	_, err := c.client.Do(ctx, "DELETE", fmt.Sprintf("v2/catalog/categories/%s", id), nil)
	if err != nil {
		return fmt.Errorf("sending delete request: %w", err)
	}

	return nil
}

func (c *RudderDataCatalog) GetCategory(ctx context.Context, id string) (*Category, error) {
	resp, err := c.client.Do(ctx, "GET", fmt.Sprintf("v2/catalog/categories/%s", id), nil)
	if err != nil {
		return nil, fmt.Errorf("sending get request: %w", err)
	}

	var category Category
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&category); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &category, nil
}
