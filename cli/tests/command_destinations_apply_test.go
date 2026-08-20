package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// destinationSnapshotIgnore are the volatile upstream fields excluded from the
// snapshot comparison: server-assigned id, workspace scoping, version, and
// timestamps. Write-only secrets (e.g. S3 access keys) are never returned by the
// API, so they never appear here.
//
// These fields are ignored by value, not by presence: CompareStates reports a
// missing key before it consults the ignore list, so the API must still return
// each key or the comparison fails.
var destinationSnapshotIgnore = []string{"id", "workspaceId", "version", "createdAt", "updatedAt"}

// destinationRawSecrets are literal secret values from the var file that must
// never surface in CLI output.
var destinationRawSecrets = []string{
	"AKIAXXXXXXXXXXXXXXXX",
	"wJalrXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
	"httpBasicPasswordXXXXXXXXXXXX",
	"httpBearerTokenXXXXXXXXXXXXXXX",
	"httpApiKeyValueXXXXXXXXXXXXXXX",
	"rsPasswordXXXXXXXXXXXXXXXXXXXX",
	"AKIARSXXXXXXXXXXXXXX",
	"rsSecretAccessKeyXXXXXXXXXXXXXXXXXXXXXXXXX",
	"attentiveTagApiKeyXXXXXXXXXXXX",
	"customerioAudienceApiKeyXXXXXX",
	"customerioAudienceAppApiKeyXXX",
	"confluentCloudApiKeyXXXXXXXXXX",
	"confluentCloudApiSecretXXXXXXX",
	"not-a-real-key",
	"rudder-cli-e2e@example.iam.gserviceaccount.com",
	"000000000000000000000",
	"not-a-real-google-pubsub-key",
	"rudder-cli-google-pubsub-e2e@example.iam.gserviceaccount.com",
	"111111111111111111111",
	"facebookConversionsAccessTokenXXX",
	"AKIAKINESISXXXXXXXXX",
	"kinesisSecretAccessKeyXXXXXXXXXXXXXXXXXXXX",
	"marketoClientSecretXXXXXXXXXXX",
	"salesforcePasswordXXXXXXXXXXXX",
	"salesforceInitialAccessTokenXX",
	"statsigSecretKeyXXXXXXXXXXXXXX",
	"zendeskApiTokenXXXXXXXXXXXXXXX",
	"redisPasswordXXXXXXXXXXXXXXXXX",
	"redisCaCertificateXXXXXXXXXXXX",
	"snowflakePasswordXXXXXXXXXXXXX",
	"snowflakePrivateKeyXXXXXXXXXXX",
	"snowflakeKeyPassphraseXXXXXXXX",
	"AKIASNOWFLAKEXXXXXXX",
	"snowflakeSecretAccessKeyXXXXXXXXXXXXXXXXXX",
	"snowflake-dummy",
	"snowflakeAzureAccountKeyXXXXXX",
	"snowflakeAzureSasTokenXXXXXXXX",
	"bqstream-dummy",
}

// assertNoRawSecrets fails if any write-only secret appears in CLI output. Both
// apply and destroy render plans, so both are leak surfaces.
func assertNoRawSecrets(t *testing.T, out []byte) {
	t.Helper()
	for _, secret := range destinationRawSecrets {
		assert.NotContains(t, string(out), secret, "raw secret must never appear in CLI output")
	}
}

// TestDestinationsApply drives the destination provider end-to-end against a live
// stack: apply create → apply update → re-apply churns only the write-only secret.
// All destination specs live side by side under one folder and are applied
// together (like the catalog e2e applies events, properties, etc. in one shot),
// then the managed destinations are snapshot-compared upstream via
// DestinationSnapshotTester — the same file-manager + count-guard +
// per-resource-compare structure verifyState uses. DestinationSupport enables
// the kind, while UnverifiedDestinations remains enabled here because these
// fixtures include unverified destination types such as attentive_tag, http, rs,
// and salesforce. Key-auth specs reference secrets via {{ .VAR }} placeholders resolved at
// apply time.
//
// Gated behind RUN_DESTINATION_E2E because it needs a live stack with support for
// the unverified fixture set, and the destroy below would otherwise wipe the
// workspace before the (failing) apply on a stack without that support. The skip
// must come before the destroy.
func TestDestinationsApply(t *testing.T) {
	if os.Getenv("RUN_DESTINATION_E2E") != "1" {
		t.Skip("set RUN_DESTINATION_E2E=1 with a live destination-enabled stack; current fixtures include unverified destination types")
	}

	t.Setenv("RUDDERSTACK_X_DESTINATION_SUPPORT", "true")
	t.Setenv("RUDDERSTACK_X_UNVERIFIED_DESTINATIONS", "true")
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	t.Setenv("RUDDERSTACK_X_ENABLE_VAR_SUBSTITUTION", "true")

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	projectDir := filepath.Join("testdata", "destinations")
	varFile := filepath.Join(projectDir, "destinations.vars.yaml")

	out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "destroy failed: %s", out)
	assertNoRawSecrets(t, out)

	// A leftover unverified destination breaks other e2e tests' destroy when
	// UnverifiedDestinations is off. Registered after t.Setenv so the flags still
	// hold when this cleanup runs. assert (not require) avoids FailNow mid-cleanup.
	t.Cleanup(func() {
		out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
		assert.NoError(t, err, "cleanup destroy failed: %s", out)
		assertNoRawSecrets(t, out)
	})

	apply := func(t *testing.T, dir string) {
		t.Helper()
		out, err := executor.Execute(cliBinPath, "apply", "-l",
			filepath.Join(projectDir, dir), "--var-file", varFile, "--confirm=false")
		require.NoError(t, err, "%s apply failed: %s", dir, out)
		assertNoRawSecrets(t, out)
	}

	t.Run("apply create", func(t *testing.T) {
		apply(t, "create")
		verifyDestinationState(t, "create")
	})

	t.Run("apply update", func(t *testing.T) {
		apply(t, "update")
		verifyDestinationState(t, "update")
	})

	// Re-apply cannot be a full no-op here: the key-based spec's access keys are
	// write-only, so they map to always-unknown secrets that re-apply every run
	// (see secret.String.Diff). A dry-run would therefore always report a diff.
	// Snapshot the non-secret upstream fields instead to prove nothing else churns,
	// matching the accounts e2e (TestAccountsApply's re-apply subtest).
	t.Run("re-apply churns only the write-only secret", func(t *testing.T) {
		apply(t, "update")
		verifyDestinationState(t, "update")
	})
}

// verifyDestinationState snapshot-compares the managed destinations against
// testdata/expected/upstream/destinations/<dir>, mirroring verifyState.
func verifyDestinationState(t *testing.T, dir string) {
	t.Helper()

	config.InitConfig(config.DefaultConfigFile())
	apiClient, err := client.New(
		config.GetConfig().Auth.AccessToken,
		client.WithBaseURL(config.GetConfig().APIURL),
		client.WithUserAgent("rudder-cli-test"),
	)
	require.NoError(t, err)

	expectedStateDir := filepath.Join("testdata", "expected", "upstream", "destinations", dir)
	fileManager, err := helpers.NewSnapshotFileManager(expectedStateDir)
	require.NoError(t, err)

	tester := helpers.NewDestinationSnapshotTester(
		apiClient.Destinations,
		fileManager,
		destinationSnapshotIgnore,
	)
	assert.NoError(t, tester.SnapshotTest(context.Background()),
		"upstream destination state verification failed for %q", dir)
}
