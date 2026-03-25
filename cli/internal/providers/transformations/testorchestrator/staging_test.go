package testorchestrator

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tc "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/testutil"
)

// --- StageTransformation ---

func TestStageTransformation(t *testing.T) {
	t.Run("creates new transformation when remoteID is empty", func(t *testing.T) {
		var capturedReq *tc.CreateTransformationRequest
		store := &testutil.MockTransformationStore{
			CreateTransformationFunc: func(_ context.Context, req *tc.CreateTransformationRequest, publish bool) (*tc.Transformation, error) {
				capturedReq = req
				assert.False(t, publish, "must create as unpublished")
				return &tc.Transformation{VersionID: "ver-new"}, nil
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
		var capturedReq *tc.UpdateTransformationRequest
		store := &testutil.MockTransformationStore{
			UpdateTransformationFunc: func(_ context.Context, id string, req *tc.UpdateTransformationRequest, publish bool) (*tc.Transformation, error) {
				capturedID = id
				capturedReq = req
				assert.False(t, publish, "must update as unpublished")
				return &tc.Transformation{VersionID: "ver-updated"}, nil
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
		store := &testutil.MockTransformationStore{
			CreateTransformationFunc: func(_ context.Context, _ *tc.CreateTransformationRequest, _ bool) (*tc.Transformation, error) {
				return nil, fmt.Errorf("upstream error")
			},
		}

		_, err := StageTransformation(context.Background(), store, &model.TransformationResource{ID: "t1"}, "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "upstream error")
	})

	t.Run("propagates update API error", func(t *testing.T) {
		store := &testutil.MockTransformationStore{
			UpdateTransformationFunc: func(_ context.Context, _ string, _ *tc.UpdateTransformationRequest, _ bool) (*tc.Transformation, error) {
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
		var capturedReq *tc.CreateLibraryRequest
		store := &testutil.MockTransformationStore{
			CreateLibraryFunc: func(_ context.Context, req *tc.CreateLibraryRequest, publish bool) (*tc.TransformationLibrary, error) {
				capturedReq = req
				assert.False(t, publish, "must create as unpublished")
				return &tc.TransformationLibrary{VersionID: "lib-ver-new"}, nil
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
		var capturedReq *tc.UpdateLibraryRequest
		store := &testutil.MockTransformationStore{
			UpdateLibraryFunc: func(_ context.Context, id string, req *tc.UpdateLibraryRequest, publish bool) (*tc.TransformationLibrary, error) {
				capturedID = id
				capturedReq = req
				assert.False(t, publish, "must update as unpublished")
				return &tc.TransformationLibrary{VersionID: "lib-ver-updated"}, nil
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
		store := &testutil.MockTransformationStore{
			CreateLibraryFunc: func(_ context.Context, _ *tc.CreateLibraryRequest, _ bool) (*tc.TransformationLibrary, error) {
				return nil, fmt.Errorf("upstream error")
			},
		}

		_, err := StageLibrary(context.Background(), store, &model.LibraryResource{ID: "lib-1"}, "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "upstream error")
	})

	t.Run("propagates update API error", func(t *testing.T) {
		store := &testutil.MockTransformationStore{
			UpdateLibraryFunc: func(_ context.Context, _ string, _ *tc.UpdateLibraryRequest, _ bool) (*tc.TransformationLibrary, error) {
				return nil, fmt.Errorf("upstream error")
			},
		}

		_, err := StageLibrary(context.Background(), store, &model.LibraryResource{ID: "lib-1"}, "remote-lib-id")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "upstream error")
	})
}
