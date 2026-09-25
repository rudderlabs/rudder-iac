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
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	transformationExternalID = "tag-source"
	unmanagedWebhookPrefix   = "E2E Unmanaged Webhook"
)

// TestImportReferencesManagedTransformation checks that an unmanaged destination
// linked to a managed transformation is imported with a "#transformation:" ref
// rather than the server-assigned id, and that adopting it keeps the link.
//
// Like TestAccountsImportWorkspace, it opens with a workspace-wide destroy and
// runs `import workspace`, which stays opt-in until it has a CI-proven run.
func TestImportReferencesManagedTransformation(t *testing.T) {
	if os.Getenv("RUN_IMPORT_E2E") != "1" {
		t.Skip("set RUN_IMPORT_E2E=1 with a disposable live stack to run the transformation ref import e2e")
	}

	allowManagedResidue(t)
	// pruneImport rewrites import-manifest.yaml, which import only emits under this flag.
	t.Setenv("RUDDERSTACK_X_IMPORT_MERGE", "true")

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	ctx := context.Background()
	apiClient := newAccountsAPIClient(t)

	// A cancelled run skips t.Cleanup, and its seeded destination would still be
	// linked to the managed transformation, which blocks the destroy below.
	deleteLeftoverUnmanagedWebhooks(t, apiClient)

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
		// Once adopted, the cleanup destroy has already removed it.
		if err := apiClient.Destinations.Delete(context.Background(), seeded.ID); err != nil {
			t.Logf("cleaning up seeded destination %s: %v", seeded.ID, err)
		}
	})

	_, err = apiClient.Destinations.ConnectTransformation(ctx, seeded.ID, transformationID)
	require.NoError(t, err, "linking the seeded destination to the managed transformation failed")

	out, err = executor.Execute(cliBinPath, "import", "workspace", "-l", projectDir)
	require.NoError(t, err, "import workspace failed: %s", out)

	importedDir := filepath.Join(projectDir, importer.ImportedDir)
	specPath, spec := findDestinationSpec(t, importedDir, seededName)

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

func deleteLeftoverUnmanagedWebhooks(t *testing.T, apiClient *client.Client) {
	t.Helper()

	destinations, err := apiClient.Destinations.GetAll(context.Background())
	require.NoError(t, err)
	for _, destination := range destinations {
		if destination.ExternalID != "" || !strings.HasPrefix(destination.Name, unmanagedWebhookPrefix) {
			continue
		}
		require.NoError(t, apiClient.Destinations.Delete(context.Background(), destination.ID),
			"deleting leftover destination %s", destination.ID)
	}
}

// findDestinationSpec matches by name because the workspace may hold other
// importable destinations.
func findDestinationSpec(t *testing.T, importedDir, name string) (string, string) {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(importedDir, "destinations", "*.yaml"))
	require.NoError(t, err)

	for _, path := range matches {
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		if strings.Contains(string(content), name) {
			return path, string(content)
		}
	}

	t.Fatalf("no scaffolded spec found for destination %q among %v", name, matches)
	return "", ""
}
