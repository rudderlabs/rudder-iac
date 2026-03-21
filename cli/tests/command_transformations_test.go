package tests

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/testorchestrator"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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

		verifyTestResults(t, fixtureDir, filepath.Join(outputDir, "test-results.json"))
	})

	t.Run("failures detected alongside passing resources", func(t *testing.T) {
		outputDir := t.TempDir()
		fixtureDir := filepath.Join("testdata", "project", "transformations-test", "failure")

		output, err := executor.Execute(cliBinPath, "transformations", "test", "--all",
			"-l", fixtureDir, "--output-path", outputDir)
		require.Error(t, err, "test command should fail: %s", string(output))

		verifyFailureTestResults(t, fixtureDir, filepath.Join(outputDir, "test-results.json"))
	})
}

func verifyTestResults(t *testing.T, fixtureDir, resultsFile string) {
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

	verifyVersions(t, fixtureDir, results)
}

// verifyVersions fetches each transformation and library version via the API
// and compares name, description, and code against the fixture YAML spec.
func verifyVersions(t *testing.T, fixtureDir string, results *testorchestrator.TestResults) {
	t.Helper()

	fixtures := loadFixtureSpecs(t, fixtureDir)
	store := newTransformationStore(t)
	ctx := context.Background()

	for _, tr := range results.Transformations {
		version, err := store.GetTransformationVersion(ctx, tr.Result.ID, tr.Result.VersionID)
		require.NoError(t, err, "fetching version for %s", tr.Result.Name)
		expected := fixtures[tr.Result.Name]
		assert.Equal(t, expected.Name, version.Name)
		assert.Equal(t, expected.Description, version.Description)
		assert.Equal(t, strings.TrimSpace(expected.Code), strings.TrimSpace(version.Code))
	}

	for _, lib := range results.Libraries {
		version, err := store.GetLibraryVersion(ctx, lib.ID, lib.VersionID)
		require.NoError(t, err, "fetching version for %s", lib.HandleName)
		expected := fixtures[version.Name]
		assert.Equal(t, expected.Name, version.Name)
		assert.Equal(t, expected.Description, version.Description)
		assert.Equal(t, strings.TrimSpace(expected.Code), strings.TrimSpace(version.Code))
	}
}

func readResultsFile(t *testing.T, path string) *testorchestrator.TestResults {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "reading results file at %s", path)
	var results testorchestrator.TestResults
	require.NoError(t, json.Unmarshal(data, &results), "deserializing results file")
	return &results
}

type fixtureSpec struct {
	Name        string
	Description string
	Code        string
}

// loadFixtureSpecs parses all *.yaml files at the top level of fixtureDir into
// a map of spec.name → fixtureSpec. Subdirectories (input/, output/) are ignored.
func loadFixtureSpecs(t *testing.T, fixtureDir string) map[string]fixtureSpec {
	t.Helper()
	specs := make(map[string]fixtureSpec)
	files, err := filepath.Glob(filepath.Join(fixtureDir, "*.yaml"))
	require.NoError(t, err)
	for _, f := range files {
		data, err := os.ReadFile(f)
		require.NoError(t, err)
		var fixture struct {
			Spec struct {
				Name        string `yaml:"name"`
				Description string `yaml:"description"`
				Code        string `yaml:"code"`
			} `yaml:"spec"`
		}
		require.NoError(t, yaml.Unmarshal(data, &fixture))
		if fixture.Spec.Name != "" && fixture.Spec.Code != "" {
			specs[fixture.Spec.Name] = fixtureSpec{
				Name:        fixture.Spec.Name,
				Description: fixture.Spec.Description,
				Code:        fixture.Spec.Code,
			}
		}
	}
	return specs
}

func verifyFailureTestResults(t *testing.T, fixtureDir, resultsFile string) {
	t.Helper()

	results := readResultsFile(t, resultsFile)
	assert.Equal(t, testorchestrator.RunStatusExecuted, results.Status)
	assert.True(t, results.HasFailures())
	assert.Len(t, results.Transformations, 5)
	assert.Len(t, results.Libraries, 3)

	// Group transformations by suite-level status
	passNames := trNamesWithStatus(results, transformations.TestRunStatusPass)
	failNames := trNamesWithStatus(results, transformations.TestRunStatusFail)
	errorNames := trNamesWithStatus(results, transformations.TestRunStatusError)

	assert.ElementsMatch(t, []string{"Simple Transform", "Py Transform", "Greeting Transform"}, passNames)
	assert.ElementsMatch(t, []string{"Mismatch Transform"}, failNames)
	assert.ElementsMatch(t, []string{"Error Transform"}, errorNames)

	// Group libraries by pass/fail
	passingLibs := libNamesWithPass(results, true)
	failingLibs := libNamesWithPass(results, false)

	assert.ElementsMatch(t, []string{"utilsLibrary", "errorLibrary"}, passingLibs)
	assert.ElementsMatch(t, []string{"badPyLibrary"}, failingLibs)

	for _, lib := range results.Libraries {
		if !lib.Pass {
			assert.NotEmpty(t, lib.Message, "failing library %s should have an error message", lib.HandleName)
		}
	}

	verifyVersions(t, fixtureDir, results)
}

// trNamesWithStatus returns names of transformations whose TestSuiteResult.Status matches.
func trNamesWithStatus(results *testorchestrator.TestResults, status transformations.TestRunStatus) []string {
	return lo.FilterMap(results.Transformations, func(tr *testorchestrator.TransformationTestWithDefinitions, _ int) (string, bool) {
		return tr.Result.Name, tr.Result.TestSuiteResult.Status == status
	})
}

// libNamesWithPass returns handle names of libraries whose Pass field matches.
func libNamesWithPass(results *testorchestrator.TestResults, pass bool) []string {
	return lo.FilterMap(results.Libraries, func(lib transformations.LibraryTestResult, _ int) (string, bool) {
		return lib.HandleName, lib.Pass == pass
	})
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
