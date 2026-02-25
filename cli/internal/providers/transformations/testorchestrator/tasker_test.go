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

// --- resolveTransformationVersion ---

func TestResolveTransformationVersion(t *testing.T) {
	t.Run("modified transformation without remoteID is created and versionID returned", func(t *testing.T) {
		store := &stubStore{
			createTransformation: func(_ context.Context, req *transformations.CreateTransformationRequest, _ bool) (*transformations.Transformation, error) {
				return &transformations.Transformation{VersionID: "staged-ver"}, nil
			},
		}
		trans := &model.TransformationResource{ID: "t1", Name: "T1", Code: "code"}

		versionID, err := getTransformationVersionID(context.Background(), store, trans, true, "")

		require.NoError(t, err)
		assert.Equal(t, "staged-ver", versionID)
	})

	t.Run("transformation with remoteID is updated and versionID returned", func(t *testing.T) {
		store := &stubStore{
			updateTransformation: func(_ context.Context, id string, _ *transformations.UpdateTransformationRequest, _ bool) (*transformations.Transformation, error) {
				assert.Equal(t, "remote-id", id)
				return &transformations.Transformation{VersionID: "updated-ver"}, nil
			},
		}
		trans := &model.TransformationResource{ID: "t1"}

		versionID, err := getTransformationVersionID(context.Background(), store, trans, false, "remote-id")

		require.NoError(t, err)
		assert.Equal(t, "updated-ver", versionID)
	})

	t.Run("unmodified transformation with no remoteID returns error", func(t *testing.T) {
		store := &stubStore{}
		trans := &model.TransformationResource{ID: "t1"}

		_, err := getTransformationVersionID(context.Background(), store, trans, false, "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found in remote state")
	})

	t.Run("staging error is propagated", func(t *testing.T) {
		store := &stubStore{
			createTransformation: func(_ context.Context, _ *transformations.CreateTransformationRequest, _ bool) (*transformations.Transformation, error) {
				return nil, fmt.Errorf("stage failed")
			},
		}
		trans := &model.TransformationResource{ID: "t1"}

		_, err := getTransformationVersionID(context.Background(), store, trans, true, "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "stage failed")
	})
}

// --- resolveLibraryVersion ---

func TestResolveLibraryVersion(t *testing.T) {
	t.Run("modified library without remoteID is created and versionID returned", func(t *testing.T) {
		store := &stubStore{
			createLibrary: func(_ context.Context, req *transformations.CreateLibraryRequest, _ bool) (*transformations.TransformationLibrary, error) {
				return &transformations.TransformationLibrary{VersionID: "lib-staged-ver"}, nil
			},
		}
		lib := &model.LibraryResource{ID: "lib-1", Name: "L1", Code: "code"}

		versionID, err := getLibraryVersionID(context.Background(), store, lib, true, "")

		require.NoError(t, err)
		assert.Equal(t, "lib-staged-ver", versionID)
	})

	t.Run("library with remoteID is updated and versionID returned", func(t *testing.T) {
		store := &stubStore{
			updateLibrary: func(_ context.Context, id string, _ *transformations.UpdateLibraryRequest, _ bool) (*transformations.TransformationLibrary, error) {
				assert.Equal(t, "remote-lib-id", id)
				return &transformations.TransformationLibrary{VersionID: "lib-updated-ver"}, nil
			},
		}
		lib := &model.LibraryResource{ID: "lib-1"}

		versionID, err := getLibraryVersionID(context.Background(), store, lib, false, "remote-lib-id")

		require.NoError(t, err)
		assert.Equal(t, "lib-updated-ver", versionID)
	})

	t.Run("unmodified library with no remoteID returns error", func(t *testing.T) {
		store := &stubStore{}
		lib := &model.LibraryResource{ID: "lib-1"}

		_, err := getLibraryVersionID(context.Background(), store, lib, false, "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found in remote state")
	})

	t.Run("staging error is propagated", func(t *testing.T) {
		store := &stubStore{
			createLibrary: func(_ context.Context, _ *transformations.CreateLibraryRequest, _ bool) (*transformations.TransformationLibrary, error) {
				return nil, fmt.Errorf("lib stage failed")
			},
		}
		lib := &model.LibraryResource{ID: "lib-1"}

		_, err := getLibraryVersionID(context.Background(), store, lib, true, "")

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
			ID:   "t-7",
			Name: "T7",
		}
		assert.Equal(t, "t-7", task.Id())
		assert.Nil(t, task.Dependencies())
	})
}
