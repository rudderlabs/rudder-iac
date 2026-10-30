package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const transformationLinkE2EEnv = "RUN_TRANSFORMATION_LINK_E2E"

// TestApplyLinksTransformationCreatedInSameApply checks that one apply can
// create a transformation and link a destination to it: first a new
// destination, then the same destination switched to another new
// transformation. The backend links only a published transformation, so the
// apply must publish it before the destination's link runs.
//
// It opens with a workspace-wide destroy, so it runs only under
// RUN_TRANSFORMATION_LINK_E2E: the e2e CI lane sets it, and a local run should
// set it only against a disposable workspace.
func TestApplyLinksTransformationCreatedInSameApply(t *testing.T) {
	if os.Getenv(transformationLinkE2EEnv) != "1" {
		t.Skip("set RUN_TRANSFORMATION_LINK_E2E=1 with a disposable live stack to run the transformation link e2e")
	}
	allowManagedResidue(t)

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	var (
		ctx       = context.Background()
		apiClient = newAccountsAPIClient(t)
		store     = newTransformationStore(t)
	)

	out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "destroy failed: %s", out)
	t.Cleanup(func() {
		out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
		assert.NoError(t, err, "cleanup destroy failed: %s", out)
	})

	steps := []struct {
		dir            string
		transformation string
	}{
		{dir: "create", transformation: "link-first"},
		{dir: "relink", transformation: "link-second"},
	}
	for _, step := range steps {
		projectDir := filepath.Join("testdata", "project", "transformation-link", step.dir)

		out, err := executor.Execute(cliBinPath, "apply", "-l", projectDir, "--confirm=false")
		require.NoError(t, err, "apply %s failed: %s", step.dir, out)

		transformations, err := store.ListTransformations(ctx)
		require.NoError(t, err)
		var transformationID string
		for _, transformation := range transformations {
			if transformation.ExternalID == step.transformation {
				transformationID = transformation.ID
			}
		}
		require.NotEmpty(t, transformationID, "%s: transformation %q not published upstream", step.dir, step.transformation)

		destinations, err := apiClient.Destinations.GetAll(ctx)
		require.NoError(t, err)
		var destinationID string
		for _, destination := range destinations {
			if destination.ExternalID == "link-webhook" {
				destinationID = destination.ID
			}
		}
		require.NotEmpty(t, destinationID, "%s: managed destination not found upstream", step.dir)

		link, err := apiClient.Destinations.GetTransformation(ctx, destinationID)
		require.NoError(t, err, "%s: the destination must be linked to a transformation", step.dir)
		assert.Equal(t, transformationID, link.TransformationID, step.dir)

		verifyNoChangesToApplyWithArgs(t, executor, projectDir)
	}
}
