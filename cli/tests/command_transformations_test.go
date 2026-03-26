package tests

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/testorchestrator"
	"github.com/rudderlabs/rudder-iac/cli/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransformationsTestPass(t *testing.T) {
	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	// Clean slate — destroy any residual state from previous runs
	output, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "destroy failed: %s", string(output))

	// Apply base fixtures to establish remote state
	output, err = executor.Execute(cliBinPath, "apply", "-l",
		filepath.Join("testdata", "project", "transformations-test", "setup"),
		"--confirm=false")
	require.NoError(t, err, "apply failed: %s", string(output))

	t.Run("all transformations and libraries pass", func(t *testing.T) {
		outputDir := t.TempDir()
		fixtureDir := filepath.Join("testdata", "project", "transformations-test", "success")

		output, err := executor.Execute(cliBinPath, "transformations", "test", "--all",
			"-l", fixtureDir, "--output-path", outputDir)
		require.NoError(t, err, "test command failed: %s", string(output))

		verifyTestResults(t, filepath.Join(outputDir, "test-results.json"))
	})
}

func verifyTestResults(t *testing.T, resultsFile string) {
	t.Helper()

	results := readResultsFile(t, resultsFile)
	assert.Equal(t, testorchestrator.RunStatusExecuted, results.Status)
	assert.False(t, results.HasFailures())
	assert.Len(t, results.Transformations, 4)
	assert.Len(t, results.Libraries, 1)

	for _, tr := range results.Transformations {
		assert.True(t, tr.Result.Pass, "transformation %s should pass", tr.Result.Name)
		require.NotEmpty(t, tr.Result.TestSuiteResult.Results,
			"transformation %s should have test results", tr.Result.Name)
		for _, r := range tr.Result.TestSuiteResult.Results {
			assert.Equal(t, transformations.TestRunStatusPass, r.Status,
				"test %s in %s should pass", r.Name, tr.Result.Name)
		}
	}

	for _, lib := range results.Libraries {
		assert.True(t, lib.Pass, "library %s should pass", lib.HandleName)
		assert.Empty(t, lib.Message, "library %s should have no error message", lib.HandleName)
	}

	snapshotDir := filepath.Join("testdata", "expected", "upstream", "transformations-test", "success")
	verifyVersions(t, snapshotDir, results)
}

// verifyVersions uses the snapshot framework to compare each transformation and
// library version fetched from the API against expected snapshot files.
func verifyVersions(t *testing.T, snapshotDir string, results *testorchestrator.TestResults) {
	t.Helper()

	store := newTransformationStore(t)
	fileManager, err := helpers.NewSnapshotFileManager(snapshotDir)
	require.NoError(t, err)

	tester := helpers.NewTransformationSnapshotTester(
		store,
		helpers.VersionRefs(results),
		fileManager,
		[]string{
			"id",
			"versionId",
			"codeVersion",
			"workspaceId",
			"externalId",
			"createdAt",
			"updatedAt",
			"createdBy",
			"updatedBy",
		},
	)
	assert.NoError(t, tester.SnapshotTest(context.Background()),
		"version snapshot verification failed")
}

func readResultsFile(t *testing.T, path string) *testorchestrator.TestResults {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "reading results file at %s", path)
	var results testorchestrator.TestResults
	require.NoError(t, json.Unmarshal(data, &results), "deserializing results file")
	return &results
}

func newTransformationStore(t *testing.T) transformations.TransformationStore {
	t.Helper()
	config.InitConfig(config.DefaultConfigFile())
	apiClient, err := client.New(
		config.GetConfig().Auth.AccessToken,
		client.WithBaseURL(config.GetConfig().APIURL),
		client.WithUserAgent("rudder-cli-test"),
	)
	require.NoError(t, err)
	return transformations.NewRudderTransformationStore(apiClient)
}
