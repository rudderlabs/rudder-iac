package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type CategoryCreate struct {
	Name       string `json:"name"`
	ExternalId string `json:"externalId"`
}

type CategoryUpdate struct {
	Name string `json:"name"`
}

type Category struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	WorkspaceID string    `json:"workspaceId"`
	ExternalID  string    `json:"externalId,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CategoryStore interface {
	CreateCategory(ctx context.Context, input CategoryCreate) (*Category, error)
	UpdateCategory(ctx context.Context, id string, input CategoryUpdate) (*Category, error)
	DeleteCategory(ctx context.Context, id string) error
	GetCategory(ctx context.Context, id string) (*Category, error)
	GetCategories(ctx context.Context, options ListOptions) ([]*Category, error)
	SetCategoryExternalId(ctx context.Context, id string, externalId string) error
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

func (c *RudderDataCatalog) GetCategories(ctx context.Context, options ListOptions) ([]*Category, error) {
	cats, err := getAllResourcesPaginated[*Category](ctx, c.client, fmt.Sprintf("v2/catalog/categories%s", options.ToQuery()), c.concurrency)
	if err != nil {
		return nil, err
	}

	filtered := make([]*Category, 0, len(cats))
	for _, cat := range cats {
		if cat.WorkspaceID != "" {
			filtered = append(filtered, cat)
		}
	}

	return filtered, nil
}

func (c *RudderDataCatalog) SetCategoryExternalId(ctx context.Context, id string, externalId string) error {
	payload := map[string]string{
		"externalId": externalId,
	}

	byt, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshalling payload: %w", err)
	}

	_, err = c.client.Do(ctx, "PUT", fmt.Sprintf("v2/catalog/categories/%s/external-id", id), bytes.NewReader(byt))
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}

	return nil
}
