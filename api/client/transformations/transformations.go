package transformations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
)

// Transformation API response model
type Transformation struct {
	ID          string   `json:"id"`
	VersionID   string   `json:"versionId"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Code        string   `json:"code"`
	Language    string   `json:"language"`
	Imports     []string `json:"imports,omitempty"`
	WorkspaceID string   `json:"workspaceId"`
	ExternalID  string   `json:"externalId,omitempty"`
	CreatedAt   string   `json:"createdAt,omitempty"`
	CreatedBy   string   `json:"createdBy,omitempty"`
}

// TransformationLibrary API response model
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

// CreateTransformationRequest for creating/updating transformations
type CreateTransformationRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Code        string `json:"code"`
	Language    string `json:"language"`
	TestEvents  []any  `json:"testEvents,omitempty"`
	ExternalID  string `json:"externalId,omitempty"`
}

// CreateLibraryRequest for creating/updating libraries
type CreateLibraryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Code        string `json:"code"`
	Language    string `json:"language"`
	ExternalID  string `json:"externalId,omitempty"`
}

// BatchPublishRequest for batch publishing
type BatchPublishRequest struct {
	Transformations []TransformationVersionInput `json:"transformations"`
	Libraries       []LibraryVersionInput        `json:"libraries"`
}

type TransformationVersionInput struct {
	VersionID string `json:"versionId"`
	TestInput []any  `json:"testInput,omitempty"`
}

type LibraryVersionInput struct {
	VersionID string `json:"versionId"`
}

// BatchPublishResponse from batch publish API
type BatchPublishResponse struct {
	Published bool `json:"published"`
}

// List response wrappers
type TransformationsListResponse struct {
	Transformations []Transformation `json:"transformations"`
}

type LibrariesListResponse struct {
	Libraries []TransformationLibrary `json:"libraries"`
}

// TransformationStore defines the interface for transformation operations
type TransformationStore interface {
	CreateTransformation(ctx context.Context, req CreateTransformationRequest, publish bool) (*Transformation, error)
	UpdateTransformation(ctx context.Context, id string, req CreateTransformationRequest, publish bool) (*Transformation, error)
	GetTransformation(ctx context.Context, id string) (*Transformation, error)
	ListTransformations(ctx context.Context) ([]Transformation, error)
	DeleteTransformation(ctx context.Context, id string) error

	CreateLibrary(ctx context.Context, req CreateLibraryRequest, publish bool) (*TransformationLibrary, error)
	UpdateLibrary(ctx context.Context, id string, req CreateLibraryRequest, publish bool) (*TransformationLibrary, error)
	GetLibrary(ctx context.Context, id string) (*TransformationLibrary, error)
	ListLibraries(ctx context.Context) ([]TransformationLibrary, error)
	DeleteLibrary(ctx context.Context, id string) error

	BatchPublish(ctx context.Context, req BatchPublishRequest) (*BatchPublishResponse, error)
}

type rudderTransformationStore struct {
	client *client.Client
}

// NewRudderTransformationStore creates a transformation store with basic auth
// It takes the base client for configuration (baseURL, userAgent) and an access token for basic auth
func NewRudderTransformationStore(baseClient *client.Client, accessToken string) (TransformationStore, error) {
	// Create basic auth HTTP client
	basicAuthHTTP := NewBasicAuthHTTPClient(accessToken)

	// Create new client with basic auth HTTP client, inheriting baseURL and userAgent from base client
	transformationClient, err := client.New(
		accessToken,
		client.WithHTTPClient(basicAuthHTTP),
	)
	if err != nil {
		return nil, err
	}

	return &rudderTransformationStore{client: transformationClient}, nil
}

// Transformation CRUD methods

func (s *rudderTransformationStore) CreateTransformation(ctx context.Context, req CreateTransformationRequest, publish bool) (*Transformation, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/transformations?publish=%t", publish)
	res, err := s.client.Do(ctx, "POST", path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var transformation Transformation
	if err = json.Unmarshal(res, &transformation); err != nil {
		return nil, err
	}

	return &transformation, nil
}

func (s *rudderTransformationStore) UpdateTransformation(ctx context.Context, id string, req CreateTransformationRequest, publish bool) (*Transformation, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/transformations/%s?publish=%t", id, publish)
	res, err := s.client.Do(ctx, "POST", path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var transformation Transformation
	if err = json.Unmarshal(res, &transformation); err != nil {
		return nil, err
	}

	return &transformation, nil
}

func (s *rudderTransformationStore) GetTransformation(ctx context.Context, id string) (*Transformation, error) {
	res, err := s.client.Do(ctx, "GET", fmt.Sprintf("/transformations/%s", id), nil)
	if err != nil {
		return nil, err
	}

	var transformation Transformation
	if err = json.Unmarshal(res, &transformation); err != nil {
		return nil, err
	}

	return &transformation, nil
}

func (s *rudderTransformationStore) ListTransformations(ctx context.Context) ([]Transformation, error) {
	res, err := s.client.Do(ctx, "GET", "/transformations", nil)
	if err != nil {
		return nil, err
	}

	var response TransformationsListResponse
	if err = json.Unmarshal(res, &response); err != nil {
		return nil, err
	}

	return response.Transformations, nil
}

func (s *rudderTransformationStore) DeleteTransformation(ctx context.Context, id string) error {
	_, err := s.client.Do(ctx, "DELETE", fmt.Sprintf("/transformations/%s", id), nil)
	return err
}

// Library CRUD methods

func (s *rudderTransformationStore) CreateLibrary(ctx context.Context, req CreateLibraryRequest, publish bool) (*TransformationLibrary, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/transformationLibraries?publish=%t", publish)
	res, err := s.client.Do(ctx, "POST", path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var library TransformationLibrary
	if err = json.Unmarshal(res, &library); err != nil {
		return nil, err
	}

	return &library, nil
}

func (s *rudderTransformationStore) UpdateLibrary(ctx context.Context, id string, req CreateLibraryRequest, publish bool) (*TransformationLibrary, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/transformationLibraries/%s?publish=%t", id, publish)
	res, err := s.client.Do(ctx, "POST", path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var library TransformationLibrary
	if err = json.Unmarshal(res, &library); err != nil {
		return nil, err
	}

	return &library, nil
}

func (s *rudderTransformationStore) GetLibrary(ctx context.Context, id string) (*TransformationLibrary, error) {
	res, err := s.client.Do(ctx, "GET", fmt.Sprintf("/transformationLibraries/%s", id), nil)
	if err != nil {
		return nil, err
	}

	var library TransformationLibrary
	if err = json.Unmarshal(res, &library); err != nil {
		return nil, err
	}

	return &library, nil
}

func (s *rudderTransformationStore) ListLibraries(ctx context.Context) ([]TransformationLibrary, error) {
	res, err := s.client.Do(ctx, "GET", "/transformationLibraries", nil)
	if err != nil {
		return nil, err
	}

	var response LibrariesListResponse
	if err = json.Unmarshal(res, &response); err != nil {
		return nil, err
	}

	return response.Libraries, nil
}

func (s *rudderTransformationStore) DeleteLibrary(ctx context.Context, id string) error {
	_, err := s.client.Do(ctx, "DELETE", fmt.Sprintf("/transformationLibraries/%s", id), nil)
	return err
}

// Batch publish

func (s *rudderTransformationStore) BatchPublish(ctx context.Context, req BatchPublishRequest) (*BatchPublishResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := "/transformations/libraries/publish"
	res, err := s.client.Do(ctx, "POST", path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response BatchPublishResponse
	if err = json.Unmarshal(res, &response); err != nil {
		return nil, err
	}

	return &response, nil
}
