package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// transformations provides CRUD operations for transformations
type transformations struct {
	*service
}

// Create creates a new transformation.
// When publish is false, creates an unpublished revision.
// When publish is true, validates and publishes the transformation.
func (t *transformations) Create(ctx context.Context, req *CreateTransformationRequest, publish bool) (*Transformation, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("%s?publish=%t", t.basePath, publish)
	res, err := t.client.Do(ctx, "POST", path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var transformation Transformation
	if err = json.Unmarshal(res, &transformation); err != nil {
		return nil, err
	}

	return &transformation, nil
}

// Update updates an existing transformation.
// Note: The API uses POST (not PUT) for updates.
// When publish is false, creates an unpublished revision.
// When publish is true, validates and publishes the transformation.
func (t *transformations) Update(ctx context.Context, id string, req *CreateTransformationRequest, publish bool) (*Transformation, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("%s/%s?publish=%t", t.basePath, id, publish)
	res, err := t.client.Do(ctx, "POST", path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var transformation Transformation
	if err = json.Unmarshal(res, &transformation); err != nil {
		return nil, err
	}

	return &transformation, nil
}

// Get retrieves a transformation by ID.
func (t *transformations) Get(ctx context.Context, id string) (*Transformation, error) {
	var transformation Transformation
	if err := t.get(ctx, id, &transformation); err != nil {
		return nil, err
	}

	return &transformation, nil
}

// List retrieves all transformations.
func (t *transformations) List(ctx context.Context) (*TransformationsListResponse, error) {
	var response TransformationsListResponse
	if err := t.list(ctx, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// Delete deletes a transformation by ID.
func (t *transformations) Delete(ctx context.Context, id string) error {
	return t.service.delete(ctx, id)
}

// BatchPublish publishes multiple transformations and libraries atomically.
// This endpoint uses an absolute path: /transformations/libraries/publish
func (t *transformations) BatchPublish(ctx context.Context, req *BatchPublishRequest) (*BatchPublishResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// Use absolute path as per API specification
	res, err := t.client.Do(ctx, "POST", "/transformations/libraries/publish", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response BatchPublishResponse
	if err = json.Unmarshal(res, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// transformationLibraries provides CRUD operations for transformation libraries
type transformationLibraries struct {
	*service
}

// Create creates a new transformation library.
// When publish is false, creates an unpublished revision.
// When publish is true, validates and publishes the library.
func (l *transformationLibraries) Create(ctx context.Context, req *CreateLibraryRequest, publish bool) (*TransformationLibrary, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("%s?publish=%t", l.basePath, publish)
	res, err := l.client.Do(ctx, "POST", path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var library TransformationLibrary
	if err = json.Unmarshal(res, &library); err != nil {
		return nil, err
	}

	return &library, nil
}

// Update updates an existing transformation library.
// Note: The API uses POST (not PUT) for updates.
// When publish is false, creates an unpublished revision.
// When publish is true, validates and publishes the library.
func (l *transformationLibraries) Update(ctx context.Context, id string, req *CreateLibraryRequest, publish bool) (*TransformationLibrary, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("%s/%s?publish=%t", l.basePath, id, publish)
	res, err := l.client.Do(ctx, "POST", path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var library TransformationLibrary
	if err = json.Unmarshal(res, &library); err != nil {
		return nil, err
	}

	return &library, nil
}

// Get retrieves a transformation library by ID.
func (l *transformationLibraries) Get(ctx context.Context, id string) (*TransformationLibrary, error) {
	var library TransformationLibrary
	if err := l.get(ctx, id, &library); err != nil {
		return nil, err
	}

	return &library, nil
}

// List retrieves all transformation libraries.
func (l *transformationLibraries) List(ctx context.Context) (*LibrariesListResponse, error) {
	var response LibrariesListResponse
	if err := l.list(ctx, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// Delete deletes a transformation library by ID.
func (l *transformationLibraries) Delete(ctx context.Context, id string) error {
	return l.service.delete(ctx, id)
}
