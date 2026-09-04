package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/testorchestrator"
	"github.com/rudderlabs/rudder-iac/cli/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// badPyLibraryHandle is the handle of the intentionally broken Python library in the
// failure fixture set.
const badPyLibraryHandle = "badPyLibrary"

func TestTransformationsTest(t *testing.T) {
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

	t.Run("success", func(t *testing.T) {
		resultsFile := filepath.Join(t.TempDir(), "test-results.json")
		fixtureDir := filepath.Join("testdata", "project", "transformations-test", "success")

		output, err := executor.Execute(cliBinPath, "transformations", "test", "--all",
			"-l", fixtureDir, "-o", resultsFile)
		require.NoError(t, err, "test command failed: %s", string(output))

		verifyTestResults(t, "success", resultsFile)
	})

	t.Run("failure", func(t *testing.T) {
		resultsFile := filepath.Join(t.TempDir(), "test-results.json")
		fixtureDir := filepath.Join("testdata", "project", "transformations-test", "failure")

		output, err := executor.Execute(cliBinPath, "transformations", "test", "--all",
			"-l", fixtureDir, "-o", resultsFile)
		require.Error(t, err, "test command should fail: %s", string(output))

		verifyTestResults(t, "failure", resultsFile)
		verifyBadLibraryMessage(t, resultsFile)
	})
}

// verifyBadLibraryMessage asserts the stable part of the upstream error reported for
// the deliberately broken Python library.
//
// The full message is snapshot-exempt (see testResultsIgnoreFields) because the
// wrapper the backend puts around the interpreter error is not stable: the same
// fixture has produced both
//
//	BadCodeError("'(' was never closed (<unknown>, line 1)")
//	SyntaxError(("Line 1: SyntaxError: '(' was never closed at statement: ...",))
//
// on main within days of each other, each passing CI at the time. Pinning either
// form makes this test flap on whatever the backend happens to return. The
// interpreter's own diagnostic is the part we actually care about and is common to
// both, so assert on that and let the wrapper vary.
func verifyBadLibraryMessage(t *testing.T, resultsFile string) {
	t.Helper()

	results := readJSONFile(t, resultsFile)
	libs, ok := results["libraries"].([]any)
	require.True(t, ok, "test results contain no libraries array")

	for _, entry := range libs {
		lib, ok := entry.(map[string]any)
		if !ok || lib["handleName"] != badPyLibraryHandle {
			continue
		}

		assert.Equal(t, false, lib["pass"], "%s must be reported as failing", badPyLibraryHandle)
		assert.Contains(t, lib["message"], "'(' was never closed",
			"%s must report the unclosed-paren syntax error", badPyLibraryHandle)
		return
	}

	t.Fatalf("library %q not found in test results", badPyLibraryHandle)
}

func verifyTestResults(t *testing.T, dir, resultsFile string) {
	t.Helper()

	snapshotDir := filepath.Join("testdata", "expected", "upstream", "transformations-test", dir)

	// Snapshot-compare the full test results output against expected.
	actual := readJSONFile(t, resultsFile)
	expected := readJSONFile(t, filepath.Join(snapshotDir, "expected_results.json"))
	sortResultsByName(actual)
	sortResultsByName(expected)

	assert.NoError(t, helpers.CompareStates(actual, expected,
		testResultsIgnoreFields(actual)),
		"test results snapshot comparison failed")

	// Verify upstream versions via API
	results := readResultsFile(t, resultsFile)
	verifyVersions(t, snapshotDir, results)
}

// sortResultsByName sorts both transformations and libraries in-place by name
// so that CompareStates can do index-based comparison deterministically.
func sortResultsByName(results map[string]any) {
	if trs, ok := results["transformations"].([]any); ok {
		sort.Slice(trs, func(i, j int) bool {
			iName := trs[i].(map[string]any)["result"].(map[string]any)["name"].(string)
			jName := trs[j].(map[string]any)["result"].(map[string]any)["name"].(string)
			return iName < jName
		})
	}

	if libs, ok := results["libraries"].([]any); ok {
		sort.Slice(libs, func(i, j int) bool {
			iName := libs[i].(map[string]any)["name"].(string)
			jName := libs[j].(map[string]any)["name"].(string)
			return iName < jName
		})
	}
}

// testResultsIgnoreFields builds ignore paths for dynamic fields in test results,
// based on the actual transformation and library counts.
func testResultsIgnoreFields(results map[string]any) []string {
	var ignore []string

	if trs, ok := results["transformations"].([]any); ok {
		for i := range trs {
			prefix := fmt.Sprintf("transformations[%d].result", i)
			ignore = append(ignore,
				prefix+".id",
				prefix+".versionId",
				prefix+".externalId",
				prefix+".testResult.results",
			)
		}
	}

	if libs, ok := results["libraries"].([]any); ok {
		for i := range libs {
			prefix := fmt.Sprintf("libraries[%d]", i)
			ignore = append(ignore,
				prefix+".id",
				prefix+".versionId",
				prefix+".externalId",
				// The upstream error wrapper varies between backend responses;
				// verifyBadLibraryMessage asserts the stable part instead.
				prefix+".message",
			)
		}
	}

	return ignore
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

func readJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "reading JSON file at %s", path)
	var result map[string]any
	require.NoError(t, json.Unmarshal(data, &result), "deserializing JSON file")
	return result
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
