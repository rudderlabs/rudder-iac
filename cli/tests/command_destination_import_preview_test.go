package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importer"
	"github.com/rudderlabs/rudder-iac/cli/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// importPreviewFillValue fills the scaffolded bearer_token reference (a dummy). Long and
// unique so it cannot be a substring of any other config value (DEX-888).
const importPreviewFillValue = "e2e-import-preview-bearer-0c5d1f7a9b3e"

// TestImportPreviewShowsWhatTheAdoptChanges checks the import -> dry-run ->
// apply cycle for a destination created outside the CLI with the empty values
// and legacy consent blocks the dashboard stores:
//
//   - import scaffolds a reference for the required secret the API never
//     returns (bearer_token under bearerTokenAuth), and the spec validates once
//     the var file is filled in;
//   - the dry run lists the destination as changed on import, and every config
//     key the adopt then removes from the destination is named in it;
//   - after the apply nothing but secret re-application is left to change.
//
// Like the other import e2e it opens with a workspace-wide destroy, so it runs
// only under RUN_IMPORT_E2E against a disposable workspace.
func TestImportPreviewShowsWhatTheAdoptChanges(t *testing.T) {
	if os.Getenv(importE2EEnv) != "1" {
		t.Skip("set RUN_IMPORT_E2E=1 with a disposable live stack to run the destination import preview e2e")
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

	// The name carries the unmanaged-webhook prefix so a cancelled run's
	// leftover is swept by TestMain like the other import e2e's.
	seededName := fmt.Sprintf("%s Preview %d", unmanagedWebhookPrefix, time.Now().UnixNano())
	seeded, err := apiClient.Destinations.Create(ctx, &client.Destination{
		Name: seededName,
		Type: "HTTP",
		Config: []byte(`{
			"apiUrl": "https://e2e-import-preview.example.com/hook",
			"auth": "bearerTokenAuth",
			"bearerToken": "` + importPreviewFillValue + `",
			"method": "POST",
			"format": "JSON",
			"headers": [],
			"queryParams": [],
			"consentManagement": {},
			"ketchConsentPurposes": {},
			"oneTrustCookieCategories": {}
		}`),
		IsEnabled: true,
	})
	require.NoError(t, err, "seeding the unmanaged destination failed")
	t.Cleanup(func() {
		if err := apiClient.Destinations.Delete(context.Background(), seeded.ID); err != nil {
			t.Logf("cleaning up seeded destination %s: %v", seeded.ID, err)
		}
	})

	before, err := apiClient.Destinations.Get(ctx, seeded.ID)
	require.NoError(t, err)

	projectDir := t.TempDir()
	out, err = executor.Execute(cliBinPath, "import", "workspace", "-l", projectDir)
	require.NoError(t, err, "import workspace failed: %s", out)

	importedDir := filepath.Join(projectDir, importer.ImportedDir)
	specPath, spec := findImportedSpec(t, importedDir, "destinations", seededName)
	pruneImport(t, importedDir, "destinations", specPath, seeded.ID)

	reference := regexp.MustCompile(`bearer_token: '\{\{ \.([A-Z0-9_]+) \}\}'`).FindStringSubmatch(spec)
	require.Len(t, reference, 2, "import must scaffold a bearer_token reference: %s", spec)
	// The import is workspace-wide, so on a shared workspace the scaffolded var
	// file also lists the secrets of every other unmanaged resource. pruneImport
	// dropped their specs; keep only this destination's entry, as a user who
	// adopts one slice would.
	varFile := filepath.Join(importedDir, importer.SecretsVarFileName)
	require.NoError(t, os.WriteFile(varFile, []byte(fmt.Sprintf("%s: '%s'\n", reference[1], importPreviewFillValue)), 0o600))

	out, err = executor.Execute(cliBinPath, "validate", "-l", projectDir, "--var-file", varFile)
	require.NoError(t, err, "the imported spec must validate once the var file is filled in: %s", out)

	dryRun, err := executor.Execute(cliBinPath, "apply", "-l", projectDir, "--var-file", varFile, "--dry-run", "--confirm=false")
	require.NoError(t, err, "dry run failed: %s", dryRun)
	plan := string(dryRun)
	assert.Contains(t, plan, "Importable resources, changed on import")

	out, err = executor.Execute(cliBinPath, "apply", "-l", projectDir, "--var-file", varFile, "--confirm=false")
	require.NoError(t, err, "apply after import failed: %s", out)

	after, err := apiClient.Destinations.Get(ctx, seeded.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, after.ExternalID, "apply must claim the destination with an external id")

	removed := removedConfigKeys(t, before.Config, after.Config)
	require.NotEmpty(t, removed, "the seeded empty values are expected to be dropped by the adopt")
	for _, key := range removed {
		assert.True(t, strings.Contains(plan, key) || strings.Contains(plan, utils.ToSnakeCase(key)),
			"the dry run must name %q, which the adopt removed; dry run was:\n%s", key, plan)
	}

	afterPlan, err := executor.Execute(cliBinPath, "apply", "-l", projectDir, "--var-file", varFile, "--dry-run", "--confirm=false")
	require.NoError(t, err, "dry run after apply failed: %s", afterPlan)
	assert.NotContains(t, string(afterPlan), "Importable resources")
	assert.NotContains(t, string(afterPlan), "Updated resources")

	// A managed update replaces the whole config too: a key the definition
	// doesn't model, written by the dashboard after the adopt, is named in the
	// dry run and then removed. The legacy block is one the seed proved the API
	// stores and returns.
	managed, err := apiClient.Destinations.Get(ctx, seeded.ID)
	require.NoError(t, err)
	var config map[string]any
	require.NoError(t, json.Unmarshal(managed.Config, &config))
	config["oneTrustCookieCategories"] = map[string]any{}
	managed.Config, err = json.Marshal(config)
	require.NoError(t, err)
	_, err = apiClient.Destinations.Update(ctx, managed)
	require.NoError(t, err, "adding a key the definition doesn't model failed")

	updatePlan, err := executor.Execute(cliBinPath, "apply", "-l", projectDir, "--var-file", varFile, "--dry-run", "--confirm=false")
	require.NoError(t, err, "dry run after the dashboard edit failed: %s", updatePlan)
	assert.Contains(t, string(updatePlan), "Config (API keys the update removes).oneTrustCookieCategories")

	out, err = executor.Execute(cliBinPath, "apply", "-l", projectDir, "--var-file", varFile, "--confirm=false")
	require.NoError(t, err, "apply after the dashboard edit failed: %s", out)
	updated, err := apiClient.Destinations.Get(ctx, seeded.ID)
	require.NoError(t, err)
	assert.NotContains(t, string(updated.Config), "oneTrustCookieCategories", "the update must remove the key its dry run named")
}

// removedConfigKeys returns the top-level keys of before that after lacks.
func removedConfigKeys(t *testing.T, before, after json.RawMessage) []string {
	t.Helper()

	var b, a map[string]any
	require.NoError(t, json.Unmarshal(before, &b))
	require.NoError(t, json.Unmarshal(after, &a))
	var removed []string
	for key := range b {
		if _, ok := a[key]; !ok {
			removed = append(removed, key)
		}
	}
	return removed
}
