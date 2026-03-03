package testorchestrator

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
)

// --- StageTransformation ---

func TestStageTransformation(t *testing.T) {
	t.Run("creates new transformation when remoteID is empty", func(t *testing.T) {
		var capturedReq *transformations.CreateTransformationRequest
		store := &stubStore{
			createTransformation: func(_ context.Context, req *transformations.CreateTransformationRequest, publish bool) (*transformations.Transformation, error) {
				capturedReq = req
				assert.False(t, publish, "must create as unpublished")
				return &transformations.Transformation{VersionID: "ver-new"}, nil
			},
		}

		trans := &model.TransformationResource{
			ID:       "t1",
			Name:     "My Trans",
			Code:     "export function transformEvent(e) { return e; }",
			Language: "javascript",
		}

		versionID, err := StageTransformation(context.Background(), store, trans, "")

		require.NoError(t, err)
		assert.Equal(t, "ver-new", versionID)
		require.NotNil(t, capturedReq)
		assert.Equal(t, "t1", capturedReq.ExternalID)
		assert.Equal(t, "My Trans", capturedReq.Name)
		assert.Equal(t, trans.Code, capturedReq.Code)
		assert.Equal(t, "javascript", capturedReq.Language)
	})

	t.Run("updates existing transformation when remoteID is present", func(t *testing.T) {
		var capturedID string
		var capturedReq *transformations.UpdateTransformationRequest
		store := &stubStore{
			updateTransformation: func(_ context.Context, id string, req *transformations.UpdateTransformationRequest, publish bool) (*transformations.Transformation, error) {
				capturedID = id
				capturedReq = req
				assert.False(t, publish, "must update as unpublished")
				return &transformations.Transformation{VersionID: "ver-updated"}, nil
			},
		}

		trans := &model.TransformationResource{
			ID:       "t1",
			Name:     "Updated Trans",
			Code:     "export function transformEvent(e) { return e; }",
			Language: "javascript",
		}

		versionID, err := StageTransformation(context.Background(), store, trans, "remote-id-123")

		require.NoError(t, err)
		assert.Equal(t, "ver-updated", versionID)
		assert.Equal(t, "remote-id-123", capturedID)
		require.NotNil(t, capturedReq)
		assert.Equal(t, "Updated Trans", capturedReq.Name)
	})

	t.Run("propagates create API error", func(t *testing.T) {
		store := &stubStore{
			createTransformation: func(_ context.Context, _ *transformations.CreateTransformationRequest, _ bool) (*transformations.Transformation, error) {
				return nil, fmt.Errorf("upstream error")
			},
		}

		_, err := StageTransformation(context.Background(), store, &model.TransformationResource{ID: "t1"}, "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "upstream error")
	})

	t.Run("propagates update API error", func(t *testing.T) {
		store := &stubStore{
			updateTransformation: func(_ context.Context, _ string, _ *transformations.UpdateTransformationRequest, _ bool) (*transformations.Transformation, error) {
				return nil, fmt.Errorf("upstream error")
			},
		}

		_, err := StageTransformation(context.Background(), store, &model.TransformationResource{ID: "t1"}, "remote-id")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "upstream error")
	})
}

// --- StageLibrary ---

func TestStageLibrary(t *testing.T) {
	t.Run("creates new library when remoteID is empty", func(t *testing.T) {
		var capturedReq *transformations.CreateLibraryRequest
		store := &stubStore{
			createLibrary: func(_ context.Context, req *transformations.CreateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
				capturedReq = req
				assert.False(t, publish, "must create as unpublished")
				return &transformations.TransformationLibrary{VersionID: "lib-ver-new"}, nil
			},
		}

		lib := &model.LibraryResource{
			ID:          "lib-1",
			Name:        "My Lib",
			Description: "helper functions",
			Code:        "export function helper() {}",
			Language:    "javascript",
		}

		versionID, err := StageLibrary(context.Background(), store, lib, "")

		require.NoError(t, err)
		assert.Equal(t, "lib-ver-new", versionID)
		require.NotNil(t, capturedReq)
		assert.Equal(t, "lib-1", capturedReq.ExternalID)
		assert.Equal(t, "My Lib", capturedReq.Name)
		assert.Equal(t, "helper functions", capturedReq.Description)
		assert.Equal(t, lib.Code, capturedReq.Code)
		assert.Equal(t, "javascript", capturedReq.Language)
	})

	t.Run("updates existing library when remoteID is present", func(t *testing.T) {
		var capturedID string
		var capturedReq *transformations.UpdateLibraryRequest
		store := &stubStore{
			updateLibrary: func(_ context.Context, id string, req *transformations.UpdateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
				capturedID = id
				capturedReq = req
				assert.False(t, publish, "must update as unpublished")
				return &transformations.TransformationLibrary{VersionID: "lib-ver-updated"}, nil
			},
		}

		lib := &model.LibraryResource{
			ID:       "lib-1",
			Name:     "Updated Lib",
			Code:     "export function helper() { updated }",
			Language: "javascript",
		}

		versionID, err := StageLibrary(context.Background(), store, lib, "remote-lib-id")

		require.NoError(t, err)
		assert.Equal(t, "lib-ver-updated", versionID)
		assert.Equal(t, "remote-lib-id", capturedID)
		require.NotNil(t, capturedReq)
		assert.Equal(t, "Updated Lib", capturedReq.Name)
	})

	t.Run("propagates create API error", func(t *testing.T) {
		store := &stubStore{
			createLibrary: func(_ context.Context, _ *transformations.CreateLibraryRequest, _ bool) (*transformations.TransformationLibrary, error) {
				return nil, fmt.Errorf("upstream error")
			},
		}

		_, err := StageLibrary(context.Background(), store, &model.LibraryResource{ID: "lib-1"}, "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "upstream error")
	})

	t.Run("propagates update API error", func(t *testing.T) {
		store := &stubStore{
			updateLibrary: func(_ context.Context, _ string, _ *transformations.UpdateLibraryRequest, _ bool) (*transformations.TransformationLibrary, error) {
				return nil, fmt.Errorf("upstream error")
			},
		}

		_, err := StageLibrary(context.Background(), store, &model.LibraryResource{ID: "lib-1"}, "remote-lib-id")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "upstream error")
	})
}

