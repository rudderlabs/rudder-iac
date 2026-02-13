package testorchestrator

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

// mockTransformationStore implements TransformationStore for testing
type mockTransformationStore struct {
	createTransformationFunc func(ctx context.Context, req *transformations.CreateTransformationRequest, publish bool) (*transformations.Transformation, error)
	updateTransformationFunc func(ctx context.Context, id string, req *transformations.UpdateTransformationRequest, publish bool) (*transformations.Transformation, error)
	createLibraryFunc        func(ctx context.Context, req *transformations.CreateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error)
	updateLibraryFunc        func(ctx context.Context, id string, req *transformations.UpdateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error)
}

func (m *mockTransformationStore) CreateTransformation(ctx context.Context, req *transformations.CreateTransformationRequest, publish bool) (*transformations.Transformation, error) {
	if m.createTransformationFunc != nil {
		return m.createTransformationFunc(ctx, req, publish)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) UpdateTransformation(ctx context.Context, id string, req *transformations.UpdateTransformationRequest, publish bool) (*transformations.Transformation, error) {
	if m.updateTransformationFunc != nil {
		return m.updateTransformationFunc(ctx, id, req, publish)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) CreateLibrary(ctx context.Context, req *transformations.CreateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
	if m.createLibraryFunc != nil {
		return m.createLibraryFunc(ctx, req, publish)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) UpdateLibrary(ctx context.Context, id string, req *transformations.UpdateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
	if m.updateLibraryFunc != nil {
		return m.updateLibraryFunc(ctx, id, req, publish)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) GetTransformation(ctx context.Context, id string) (*transformations.Transformation, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) ListTransformations(ctx context.Context) ([]*transformations.Transformation, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) DeleteTransformation(ctx context.Context, id string) error {
	return fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) SetTransformationExternalID(ctx context.Context, id string, externalID string) error {
	return fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) GetLibrary(ctx context.Context, id string) (*transformations.TransformationLibrary, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) ListLibraries(ctx context.Context) ([]*transformations.TransformationLibrary, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) DeleteLibrary(ctx context.Context, id string) error {
	return fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) SetLibraryExternalID(ctx context.Context, id string, externalID string) error {
	return fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) BatchPublish(ctx context.Context, req *transformations.BatchPublishRequest) error {
	return fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) BatchTest(ctx context.Context, req *transformations.BatchTestRequest) ([]*transformations.TransformationTestResult, error) {
	return nil, fmt.Errorf("not implemented")
}

func TestStageTransformation(t *testing.T) {
	t.Run("creates new transformation when not in remote state", func(t *testing.T) {
		var capturedReq *transformations.CreateTransformationRequest
		var capturedPublish bool

		mockStore := &mockTransformationStore{
			createTransformationFunc: func(ctx context.Context, req *transformations.CreateTransformationRequest, publish bool) (*transformations.Transformation, error) {
				capturedReq = req
				capturedPublish = publish
				return &transformations.Transformation{
					ID:        "remote-123",
					VersionID: "ver-456",
				}, nil
			},
		}

		manager := NewStagingManager(mockStore)
		transformation := &model.TransformationResource{
			ID:          "test-trans",
			Name:        "Test Transformation",
			Description: "Test description",
			Code:        "export function transformEvent() {}",
			Language:    "javascript",
		}

		remoteState := state.EmptyState()
		versionID, err := manager.StageTransformation(context.Background(), transformation, remoteState)

		require.NoError(t, err)
		assert.Equal(t, "ver-456", versionID)
		require.NotNil(t, capturedReq)
		assert.Equal(t, "Test Transformation", capturedReq.Name)
		assert.Equal(t, "Test description", capturedReq.Description)
		assert.Equal(t, "export function transformEvent() {}", capturedReq.Code)
		assert.Equal(t, "javascript", capturedReq.Language)
		assert.Equal(t, "test-trans", capturedReq.ExternalID)
		assert.False(t, capturedPublish, "should create unpublished version")
	})

	t.Run("updates existing transformation when in remote state", func(t *testing.T) {
		var capturedID string
		var capturedReq *transformations.UpdateTransformationRequest
		var capturedPublish bool

		mockStore := &mockTransformationStore{
			updateTransformationFunc: func(ctx context.Context, id string, req *transformations.UpdateTransformationRequest, publish bool) (*transformations.Transformation, error) {
				capturedID = id
				capturedReq = req
				capturedPublish = publish
				return &transformations.Transformation{
					ID:        id,
					VersionID: "ver-updated",
				}, nil
			},
		}

		manager := NewStagingManager(mockStore)
		transformation := &model.TransformationResource{
			ID:          "test-trans",
			Name:        "Updated Transformation",
			Description: "Updated description",
			Code:        "export function transformEvent() { updated }",
			Language:    "javascript",
		}

		// Create remote state with existing transformation
		remoteState := state.EmptyState()
		remoteState.Resources = map[string]*state.ResourceState{
			"transformation:test-trans": {
				Output: map[string]any{
					"id":        "remote-123",
					"versionId": "ver-old",
				},
			},
		}

		versionID, err := manager.StageTransformation(context.Background(), transformation, remoteState)

		require.NoError(t, err)
		assert.Equal(t, "ver-updated", versionID)
		assert.Equal(t, "remote-123", capturedID)
		require.NotNil(t, capturedReq)
		assert.Equal(t, "Updated Transformation", capturedReq.Name)
		assert.Equal(t, "Updated description", capturedReq.Description)
		assert.Equal(t, "export function transformEvent() { updated }", capturedReq.Code)
		assert.Equal(t, "javascript", capturedReq.Language)
		assert.False(t, capturedPublish, "should create unpublished version")
	})

	t.Run("error when remote transformation has no ID", func(t *testing.T) {
		mockStore := &mockTransformationStore{}
		manager := NewStagingManager(mockStore)

		transformation := &model.TransformationResource{
			ID:   "test-trans",
			Name: "Test",
		}

		// Create remote state with transformation but no ID
		remoteState := state.EmptyState()
		remoteState.Resources = map[string]*state.ResourceState{
			"transformation:test-trans": {
				Output: map[string]any{}, // No ID
			},
		}

		versionID, err := manager.StageTransformation(context.Background(), transformation, remoteState)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "has no valid remote ID")
		assert.Empty(t, versionID)
	})

	t.Run("error when create API fails", func(t *testing.T) {
		mockStore := &mockTransformationStore{
			createTransformationFunc: func(ctx context.Context, req *transformations.CreateTransformationRequest, publish bool) (*transformations.Transformation, error) {
				return nil, fmt.Errorf("API error")
			},
		}

		manager := NewStagingManager(mockStore)
		transformation := &model.TransformationResource{
			ID:   "test-trans",
			Name: "Test",
		}

		remoteState := state.EmptyState()
		versionID, err := manager.StageTransformation(context.Background(), transformation, remoteState)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "creating transformation")
		assert.Contains(t, err.Error(), "API error")
		assert.Empty(t, versionID)
	})

	t.Run("error when update API fails", func(t *testing.T) {
		mockStore := &mockTransformationStore{
			updateTransformationFunc: func(ctx context.Context, id string, req *transformations.UpdateTransformationRequest, publish bool) (*transformations.Transformation, error) {
				return nil, fmt.Errorf("API error")
			},
		}

		manager := NewStagingManager(mockStore)
		transformation := &model.TransformationResource{
			ID:   "test-trans",
			Name: "Test",
		}

		// Create remote state with existing transformation
		remoteState := state.EmptyState()
		remoteState.Resources = map[string]*state.ResourceState{
			"transformation:test-trans": {
				Output: map[string]any{
					"id": "remote-123",
				},
			},
		}

		versionID, err := manager.StageTransformation(context.Background(), transformation, remoteState)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "updating transformation")
		assert.Contains(t, err.Error(), "API error")
		assert.Empty(t, versionID)
	})
}

func TestStageLibrary(t *testing.T) {
	t.Run("creates new library when not in remote state", func(t *testing.T) {
		var capturedReq *transformations.CreateLibraryRequest
		var capturedPublish bool

		mockStore := &mockTransformationStore{
			createLibraryFunc: func(ctx context.Context, req *transformations.CreateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
				capturedReq = req
				capturedPublish = publish
				return &transformations.TransformationLibrary{
					ID:        "remote-lib-123",
					VersionID: "lib-ver-456",
				}, nil
			},
		}

		manager := NewStagingManager(mockStore)
		library := &model.LibraryResource{
			ID:          "test-lib",
			Name:        "Test Library",
			Description: "Test library description",
			Code:        "export function helper() {}",
			Language:    "javascript",
		}

		remoteState := state.EmptyState()
		versionID, err := manager.StageLibrary(context.Background(), library, remoteState)

		require.NoError(t, err)
		assert.Equal(t, "lib-ver-456", versionID)
		require.NotNil(t, capturedReq)
		assert.Equal(t, "Test Library", capturedReq.Name)
		assert.Equal(t, "Test library description", capturedReq.Description)
		assert.Equal(t, "export function helper() {}", capturedReq.Code)
		assert.Equal(t, "javascript", capturedReq.Language)
		assert.Equal(t, "test-lib", capturedReq.ExternalID)
		assert.False(t, capturedPublish, "should create unpublished version")
	})

	t.Run("updates existing library when in remote state", func(t *testing.T) {
		var capturedID string
		var capturedReq *transformations.UpdateLibraryRequest
		var capturedPublish bool

		mockStore := &mockTransformationStore{
			updateLibraryFunc: func(ctx context.Context, id string, req *transformations.UpdateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
				capturedID = id
				capturedReq = req
				capturedPublish = publish
				return &transformations.TransformationLibrary{
					ID:        id,
					VersionID: "lib-ver-updated",
				}, nil
			},
		}

		manager := NewStagingManager(mockStore)
		library := &model.LibraryResource{
			ID:          "test-lib",
			Name:        "Updated Library",
			Description: "Updated library description",
			Code:        "export function helper() { updated }",
			Language:    "javascript",
		}

		// Create remote state with existing library
		remoteState := state.EmptyState()
		remoteState.Resources = map[string]*state.ResourceState{
			"transformation-library:test-lib": {
				Output: map[string]any{
					"id":        "remote-lib-123",
					"versionId": "lib-ver-old",
				},
			},
		}

		versionID, err := manager.StageLibrary(context.Background(), library, remoteState)

		require.NoError(t, err)
		assert.Equal(t, "lib-ver-updated", versionID)
		assert.Equal(t, "remote-lib-123", capturedID)
		require.NotNil(t, capturedReq)
		assert.Equal(t, "Updated Library", capturedReq.Name)
		assert.Equal(t, "Updated library description", capturedReq.Description)
		assert.Equal(t, "export function helper() { updated }", capturedReq.Code)
		assert.Equal(t, "javascript", capturedReq.Language)
		assert.False(t, capturedPublish, "should create unpublished version")
	})

	t.Run("error when remote library has no ID", func(t *testing.T) {
		mockStore := &mockTransformationStore{}
		manager := NewStagingManager(mockStore)

		library := &model.LibraryResource{
			ID:   "test-lib",
			Name: "Test",
		}

		// Create remote state with library but no ID
		remoteState := state.EmptyState()
		remoteState.Resources = map[string]*state.ResourceState{
			"transformation-library:test-lib": {
				Output: map[string]any{}, // No ID
			},
		}

		versionID, err := manager.StageLibrary(context.Background(), library, remoteState)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "has no valid remote ID")
		assert.Empty(t, versionID)
	})

	t.Run("error when create API fails", func(t *testing.T) {
		mockStore := &mockTransformationStore{
			createLibraryFunc: func(ctx context.Context, req *transformations.CreateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
				return nil, fmt.Errorf("API error")
			},
		}

		manager := NewStagingManager(mockStore)
		library := &model.LibraryResource{
			ID:   "test-lib",
			Name: "Test",
		}

		remoteState := state.EmptyState()
		versionID, err := manager.StageLibrary(context.Background(), library, remoteState)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "creating library")
		assert.Contains(t, err.Error(), "API error")
		assert.Empty(t, versionID)
	})

	t.Run("error when update API fails", func(t *testing.T) {
		mockStore := &mockTransformationStore{
			updateLibraryFunc: func(ctx context.Context, id string, req *transformations.UpdateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
				return nil, fmt.Errorf("API error")
			},
		}

		manager := NewStagingManager(mockStore)
		library := &model.LibraryResource{
			ID:   "test-lib",
			Name: "Test",
		}

		// Create remote state with existing library
		remoteState := state.EmptyState()
		remoteState.Resources = map[string]*state.ResourceState{
			"transformation-library:test-lib": {
				Output: map[string]any{
					"id": "remote-lib-123",
				},
			},
		}

		versionID, err := manager.StageLibrary(context.Background(), library, remoteState)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "updating library")
		assert.Contains(t, err.Error(), "API error")
		assert.Empty(t, versionID)
	})
}
