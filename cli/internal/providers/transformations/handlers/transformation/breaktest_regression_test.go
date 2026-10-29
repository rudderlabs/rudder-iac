package transformation_test

import (
	"context"
	"testing"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/transformation"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportUnchangedTransformationDoesNotCreateVersion(t *testing.T) {
	remote := &transformations.Transformation{
		ID: "trans-123", VersionID: "version-1", Name: "normalize",
		Description: "same description", Code: "return event;", Language: "javascript",
	}
	store := newMockTransformationStore()
	store.getTransformationFunc = func(context.Context, string) (*transformations.Transformation, error) { return remote, nil }
	handler := transformation.NewHandler(store)

	got, err := handler.Impl.Import(context.Background(), &model.TransformationResource{
		ID: "normalize", Name: remote.Name, Description: remote.Description,
		Code: remote.Code, Language: remote.Language,
	}, remote.ID)

	require.NoError(t, err)
	assert.Equal(t, &model.TransformationState{ID: remote.ID, VersionID: remote.VersionID}, got)
	assert.True(t, store.setTransformationExternalIDCalled)
	assert.False(t, store.updateCalled, "adoption alone must not create a new version")
}
