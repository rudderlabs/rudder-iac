package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
)

const (
	transformationsPrefix = "/transformations"
	librariesPrefix       = "/transformationLibraries"
)

// Transformation represents a transformation resource from the API
type Transformation struct {
	ID          string   `json:"id"`
	VersionID   string   `json:"versionId"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Code        string   `json:"code"`
	Language    string   `json:"language"`
	Imports     []string `json:"imports"`
	WorkspaceID string   `json:"workspaceId"`
	ExternalID  string   `json:"externalId,omitempty"`
	CreatedAt   string   `json:"createdAt,omitempty"`
	CreatedBy   string   `json:"createdBy,omitempty"`
}

// TransformationLibrary represents a transformation library resource from the API
type TransformationLibrary struct {
	ID          string `json:"id"`
	VersionID   string `json:"versionId"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Code        string `json:"code"`
	Language    string `json:"language"`
	HandleName  string `json:"handleName"`
	WorkspaceID string `json:"workspaceId"`
	ExternalID  string `json:"externalId,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
	CreatedBy   string `json:"createdBy,omitempty"`
}

// CreateTransformationRequest is the request body for creating/updating transformations
type CreateTransformationRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Code        string `json:"code"`
	Language    string `json:"language"`
	TestEvents  []any  `json:"testEvents,omitempty"`
	ExternalID  string `json:"externalId,omitempty"`
}

// CreateLibraryRequest is the request body for creating/updating libraries
type CreateLibraryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Code        string `json:"code"`
	Language    string `json:"language"`
	ExternalID  string `json:"externalId,omitempty"`
}

// BatchPublishRequest is the request body for batch publishing transformations and libraries
type BatchPublishRequest struct {
	Transformations []BatchPublishTransformation `json:"transformations,omitempty"`
	Libraries       []BatchPublishLibrary        `json:"libraries,omitempty"`
}

type BatchPublishTransformation struct {
	VersionID string `json:"versionID"`
	TestInput []any  `json:"testInput,omitempty"`
}

type BatchPublishLibrary struct {
	VersionID string `json:"versionID"`
}

// BatchPublishResponse is the response from batch publish
type BatchPublishResponse struct {
	Published bool `json:"published"`
}

// TransformationStore provides access to transformation and library operations
type TransformationStore interface {
	CreateTransformation(ctx context.Context, req *CreateTransformationRequest, publish bool) (*Transformation, error)
	CreateLibrary(ctx context.Context, req *CreateLibraryRequest, publish bool) (*TransformationLibrary, error)
	BatchPublish(ctx context.Context, req *BatchPublishRequest) error
}

type rudderTransformationStore struct {
	client *client.Client
}

// NewRudderTransformationStore creates a new transformation store
func NewRudderTransformationStore(client *client.Client) TransformationStore {
	return &rudderTransformationStore{
		client: client,
	}
}

// CreateTransformation creates a new transformation
// publish=false creates unpublished revision, publish=true validates and publishes
func (r *rudderTransformationStore) CreateTransformation(ctx context.Context, req *CreateTransformationRequest, publish bool) (*Transformation, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshalling create transformation request: %w", err)
	}

	path := fmt.Sprintf("%s?publish=%t", transformationsPrefix, publish)
	resp, err := r.client.Do(ctx, "POST", path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating transformation: %w", err)
	}

	var transformation Transformation
	if err := json.Unmarshal(resp, &transformation); err != nil {
		return nil, fmt.Errorf("unmarshalling create transformation response: %w", err)
	}

	return &transformation, nil
}

// CreateLibrary creates a new transformation library
// publish=false creates unpublished revision, publish=true validates and publishes
func (r *rudderTransformationStore) CreateLibrary(ctx context.Context, req *CreateLibraryRequest, publish bool) (*TransformationLibrary, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshalling create library request: %w", err)
	}

	path := fmt.Sprintf("%s?publish=%t", librariesPrefix, publish)
	resp, err := r.client.Do(ctx, "POST", path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating library: %w", err)
	}

	var library TransformationLibrary
	if err := json.Unmarshal(resp, &library); err != nil {
		return nil, fmt.Errorf("unmarshalling create library response: %w", err)
	}

	return &library, nil
}

// BatchPublish publishes multiple transformations and libraries in a single batch operation
func (r *rudderTransformationStore) BatchPublish(ctx context.Context, req *BatchPublishRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshalling batch publish request: %w", err)
	}

	path := "/transformations/libraries/publish"
	resp, err := r.client.Do(ctx, "POST", path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("batch publishing: %w", err)
	}

	var publishResp BatchPublishResponse
	if err := json.Unmarshal(resp, &publishResp); err != nil {
		return fmt.Errorf("unmarshalling batch publish response: %w", err)
	}

	if !publishResp.Published {
		return fmt.Errorf("batch publish failed: published=false")
	}

	return nil
}

// TODO: Implement the rest of the methods