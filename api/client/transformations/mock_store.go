package transformations

import (
	"context"
	"fmt"
)

// MockTransformationStore is a test double for TransformationStore.
// Set any Func field to override the default "not implemented" behaviour for that method.
type MockTransformationStore struct {
	CreateTransformationFunc        func(ctx context.Context, req *CreateTransformationRequest, publish bool) (*Transformation, error)
	UpdateTransformationFunc        func(ctx context.Context, id string, req *UpdateTransformationRequest, publish bool) (*Transformation, error)
	GetTransformationFunc           func(ctx context.Context, id string) (*Transformation, error)
	GetTransformationVersionFunc    func(ctx context.Context, id string, versionID string) (*Transformation, error)
	ListTransformationsFunc         func(ctx context.Context) ([]*Transformation, error)
	DeleteTransformationFunc        func(ctx context.Context, id string) error
	SetTransformationExternalIDFunc func(ctx context.Context, id string, externalID string) error

	CreateLibraryFunc        func(ctx context.Context, req *CreateLibraryRequest, publish bool) (*TransformationLibrary, error)
	UpdateLibraryFunc        func(ctx context.Context, id string, req *UpdateLibraryRequest, publish bool) (*TransformationLibrary, error)
	GetLibraryFunc           func(ctx context.Context, id string) (*TransformationLibrary, error)
	GetLibraryVersionFunc    func(ctx context.Context, id string, versionID string) (*TransformationLibrary, error)
	ListLibrariesFunc        func(ctx context.Context) ([]*TransformationLibrary, error)
	DeleteLibraryFunc        func(ctx context.Context, id string) error
	SetLibraryExternalIDFunc func(ctx context.Context, id string, externalID string) error

	BatchPublishFunc func(ctx context.Context, req *BatchPublishRequest) (*BatchPublishResponse, error)
	BatchTestFunc    func(ctx context.Context, req *BatchTestRequest) (*BatchTestResponse, error)
}

func (m *MockTransformationStore) CreateTransformation(ctx context.Context, req *CreateTransformationRequest, publish bool) (*Transformation, error) {
	if m.CreateTransformationFunc != nil {
		return m.CreateTransformationFunc(ctx, req, publish)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *MockTransformationStore) UpdateTransformation(ctx context.Context, id string, req *UpdateTransformationRequest, publish bool) (*Transformation, error) {
	if m.UpdateTransformationFunc != nil {
		return m.UpdateTransformationFunc(ctx, id, req, publish)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *MockTransformationStore) GetTransformation(ctx context.Context, id string) (*Transformation, error) {
	if m.GetTransformationFunc != nil {
		return m.GetTransformationFunc(ctx, id)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *MockTransformationStore) GetTransformationVersion(ctx context.Context, id string, versionID string) (*Transformation, error) {
	if m.GetTransformationVersionFunc != nil {
		return m.GetTransformationVersionFunc(ctx, id, versionID)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *MockTransformationStore) ListTransformations(ctx context.Context) ([]*Transformation, error) {
	if m.ListTransformationsFunc != nil {
		return m.ListTransformationsFunc(ctx)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *MockTransformationStore) DeleteTransformation(ctx context.Context, id string) error {
	if m.DeleteTransformationFunc != nil {
		return m.DeleteTransformationFunc(ctx, id)
	}
	return fmt.Errorf("not implemented")
}

func (m *MockTransformationStore) SetTransformationExternalID(ctx context.Context, id string, externalID string) error {
	if m.SetTransformationExternalIDFunc != nil {
		return m.SetTransformationExternalIDFunc(ctx, id, externalID)
	}
	return fmt.Errorf("not implemented")
}

func (m *MockTransformationStore) CreateLibrary(ctx context.Context, req *CreateLibraryRequest, publish bool) (*TransformationLibrary, error) {
	if m.CreateLibraryFunc != nil {
		return m.CreateLibraryFunc(ctx, req, publish)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *MockTransformationStore) UpdateLibrary(ctx context.Context, id string, req *UpdateLibraryRequest, publish bool) (*TransformationLibrary, error) {
	if m.UpdateLibraryFunc != nil {
		return m.UpdateLibraryFunc(ctx, id, req, publish)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *MockTransformationStore) GetLibrary(ctx context.Context, id string) (*TransformationLibrary, error) {
	if m.GetLibraryFunc != nil {
		return m.GetLibraryFunc(ctx, id)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *MockTransformationStore) GetLibraryVersion(ctx context.Context, id string, versionID string) (*TransformationLibrary, error) {
	if m.GetLibraryVersionFunc != nil {
		return m.GetLibraryVersionFunc(ctx, id, versionID)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *MockTransformationStore) ListLibraries(ctx context.Context) ([]*TransformationLibrary, error) {
	if m.ListLibrariesFunc != nil {
		return m.ListLibrariesFunc(ctx)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *MockTransformationStore) DeleteLibrary(ctx context.Context, id string) error {
	if m.DeleteLibraryFunc != nil {
		return m.DeleteLibraryFunc(ctx, id)
	}
	return fmt.Errorf("not implemented")
}

func (m *MockTransformationStore) SetLibraryExternalID(ctx context.Context, id string, externalID string) error {
	if m.SetLibraryExternalIDFunc != nil {
		return m.SetLibraryExternalIDFunc(ctx, id, externalID)
	}
	return fmt.Errorf("not implemented")
}

func (m *MockTransformationStore) BatchPublish(ctx context.Context, req *BatchPublishRequest) (*BatchPublishResponse, error) {
	if m.BatchPublishFunc != nil {
		return m.BatchPublishFunc(ctx, req)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *MockTransformationStore) BatchTest(ctx context.Context, req *BatchTestRequest) (*BatchTestResponse, error) {
	if m.BatchTestFunc != nil {
		return m.BatchTestFunc(ctx, req)
	}
	return nil, fmt.Errorf("not implemented")
}
