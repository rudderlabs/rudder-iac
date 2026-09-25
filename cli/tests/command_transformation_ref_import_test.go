package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	transformationExternalID = "tag-source"
	unmanagedWebhookPrefix   = "E2E Unmanaged Webhook"
	importE2EEnv             = "RUN_IMPORT_E2E"
)

// TestImportReferencesManagedTransformation checks that an unmanaged destination
// linked to a managed transformation is imported with a "#transformation:" ref
// rather than the server-assigned id, and that adopting it keeps the link.
//
// It opens with a workspace-wide destroy, so it runs only under RUN_IMPORT_E2E:
// the e2e CI lane sets it, and a local run should set it only against a
// disposable workspace.
func TestImportReferencesManagedTransformation(t *testing.T) {
	if os.Getenv(importE2EEnv) != "1" {
		t.Skip("set RUN_IMPORT_E2E=1 with a disposable live stack to run the transformation ref import e2e")
	}

	allowManagedResidue(t)
	// pruneImport rewrites import-manifest.yaml, which import only emits under this flag.
	t.Setenv("RUDDERSTACK_X_IMPORT_MERGE", "true")

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	ctx := context.Background()
	apiClient := newAccountsAPIClient(t)

	out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "destroy failed: %s", out)

	t.Cleanup(func() {
		out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
		assert.NoError(t, err, "cleanup destroy failed: %s", out)
	})

	// Applied from the directory import later scaffolds into: import refuses to
	// run while a managed resource is missing from the local project.
	projectDir := t.TempDir()
	require.NoError(t, os.CopyFS(projectDir, os.DirFS(filepath.Join("testdata", "project", "transformation-ref-import"))))

	out, err = executor.Execute(cliBinPath, "apply", "-l", projectDir, "--confirm=false")
	require.NoError(t, err, "applying the transformation failed: %s", out)

	all, err := newTransformationStore(t).ListTransformations(ctx)
	require.NoError(t, err)
	var transformationID string
	for _, transformation := range all {
		if transformation.ExternalID == transformationExternalID {
			transformationID = transformation.ID
		}
	}
	require.NotEmpty(t, transformationID, "managed transformation %q not found upstream", transformationExternalID)

	seededName := fmt.Sprintf("%s %d", unmanagedWebhookPrefix, time.Now().UnixNano())
	seeded, err := apiClient.Destinations.Create(ctx, &client.Destination{
		Name:      seededName,
		Type:      "WEBHOOK",
		Config:    []byte(`{"webhookUrl":"https://unmanaged.example.com/hook","webhookMethod":"POST"}`),
		IsEnabled: true,
	})
	require.NoError(t, err, "seeding the unmanaged destination failed")
	t.Cleanup(func() {
		// Cleanups run last-in-first-out, so this runs before the cleanup destroy
		// above. It must: the destination's transformation link blocks destroy.
		if err := apiClient.Destinations.Delete(context.Background(), seeded.ID); err != nil {
			t.Logf("cleaning up seeded destination %s: %v", seeded.ID, err)
		}
	})

	_, err = apiClient.Destinations.ConnectTransformation(ctx, seeded.ID, transformationID)
	require.NoError(t, err, "linking the seeded destination to the managed transformation failed")

	out, err = executor.Execute(cliBinPath, "import", "workspace", "-l", projectDir)
	require.NoError(t, err, "import workspace failed: %s", out)

	importedDir := filepath.Join(projectDir, importer.ImportedDir)
	specPath, spec := findImportedSpec(t, importedDir, "destinations", seededName)

	assert.Contains(t, spec, "#transformation:"+transformationExternalID,
		"the imported destination must reference the managed transformation by name")
	assert.NotContains(t, spec, transformationID,
		"the raw server-assigned id must not survive into the spec")

	pruneImport(t, importedDir, "destinations", specPath, seeded.ID)

	out, err = executor.Execute(cliBinPath, "apply", "-l", projectDir, "--confirm=false")
	require.NoError(t, err, "apply after import failed: %s", out)

	adopted, err := apiClient.Destinations.Get(ctx, seeded.ID)
	require.NoError(t, err, "the seeded destination must still exist after apply")
	assert.NotEmpty(t, adopted.ExternalID, "apply must claim the destination with an external id")

	link, err := apiClient.Destinations.GetTransformation(ctx, seeded.ID)
	require.NoError(t, err, "the adopted destination must stay linked to a transformation")
	assert.Equal(t, transformationID, link.TransformationID)

	verifyNoChangesToApplyWithArgs(t, executor, projectDir)
}

// sweepLeftoverUnmanagedWebhooks runs from TestMain because a cancelled run
// skips t.Cleanup, and its seeded destination stays linked to the managed
// transformation. That link blocks the destroy every live suite opens with, so
// the sweep must precede the first of them, not just this test's.
func sweepLeftoverUnmanagedWebhooks() {
	if os.Getenv(importE2EEnv) != "1" {
		return
	}
	if err := deleteLeftoverUnmanagedWebhooks(context.Background()); err != nil {
		fmt.Println("failed to sweep leftover unmanaged webhooks:", err)
		os.Exit(1)
	}
}

func deleteLeftoverUnmanagedWebhooks(ctx context.Context) error {
	config.InitConfig(config.DefaultConfigFile())
	apiClient, err := client.New(
		config.GetConfig().Auth.AccessToken,
		client.WithBaseURL(config.GetConfig().APIURL),
		client.WithUserAgent("rudder-cli-test"),
	)
	if err != nil {
		return fmt.Errorf("creating api client: %w", err)
	}

	destinations, err := apiClient.Destinations.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("listing destinations: %w", err)
	}
	for _, destination := range destinations {
		if destination.ExternalID != "" || !strings.HasPrefix(destination.Name, unmanagedWebhookPrefix) {
			continue
		}
		if err := apiClient.Destinations.Delete(ctx, destination.ID); err != nil {
			return fmt.Errorf("deleting leftover destination %s: %w", destination.ID, err)
		}
	}
	return nil
}
