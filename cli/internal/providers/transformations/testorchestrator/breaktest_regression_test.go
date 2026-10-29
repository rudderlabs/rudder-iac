package testorchestrator

import (
	"context"
	"testing"

	tc "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmodifiedTransformationTestDoesNotStageNewVersion(t *testing.T) {
	updates := 0
	store := &testutil.MockTransformationStore{
		GetTransformationFunc: func(_ context.Context, id string) (*tc.Transformation, error) {
			return &tc.Transformation{ID: id, VersionID: "published-version"}, nil
		},
		UpdateTransformationFunc: func(_ context.Context, _ string, _ *tc.UpdateTransformationRequest, _ bool) (*tc.Transformation, error) {
			updates++
			return &tc.Transformation{VersionID: "new-draft-version"}, nil
		},
	}
	versionID, err := getTransformationVersionID(context.Background(), store, &model.TransformationResource{ID: "trans"}, false, "remote-trans")
	require.NoError(t, err)
	assert.Equal(t, "published-version", versionID)
	assert.Zero(t, updates)
}

func TestUnmodifiedLibraryTestDoesNotStageNewVersion(t *testing.T) {
	updates := 0
	store := &testutil.MockTransformationStore{
		GetLibraryFunc: func(_ context.Context, id string) (*tc.TransformationLibrary, error) {
			return &tc.TransformationLibrary{ID: id, VersionID: "published-version"}, nil
		},
		UpdateLibraryFunc: func(_ context.Context, _ string, _ *tc.UpdateLibraryRequest, _ bool) (*tc.TransformationLibrary, error) {
			updates++
			return &tc.TransformationLibrary{VersionID: "new-draft-version"}, nil
		},
	}
	versionID, err := getLibraryVersionID(context.Background(), store, &model.LibraryResource{ID: "library"}, false, "remote-library")
	require.NoError(t, err)
	assert.Equal(t, "published-version", versionID)
	assert.Zero(t, updates)
}
