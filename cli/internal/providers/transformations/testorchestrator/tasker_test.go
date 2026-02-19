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

// --- resolveTransformationVersion ---

func TestResolveTransformationVersion(t *testing.T) {
	t.Run("modified transformation is staged and versionID returned", func(t *testing.T) {
		store := &stubStore{
			createTransformation: func(_ context.Context, req *transformations.CreateTransformationRequest, _ bool) (*transformations.Transformation, error) {
				return &transformations.Transformation{VersionID: "staged-ver"}, nil
			},
		}
		trans := &model.TransformationResource{ID: "t1", Name: "T1", Code: "code"}

		versionID, err := resolveTransformationVersion(context.Background(), store, trans, true, nil)

		require.NoError(t, err)
		assert.Equal(t, "staged-ver", versionID)
	})

	t.Run("unmodified transformation reuses remote versionID", func(t *testing.T) {
		store := &stubStore{}
		trans := &model.TransformationResource{ID: "t1"}
		remote := &state.ResourceState{
			OutputRaw: &model.TransformationState{ID: "remote-id", VersionID: "existing-ver"},
		}

		versionID, err := resolveTransformationVersion(context.Background(), store, trans, false, remote)

		require.NoError(t, err)
		assert.Equal(t, "existing-ver", versionID)
	})

	t.Run("unmodified transformation with no remote resource returns error", func(t *testing.T) {
		store := &stubStore{}
		trans := &model.TransformationResource{ID: "t1"}

		_, err := resolveTransformationVersion(context.Background(), store, trans, false, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found in remote state")
	})

	t.Run("unmodified transformation with empty versionID in remote state returns error", func(t *testing.T) {
		store := &stubStore{}
		trans := &model.TransformationResource{ID: "t1"}
		remote := &state.ResourceState{
			OutputRaw: &model.TransformationState{ID: "remote-id", VersionID: ""},
		}

		_, err := resolveTransformationVersion(context.Background(), store, trans, false, remote)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no valid versionId")
	})

	t.Run("staging error is propagated", func(t *testing.T) {
		store := &stubStore{
			createTransformation: func(_ context.Context, _ *transformations.CreateTransformationRequest, _ bool) (*transformations.Transformation, error) {
				return nil, fmt.Errorf("stage failed")
			},
		}
		trans := &model.TransformationResource{ID: "t1"}

		_, err := resolveTransformationVersion(context.Background(), store, trans, true, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "stage failed")
	})
}

// --- resolveLibraryVersion ---

func TestResolveLibraryVersion(t *testing.T) {
	t.Run("modified library is staged and versionID returned", func(t *testing.T) {
		store := &stubStore{
			createLibrary: func(_ context.Context, req *transformations.CreateLibraryRequest, _ bool) (*transformations.TransformationLibrary, error) {
				return &transformations.TransformationLibrary{VersionID: "lib-staged-ver"}, nil
			},
		}
		lib := &model.LibraryResource{ID: "lib-1", Name: "L1", Code: "code"}

		versionID, err := resolveLibraryVersion(context.Background(), store, lib, true, nil)

		require.NoError(t, err)
		assert.Equal(t, "lib-staged-ver", versionID)
	})

	t.Run("unmodified library reuses remote versionID", func(t *testing.T) {
		store := &stubStore{}
		lib := &model.LibraryResource{ID: "lib-1"}
		remote := &state.ResourceState{
			OutputRaw: &model.LibraryState{ID: "remote-lib-id", VersionID: "lib-existing-ver"},
		}

		versionID, err := resolveLibraryVersion(context.Background(), store, lib, false, remote)

		require.NoError(t, err)
		assert.Equal(t, "lib-existing-ver", versionID)
	})

	t.Run("unmodified library with no remote resource returns error", func(t *testing.T) {
		store := &stubStore{}
		lib := &model.LibraryResource{ID: "lib-1"}

		_, err := resolveLibraryVersion(context.Background(), store, lib, false, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found in remote state")
	})

	t.Run("unmodified library with empty versionID in remote state returns error", func(t *testing.T) {
		store := &stubStore{}
		lib := &model.LibraryResource{ID: "lib-1"}
		remote := &state.ResourceState{
			OutputRaw: &model.LibraryState{ID: "remote-lib-id", VersionID: ""},
		}

		_, err := resolveLibraryVersion(context.Background(), store, lib, false, remote)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no valid versionId")
	})

	t.Run("staging error is propagated", func(t *testing.T) {
		store := &stubStore{
			createLibrary: func(_ context.Context, _ *transformations.CreateLibraryRequest, _ bool) (*transformations.TransformationLibrary, error) {
				return nil, fmt.Errorf("lib stage failed")
			},
		}
		lib := &model.LibraryResource{ID: "lib-1"}

		_, err := resolveLibraryVersion(context.Background(), store, lib, true, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "lib stage failed")
	})
}

// --- runTransformationVersionTasks ---

func TestRunTransformationVersionTasks(t *testing.T) {
	t.Run("returns versionID map for each task", func(t *testing.T) {
		store := &stubStore{
			createTransformation: func(_ context.Context, req *transformations.CreateTransformationRequest, _ bool) (*transformations.Transformation, error) {
				return &transformations.Transformation{VersionID: "ver-" + req.ExternalID}, nil
			},
		}

		tasks := []*transformationVersionTask{
			{transformation: &model.TransformationResource{ID: "t1"}, isModified: true},
			{transformation: &model.TransformationResource{ID: "t2"}, isModified: true},
		}

		result, err := runTransformationVersionTasks(context.Background(), store, tasks)

		require.NoError(t, err)
		assert.Equal(t, "ver-t1", result["t1"])
		assert.Equal(t, "ver-t2", result["t2"])
	})

	t.Run("empty task list returns empty map", func(t *testing.T) {
		result, err := runTransformationVersionTasks(context.Background(), &stubStore{}, nil)

		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("task failure returns error", func(t *testing.T) {
		store := &stubStore{
			createTransformation: func(_ context.Context, _ *transformations.CreateTransformationRequest, _ bool) (*transformations.Transformation, error) {
				return nil, fmt.Errorf("API failure")
			},
		}

		tasks := []*transformationVersionTask{
			{transformation: &model.TransformationResource{ID: "t1"}, isModified: true},
		}

		_, err := runTransformationVersionTasks(context.Background(), store, tasks)

		require.Error(t, err)
	})
}

// --- runLibraryVersionTasks ---

func TestRunLibraryVersionTasks(t *testing.T) {
	t.Run("returns versionID map for each task", func(t *testing.T) {
		store := &stubStore{
			createLibrary: func(_ context.Context, req *transformations.CreateLibraryRequest, _ bool) (*transformations.TransformationLibrary, error) {
				return &transformations.TransformationLibrary{VersionID: "lib-ver-" + req.ExternalID}, nil
			},
		}

		tasks := []*libraryVersionTask{
			{lib: &model.LibraryResource{ID: "lib-1"}, isModified: true},
			{lib: &model.LibraryResource{ID: "lib-2"}, isModified: true},
		}

		result, err := runLibraryVersionTasks(context.Background(), store, tasks)

		require.NoError(t, err)
		assert.Equal(t, "lib-ver-lib-1", result["lib-1"])
		assert.Equal(t, "lib-ver-lib-2", result["lib-2"])
	})

	t.Run("empty task list returns empty map", func(t *testing.T) {
		result, err := runLibraryVersionTasks(context.Background(), &stubStore{}, nil)

		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("task failure returns error", func(t *testing.T) {
		store := &stubStore{
			createLibrary: func(_ context.Context, _ *transformations.CreateLibraryRequest, _ bool) (*transformations.TransformationLibrary, error) {
				return nil, fmt.Errorf("API failure")
			},
		}

		tasks := []*libraryVersionTask{
			{lib: &model.LibraryResource{ID: "lib-1"}, isModified: true},
		}

		_, err := runLibraryVersionTasks(context.Background(), store, tasks)

		require.Error(t, err)
	})
}

// --- task ID and dependency helpers ---

func TestTaskIDs(t *testing.T) {
	t.Run("libraryVersionTask ID matches library ID", func(t *testing.T) {
		task := &libraryVersionTask{lib: &model.LibraryResource{ID: "lib-42"}}
		assert.Equal(t, "lib-42", task.Id())
		assert.Nil(t, task.Dependencies())
	})

	t.Run("transformationVersionTask ID matches transformation ID", func(t *testing.T) {
		task := &transformationVersionTask{transformation: &model.TransformationResource{ID: "t-99"}}
		assert.Equal(t, "t-99", task.Id())
		assert.Nil(t, task.Dependencies())
	})

	t.Run("testUnitTask ID matches transformation ID", func(t *testing.T) {
		task := &testUnitTask{
			unit: &TestUnit{Transformation: &model.TransformationResource{ID: "t-7"}},
		}
		assert.Equal(t, "t-7", task.Id())
		assert.Nil(t, task.Dependencies())
	})
}
