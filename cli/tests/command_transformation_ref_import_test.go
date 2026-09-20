package tests

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	transformationsClient "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// transformationExternalID is the spec id the fixture claims, and therefore the
// name the imported destination must reference it by.
const transformationExternalID = "tag-source"

// TestImportReferencesManagedTransformation covers what ReferencedByKind buys.
//
// A destination created outside the CLI can point at a transformation the
// project already manages. Exporting it with the raw transformation id produces
// a spec that applies — the id is real — but one that has quietly left the
// project: the destination no longer moves when the transformation is renamed or
// replaced, and nothing tells the author the link exists. Exporting
// "#transformation:tag-source" keeps the two joined in the project the way the
// author declared them.
//
// The unit tests pin the export shape. What they cannot show is that the id the
// importer meets at runtime — assigned by the server, not by a fixture — is
// matched back to the managed resource.
func TestImportReferencesManagedTransformation(t *testing.T) {
	allowUnverifiedDestinationResidue(t)
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	t.Setenv("RUDDERSTACK_X_TRANSFORMATIONS", "true")
	t.Setenv("RUDDERSTACK_X_DESTINATION_SUPPORT", "true")

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	ctx := context.Background()
	apiClient := newAccountsAPIClient(t)

	// Import is refused while anything is out of sync, so start from a clean
	// workspace — the precondition the accounts import suite establishes too.
	out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "destroy failed: %s", out)

	t.Cleanup(func() {
		out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
		assert.NoError(t, err, "cleanup destroy failed: %s", out)
	})

	// 1. The transformation becomes managed. It is applied from the project
	//    import will later scaffold into, because `import workspace` refuses to
	//    run while the project has changes to sync — and a managed resource
	//    missing from the local project is exactly such a change. That rules out
	//    the empty-temp-dir shape the accounts import suite uses, which only
	//    works there because it leaves nothing managed behind.
	projectDir := t.TempDir()
	copyFixture(t, filepath.Join("testdata", "transformation_ref"), projectDir)

	out, err = executor.Execute(cliBinPath, "apply", "-l", projectDir, "--confirm=false")
	require.NoError(t, err, "applying the transformation failed: %s", out)

	transformationID := managedTransformationID(t, apiClient, transformationExternalID)

	// 2. A destination the CLI did not create, linked to that transformation.
	//    This is the shape import has to recognise: an unmanaged destination
	//    whose transformation is managed.
	seeded, err := apiClient.Destinations.Create(ctx, &client.Destination{
		Name:      "E2E Unmanaged Webhook",
		Type:      "WEBHOOK",
		Config:    []byte(`{"webhookUrl":"https://unmanaged.example.com/hook","webhookMethod":"POST"}`),
		IsEnabled: true,
	})
	require.NoError(t, err, "seeding the unmanaged destination failed")
	t.Cleanup(func() {
		assert.NoError(t, apiClient.Destinations.Delete(context.Background(), seeded.ID))
	})

	_, err = apiClient.Destinations.ConnectTransformation(ctx, seeded.ID, transformationID)
	require.NoError(t, err, "linking the seeded destination to the managed transformation failed")

	// 3. Import, and read what was written for that destination.
	out, err = executor.Execute(cliBinPath, "import", "workspace", "-l", projectDir)
	require.NoError(t, err, "import workspace failed: %s", out)

	spec := findSpecMentioning(t, filepath.Join(projectDir, importer.ImportedDir), "E2E Unmanaged Webhook")

	assert.Contains(t, spec, "#transformation:"+transformationExternalID,
		"the imported destination must reference the managed transformation by name")
	assert.NotContains(t, spec, transformationID,
		"the raw server-assigned id must not survive into the spec")
}

// managedTransformationID returns the server-assigned id of the managed
// transformation claiming the given externalId.
func managedTransformationID(t *testing.T, apiClient *client.Client, externalID string) string {
	t.Helper()

	all, err := transformationsClient.NewRudderTransformationStore(apiClient).
		ListTransformations(context.Background())
	require.NoError(t, err, "listing transformations")

	for _, transformation := range all {
		if transformation.ExternalID == externalID {
			require.NotEmpty(t, transformation.ID, "managed transformation %q has no id", externalID)
			return transformation.ID
		}
	}

	t.Fatalf("managed transformation %q not found upstream", externalID)
	return ""
}

// copyFixture copies every spec in src into dst, so the test can apply from a
// writable project directory that import will later scaffold into.
func copyFixture(t *testing.T, src, dst string) {
	t.Helper()

	entries, err := os.ReadDir(src)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		body, err := os.ReadFile(filepath.Join(src, entry.Name()))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dst, entry.Name()), body, 0o600))
	}
}

// findSpecMentioning returns the contents of the one scaffolded spec containing
// the given marker, failing if none or several do.
func findSpecMentioning(t *testing.T, dir, marker string) string {
	t.Helper()

	var found []string
	require.NoError(t, filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(path) != ".yaml" {
			return err
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(body), marker) {
			found = append(found, string(body))
		}
		return nil
	}))

	require.Len(t, found, 1, "expected exactly one scaffolded spec mentioning %q", marker)
	return found[0]
}
