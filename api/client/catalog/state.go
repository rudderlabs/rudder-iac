package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

type State struct {
	Version   string                   `json:"version"`
	Resources map[string]ResourceState `json:"resources"`
}

type ResourceCollection string

const (
	ResourceCollectionEvents        ResourceCollection = "events"
	ResourceCollectionProperties    ResourceCollection = "properties"
	ResourceCollectionTrackingPlans ResourceCollection = "tracking-plans"
	ResourceCollectionCustomTypes   ResourceCollection = "custom-types"
	ResourceCollectionCategories    ResourceCollection = "categories"
)

type ResourceState struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Input        map[string]interface{} `json:"input"`
	Output       map[string]interface{} `json:"output"`
	Dependencies []string               `json:"dependencies"`
}

type PutStateRequest struct {
	Collection ResourceCollection
	ID         string
	URN        string
	State      ResourceState
}

type DeleteStateRequest struct {
	Collection ResourceCollection
	ID         string
}

type EventStateArgs struct {
	PutStateRequest
	EventID string
}

type StateStore interface {
	ReadState(ctx context.Context) (*State, error)
	PutResourceState(ctx context.Context, req PutStateRequest) error
	DeleteResourceState(ctx context.Context, req DeleteStateRequest) error
}

func (c *RudderDataCatalog) ReadState(ctx context.Context) (*State, error) {
	data, err := c.client.Do(ctx, "GET", "v2/cli/catalog/state", nil)
	if err != nil {
		return nil, fmt.Errorf("sending read state request: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &state, nil
}

func (c *RudderDataCatalog) PutResourceState(ctx context.Context, req PutStateRequest) error {
	request := map[string]interface{}{}
	request["urn"] = req.URN
	request["state"] = req.State

	data, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshalling PUT request: %w", err)
	}

	_, err = c.client.Do(ctx, "PUT", fmt.Sprintf("v2/cli/catalog/%s/%s/state", req.Collection, req.ID), bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("sending put state request: %w", err)
	}

	return nil
}

func (c *RudderDataCatalog) DeleteResourceState(ctx context.Context, req DeleteStateRequest) error {
	_, err := c.client.Do(ctx, "DELETE", fmt.Sprintf("v2/cli/catalog/%s/%s/state", req.Collection, req.ID), nil)
	if err != nil {
		return fmt.Errorf("sending delete state request: %w", err)
	}

	return nil
}
