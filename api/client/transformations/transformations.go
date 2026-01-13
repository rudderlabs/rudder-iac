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
	librariesPrefix       = "/libraries"
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
	VersionID string `json:"versionId"`
	TestInput []any  `json:"testInput,omitempty"`
}

type BatchPublishLibrary struct {
	VersionID string `json:"versionId"`
}

// BatchPublishResponse is the response from batch publish
type BatchPublishResponse struct {
	Published bool `json:"published"`
}

// TransformationStore provides access to transformation and library operations
type TransformationStore interface {
	// Transformation operations
	CreateTransformation(ctx context.Context, req *CreateTransformationRequest, publish bool) (*Transformation, error)
	UpdateTransformation(ctx context.Context, id string, req *CreateTransformationRequest, publish bool) (*Transformation, error)
	GetTransformation(ctx context.Context, id string) (*Transformation, error)
	ListTransformations(ctx context.Context) ([]*Transformation, error)
	DeleteTransformation(ctx context.Context, id string) error

	// Library operations
	CreateLibrary(ctx context.Context, req *CreateLibraryRequest, publish bool) (*TransformationLibrary, error)
	UpdateLibrary(ctx context.Context, id string, req *CreateLibraryRequest, publish bool) (*TransformationLibrary, error)
	GetLibrary(ctx context.Context, id string) (*TransformationLibrary, error)
	ListLibraries(ctx context.Context) ([]*TransformationLibrary, error)
	DeleteLibrary(ctx context.Context, id string) error

	// Batch operations
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

// UpdateTransformation updates an existing transformation
// Note: API uses POST (not PUT) for updates
func (r *rudderTransformationStore) UpdateTransformation(ctx context.Context, id string, req *CreateTransformationRequest, publish bool) (*Transformation, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshalling update transformation request: %w", err)
	}

	path := fmt.Sprintf("%s/%s?publish=%t", transformationsPrefix, id, publish)
	resp, err := r.client.Do(ctx, "POST", path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("updating transformation: %w", err)
	}

	var transformation Transformation
	if err := json.Unmarshal(resp, &transformation); err != nil {
		return nil, fmt.Errorf("unmarshalling update transformation response: %w", err)
	}

	return &transformation, nil
}

// GetTransformation retrieves a transformation by ID
func (r *rudderTransformationStore) GetTransformation(ctx context.Context, id string) (*Transformation, error) {
	path := fmt.Sprintf("%s/%s", transformationsPrefix, id)
	resp, err := r.client.Do(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting transformation: %w", err)
	}

	var transformation Transformation
	if err := json.Unmarshal(resp, &transformation); err != nil {
		return nil, fmt.Errorf("unmarshalling get transformation response: %w", err)
	}

	return &transformation, nil
}

// ListTransformations retrieves all transformations
func (r *rudderTransformationStore) ListTransformations(ctx context.Context) ([]*Transformation, error) {
	resp, err := r.client.Do(ctx, "GET", transformationsPrefix, nil)
	if err != nil {
		return nil, fmt.Errorf("listing transformations: %w", err)
	}

	var result struct {
		Transformations []Transformation `json:"transformations"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling list transformations response: %w", err)
	}

	// Convert to pointers
	transformations := make([]*Transformation, len(result.Transformations))
	for i := range result.Transformations {
		transformations[i] = &result.Transformations[i]
	}

	return transformations, nil
}

// DeleteTransformation deletes a transformation by ID
func (r *rudderTransformationStore) DeleteTransformation(ctx context.Context, id string) error {
	path := fmt.Sprintf("%s/%s", transformationsPrefix, id)
	_, err := r.client.Do(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("deleting transformation: %w", err)
	}
	return nil
}

// UpdateLibrary updates an existing library
// Note: API uses POST (not PUT) for updates
func (r *rudderTransformationStore) UpdateLibrary(ctx context.Context, id string, req *CreateLibraryRequest, publish bool) (*TransformationLibrary, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshalling update library request: %w", err)
	}

	path := fmt.Sprintf("%s/%s?publish=%t", librariesPrefix, id, publish)
	resp, err := r.client.Do(ctx, "POST", path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("updating library: %w", err)
	}

	var library TransformationLibrary
	if err := json.Unmarshal(resp, &library); err != nil {
		return nil, fmt.Errorf("unmarshalling update library response: %w", err)
	}

	return &library, nil
}

// GetLibrary retrieves a library by ID
func (r *rudderTransformationStore) GetLibrary(ctx context.Context, id string) (*TransformationLibrary, error) {
	path := fmt.Sprintf("%s/%s", librariesPrefix, id)
	resp, err := r.client.Do(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting library: %w", err)
	}

	var library TransformationLibrary
	if err := json.Unmarshal(resp, &library); err != nil {
		return nil, fmt.Errorf("unmarshalling get library response: %w", err)
	}

	return &library, nil
}

// ListLibraries retrieves all libraries
func (r *rudderTransformationStore) ListLibraries(ctx context.Context) ([]*TransformationLibrary, error) {
	resp, err := r.client.Do(ctx, "GET", librariesPrefix, nil)
	if err != nil {
		return nil, fmt.Errorf("listing libraries: %w", err)
	}

	var result struct {
		Libraries []TransformationLibrary `json:"libraries"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling list libraries response: %w", err)
	}

	// Convert to pointers
	libraries := make([]*TransformationLibrary, len(result.Libraries))
	for i := range result.Libraries {
		libraries[i] = &result.Libraries[i]
	}

	return libraries, nil
}

// DeleteLibrary deletes a library by ID
func (r *rudderTransformationStore) DeleteLibrary(ctx context.Context, id string) error {
	path := fmt.Sprintf("%s/%s", librariesPrefix, id)
	_, err := r.client.Do(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("deleting library: %w", err)
	}
	return nil
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