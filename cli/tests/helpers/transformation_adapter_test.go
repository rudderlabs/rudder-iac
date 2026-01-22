package helpers

import (
	"context"
	"fmt"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTransformationStore implements transformations.TransformationStore for testing
type mockTransformationStore struct {
	listTransformationsFunc func(ctx context.Context) ([]*transformations.Transformation, error)
	listLibrariesFunc       func(ctx context.Context) ([]*transformations.TransformationLibrary, error)
	getTransformationFunc   func(ctx context.Context, id string) (*transformations.Transformation, error)
	getLibraryFunc          func(ctx context.Context, id string) (*transformations.TransformationLibrary, error)
}

func (m *mockTransformationStore) ListTransformations(ctx context.Context) ([]*transformations.Transformation, error) {
	if m.listTransformationsFunc != nil {
		return m.listTransformationsFunc(ctx)
	}
	return []*transformations.Transformation{}, nil
}

func (m *mockTransformationStore) ListLibraries(ctx context.Context) ([]*transformations.TransformationLibrary, error) {
	if m.listLibrariesFunc != nil {
		return m.listLibrariesFunc(ctx)
	}
	return []*transformations.TransformationLibrary{}, nil
}

func (m *mockTransformationStore) GetTransformation(ctx context.Context, id string) (*transformations.Transformation, error) {
	if m.getTransformationFunc != nil {
		return m.getTransformationFunc(ctx, id)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) GetLibrary(ctx context.Context, id string) (*transformations.TransformationLibrary, error) {
	if m.getLibraryFunc != nil {
		return m.getLibraryFunc(ctx, id)
	}
	return nil, fmt.Errorf("not implemented")
}

// Implement remaining interface methods as no-ops
func (m *mockTransformationStore) CreateTransformation(ctx context.Context, req *transformations.CreateTransformationRequest, publish bool) (*transformations.Transformation, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) UpdateTransformation(ctx context.Context, id string, req *transformations.UpdateTransformationRequest, publish bool) (*transformations.Transformation, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) DeleteTransformation(ctx context.Context, id string) error {
	return fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) CreateLibrary(ctx context.Context, req *transformations.CreateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) UpdateLibrary(ctx context.Context, id string, req *transformations.UpdateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) DeleteLibrary(ctx context.Context, id string) error {
	return fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) BatchPublish(ctx context.Context, req *transformations.BatchPublishRequest) error {
	return fmt.Errorf("not implemented")
}

func TestTransformationAdapter_RemoteIDs(t *testing.T) {
	t.Parallel()

	t.Run("returns transformations and libraries with external IDs", func(t *testing.T) {
		t.Parallel()

		store := &mockTransformationStore{
			listTransformationsFunc: func(ctx context.Context) ([]*transformations.Transformation, error) {
				return []*transformations.Transformation{
					{ID: "trans-1", ExternalID: "ext-trans-1", Name: "Transform 1"},
					{ID: "trans-2", ExternalID: "ext-trans-2", Name: "Transform 2"},
					{ID: "trans-3", ExternalID: "", Name: "Transform 3"}, // No external ID - should be filtered
				}, nil
			},
			listLibrariesFunc: func(ctx context.Context) ([]*transformations.TransformationLibrary, error) {
				return []*transformations.TransformationLibrary{
					{ID: "lib-1", ExternalID: "ext-lib-1", Name: "Library 1"},
					{ID: "lib-2", ExternalID: "", Name: "Library 2"}, // No external ID - should be filtered
				}, nil
			},
		}

		adapter := NewTransformationAdapter(store)
		ids, err := adapter.RemoteIDs(context.Background())

		require.NoError(t, err)
		assert.Len(t, ids, 3)

		assert.Equal(t, "trans-1", ids["transformation:ext-trans-1"])
		assert.Equal(t, "trans-2", ids["transformation:ext-trans-2"])
		assert.Equal(t, "lib-1", ids["transformation-library:ext-lib-1"])
	})

	t.Run("returns empty map when no resources exist", func(t *testing.T) {
		t.Parallel()

		store := &mockTransformationStore{
			listTransformationsFunc: func(ctx context.Context) ([]*transformations.Transformation, error) {
				return []*transformations.Transformation{}, nil
			},
			listLibrariesFunc: func(ctx context.Context) ([]*transformations.TransformationLibrary, error) {
				return []*transformations.TransformationLibrary{}, nil
			},
		}

		adapter := NewTransformationAdapter(store)
		ids, err := adapter.RemoteIDs(context.Background())

		require.NoError(t, err)
		assert.Len(t, ids, 0)
	})

	t.Run("returns error when listing transformations fails", func(t *testing.T) {
		t.Parallel()

		store := &mockTransformationStore{
			listTransformationsFunc: func(ctx context.Context) ([]*transformations.Transformation, error) {
				return nil, fmt.Errorf("API error")
			},
		}

		adapter := NewTransformationAdapter(store)
		ids, err := adapter.RemoteIDs(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "listing transformations")
		assert.Nil(t, ids)
	})

	t.Run("returns error when listing libraries fails", func(t *testing.T) {
		t.Parallel()

		store := &mockTransformationStore{
			listTransformationsFunc: func(ctx context.Context) ([]*transformations.Transformation, error) {
				return []*transformations.Transformation{}, nil
			},
			listLibrariesFunc: func(ctx context.Context) ([]*transformations.TransformationLibrary, error) {
				return nil, fmt.Errorf("API error")
			},
		}

		adapter := NewTransformationAdapter(store)
		ids, err := adapter.RemoteIDs(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "listing libraries")
		assert.Nil(t, ids)
	})
}

func TestTransformationAdapter_FetchResource(t *testing.T) {
	t.Parallel()

	t.Run("fetches transformation by ID", func(t *testing.T) {
		t.Parallel()

		store := &mockTransformationStore{
			getTransformationFunc: func(ctx context.Context, id string) (*transformations.Transformation, error) {
				return &transformations.Transformation{
					ID:         id,
					Name:       "Test Transform",
					Language:   "javascript",
					ExternalID: "ext-123",
				}, nil
			},
		}

		adapter := NewTransformationAdapter(store)
		result, err := adapter.FetchResource(context.Background(), TransformationResourceType, "trans-123")

		require.NoError(t, err)
		trans := result.(*transformations.Transformation)
		assert.Equal(t, "trans-123", trans.ID)
		assert.Equal(t, "Test Transform", trans.Name)
	})

	t.Run("fetches library by ID", func(t *testing.T) {
		t.Parallel()

		store := &mockTransformationStore{
			getLibraryFunc: func(ctx context.Context, id string) (*transformations.TransformationLibrary, error) {
				return &transformations.TransformationLibrary{
					ID:         id,
					Name:       "Test Library",
					ImportName: "testLib",
					Language:   "javascript",
					ExternalID: "ext-456",
				}, nil
			},
		}

		adapter := NewTransformationAdapter(store)
		result, err := adapter.FetchResource(context.Background(), LibraryResourceType, "lib-456")

		require.NoError(t, err)
		lib := result.(*transformations.TransformationLibrary)
		assert.Equal(t, "lib-456", lib.ID)
		assert.Equal(t, "Test Library", lib.Name)
		assert.Equal(t, "testLib", lib.ImportName)
	})

	t.Run("returns error for unsupported resource type", func(t *testing.T) {
		t.Parallel()

		store := &mockTransformationStore{}
		adapter := NewTransformationAdapter(store)

		_, err := adapter.FetchResource(context.Background(), "unknown-type", "id-123")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported resource type")
	})
}
