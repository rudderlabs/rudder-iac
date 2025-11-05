package client

import (
	"context"
	"encoding/json"
)

type Workspace struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Environment  string  `json:"environment"`
	Status       string  `json:"status"`
	Region       string  `json:"region"`
	DataPlaneURL *string `json:"dataPlaneURL,omitempty"`
}

type workspaces struct {
	client *Client
}

type workspaceResponse struct {
	Workspace *Workspace `json:"workspace"`
}

func (w *workspaces) GetByAuthToken(ctx context.Context) (*Workspace, error) {
	res, err := w.client.Do(ctx, "GET", "/v2/workspace", nil)
	if err != nil {
		return nil, err
	}

	response := &workspaceResponse{}
	if err := json.Unmarshal(res, response); err != nil {
		return nil, err
	}

	return response.Workspace, nil
}