// stubStore is a minimal TransformationStore for staging tests.
// Only the four methods used by StageTransformation / StageLibrary are configurable.
type stubStore struct {
	createTransformation func(context.Context, *transformations.CreateTransformationRequest, bool) (*transformations.Transformation, error)
	updateTransformation func(context.Context, string, *transformations.UpdateTransformationRequest, bool) (*transformations.Transformation, error)
	createLibrary        func(context.Context, *transformations.CreateLibraryRequest, bool) (*transformations.TransformationLibrary, error)
	updateLibrary        func(context.Context, string, *transformations.UpdateLibraryRequest, bool) (*transformations.TransformationLibrary, error)
}

func (s *stubStore) CreateTransformation(ctx context.Context, req *transformations.CreateTransformationRequest, publish bool) (*transformations.Transformation, error) {
	if s.createTransformation != nil {
		return s.createTransformation(ctx, req, publish)
	}
	return nil, fmt.Errorf("CreateTransformation not configured")
}

func (s *stubStore) UpdateTransformation(ctx context.Context, id string, req *transformations.UpdateTransformationRequest, publish bool) (*transformations.Transformation, error) {
	if s.updateTransformation != nil {
		return s.updateTransformation(ctx, id, req, publish)
	}
	return nil, fmt.Errorf("UpdateTransformation not configured")
}

func (s *stubStore) CreateLibrary(ctx context.Context, req *transformations.CreateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
	if s.createLibrary != nil {
		return s.createLibrary(ctx, req, publish)
	}
	return nil, fmt.Errorf("CreateLibrary not configured")
}

func (s *stubStore) UpdateLibrary(ctx context.Context, id string, req *transformations.UpdateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
	if s.updateLibrary != nil {
		return s.updateLibrary(ctx, id, req, publish)
	}
	return nil, fmt.Errorf("UpdateLibrary not configured")
}

func (s *stubStore) GetTransformation(_ context.Context, _ string) (*transformations.Transformation, error) {
	panic("not used in staging tests")
}
func (s *stubStore) ListTransformations(_ context.Context) ([]*transformations.Transformation, error) {
	panic("not used in staging tests")
}
func (s *stubStore) DeleteTransformation(_ context.Context, _ string) error {
	panic("not used in staging tests")
}
func (s *stubStore) SetTransformationExternalID(_ context.Context, _, _ string) error {
	panic("not used in staging tests")
}
func (s *stubStore) GetLibrary(_ context.Context, _ string) (*transformations.TransformationLibrary, error) {
	panic("not used in staging tests")
}
func (s *stubStore) ListLibraries(_ context.Context) ([]*transformations.TransformationLibrary, error) {
	panic("not used in staging tests")
}
func (s *stubStore) DeleteLibrary(_ context.Context, _ string) error {
	panic("not used in staging tests")
}
func (s *stubStore) SetLibraryExternalID(_ context.Context, _, _ string) error {
	panic("not used in staging tests")
}
func (s *stubStore) BatchPublish(_ context.Context, _ *transformations.BatchPublishRequest) (*transformations.BatchPublishResponse, error) {
	panic("not used in staging tests")
}
func (s *stubStore) BatchTest(ctx context.Context, _ *transformations.BatchTestRequest) (*transformations.BatchTestResponse, error) {
	panic("not used in staging tests")
}
