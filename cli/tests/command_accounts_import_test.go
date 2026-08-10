package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	transformationsapi "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importer"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/accounts"
	ttypes "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	accountImportSecret = rawAccountSecret
)

// TestAccountsImportWorkspace drives reusable import scenarios against a live
// disposable stack. Each scenario seeds unmanaged upstream resources directly via
// the API, runs through the real `import workspace` path, and asserts only on the
// specs/manifest entries for its own seeded fixture.
//
// The account scenario's critical assertion is on disk: export masks the
// write-only credential to a "{{ .VAR }}" reference, so no generated file may
// ever contain the real secret — that masking is all that stands between a user
// and a plaintext credential committed to version control.
func TestAccountsImportWorkspace(t *testing.T) {
	t.Setenv("RUDDERSTACK_X_ACCOUNT_SUPPORT", "true")
	t.Setenv("RUDDERSTACK_X_TRANSFORMATIONS", "true")
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	t.Setenv("RUDDERSTACK_X_ENABLE_VAR_SUBSTITUTION", "true")

	runImportScenarios(t, []Scenario{
		{
			Name:   "accounts/secrets",
			Seed:   seedUnmanagedAccount,
			Assert: assertImportedAccountSecretMasked,
		},
		{
			Name:   "transformations/manifest",
			Seed:   seedUnmanagedTransformation,
			Assert: assertImportedTransformationManifest,
		},
	})
}

func seedUnmanagedAccount(t *testing.T, apiClient *client.Client) []Seeded {
	t.Helper()

	name := uniqueImportName(t, "e2e-import-bigquery")
	payload := map[string]any{
		"accountDefinitionName": "SOURCE_BIGQUERY",
		"name":                  name,
		"options": map[string]any{
			"project":  "acme-analytics-import",
			"location": "US",
		},
		"secret": map[string]any{
			"credentials": accountImportSecret,
		},
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	resp, err := apiClient.Do(context.Background(), "POST", "/v2/accounts", bytes.NewReader(body))
	require.NoError(t, err, "seeding an unmanaged account")

	var account client.Account
	require.NoError(t, json.Unmarshal(resp, &account))
	require.NotEmpty(t, account.ID, "seeded account response must include an id")

	return []Seeded{{
		Type:     accounts.AccountResourceType,
		Name:     name,
		RemoteID: account.ID,
		Secrets:  []string{accountImportSecret},
		Cleanup: func(ctx context.Context) error {
			return apiClient.Accounts.Delete(ctx, account.ID)
		},
	}}
}

func assertImportedAccountSecretMasked(t *testing.T, importedDir string, seeded []Seeded) {
	t.Helper()
	require.Len(t, seeded, 1)
	account := seeded[0]

	spec := findSpecByName(t, importedDir, account.Name)
	assert.Equal(t, accounts.AccountSpecKind, spec.Spec.Kind)
	assert.Contains(t, spec.Raw, "SOURCE_BIGQUERY", "scaffolded spec must carry the account definition")

	reference := varReference.FindStringSubmatch(spec.Raw)
	require.NotNil(t, reference, "credentials must be exported as a {{ .VAR }} reference, got:\n%s", spec.Raw)
	varName := reference[1]

	varFile := filepath.Join(importedDir, importer.SecretsVarFileName)
	scaffoldedVars, err := os.ReadFile(varFile)
	require.NoError(t, err, "import must scaffold %s", importer.SecretsVarFileName)
	assert.Contains(t, string(scaffoldedVars), varName+":",
		"var file must hold a placeholder for every variable the specs reference")

	assertNoSecretOnDisk(t, importedDir, account.Secrets)
}

func seedUnmanagedTransformation(t *testing.T, apiClient *client.Client) []Seeded {
	t.Helper()

	name := uniqueImportName(t, "e2e-import-transform")
	payload := map[string]any{
		"name":        name,
		"description": "Seeded unmanaged transformation for import scenario coverage",
		"language":    "javascript",
		"code": `export function transformEvent(event, metadata) {
  return event;
}`,
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	resp, err := apiClient.Do(context.Background(), "POST", "/transformations?publish=false", bytes.NewReader(body))
	require.NoError(t, err, "seeding an unmanaged transformation")

	var transformation transformationsapi.Transformation
	require.NoError(t, json.Unmarshal(resp, &transformation))
	require.NotEmpty(t, transformation.ID, "seeded transformation response must include an id")

	return []Seeded{{
		Type:     ttypes.TransformationResourceType,
		Name:     name,
		RemoteID: transformation.ID,
		Cleanup: func(ctx context.Context) error {
			_, err := apiClient.Do(ctx, "DELETE", fmt.Sprintf("/transformations/%s", transformation.ID), nil)
			var apiErr *client.APIError
			if errors.As(err, &apiErr) && apiErr.HTTPStatusCode == 404 {
				return nil
			}
			return err
		},
	}}
}

func assertImportedTransformationManifest(t *testing.T, importedDir string, seeded []Seeded) {
	t.Helper()
	require.Len(t, seeded, 1)
	transformation := seeded[0]

	spec := findSpecByName(t, importedDir, transformation.Name)
	assert.Equal(t, ttypes.TransformationSpecKind, spec.Spec.Kind)

	specID, ok := spec.Spec.Spec["id"].(string)
	require.True(t, ok, "transformation spec must include a string id")

	entry := manifestEntryByRemoteID(t, importedDir, transformation.RemoteID)
	assert.Equal(t, resources.URN(specID, ttypes.TransformationResourceType), entry.URN)
	assert.Equal(t, transformation.RemoteID, entry.RemoteID)
}
