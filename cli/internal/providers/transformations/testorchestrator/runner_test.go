package testorchestrator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
)

// --- HasFailures ---

func TestHasFailures(t *testing.T) {
	t.Run("returns false when all transformation tests pass", func(t *testing.T) {
		results := &TestResults{
			Transformations: []*TransformationTestWithDefinitions{
				{Result: &transformations.TransformationTestResult{
					TestSuiteResult: transformations.TestSuiteRunResult{
						Results: []transformations.TestResult{
							{Status: transformations.TestRunStatusPass},
							{Status: transformations.TestRunStatusPass},
						},
					},
				}},
			},
		}

		assert.False(t, results.HasFailures())
	})

	t.Run("returns true when any transformation test has fail status", func(t *testing.T) {
		results := &TestResults{
			Transformations: []*TransformationTestWithDefinitions{
				{Result: &transformations.TransformationTestResult{
					TestSuiteResult: transformations.TestSuiteRunResult{
						Results: []transformations.TestResult{
							{Status: transformations.TestRunStatusPass},
							{Status: transformations.TestRunStatusFail},
						},
					},
				}},
			},
		}

		assert.True(t, results.HasFailures())
	})

	t.Run("returns true when any transformation test has error status", func(t *testing.T) {
		results := &TestResults{
			Transformations: []*TransformationTestWithDefinitions{
				{Result: &transformations.TransformationTestResult{
					TestSuiteResult: transformations.TestSuiteRunResult{
						Results: []transformations.TestResult{
							{Status: transformations.TestRunStatusError},
						},
					},
				}},
			},
		}

		assert.True(t, results.HasFailures())
	})

	t.Run("returns true when any library test did not pass", func(t *testing.T) {
		results := &TestResults{
			Libraries: []transformations.LibraryTestResult{
				{Pass: true},
				{Pass: false},
			},
		}

		assert.True(t, results.HasFailures())
	})

	t.Run("returns false when all library tests pass", func(t *testing.T) {
		results := &TestResults{
			Libraries: []transformations.LibraryTestResult{
				{Pass: true},
				{Pass: true},
			},
		}

		assert.False(t, results.HasFailures())
	})

	t.Run("returns false for empty results", func(t *testing.T) {
		results := &TestResults{}

		assert.False(t, results.HasFailures())
	})

	t.Run("returns true when one of many transformations has a failed test", func(t *testing.T) {
		results := &TestResults{
			Transformations: []*TransformationTestWithDefinitions{
				{Result: &transformations.TransformationTestResult{
					TestSuiteResult: transformations.TestSuiteRunResult{
						Results: []transformations.TestResult{{Status: transformations.TestRunStatusPass}},
					},
				}},
				{Result: &transformations.TransformationTestResult{
					TestSuiteResult: transformations.TestSuiteRunResult{
						Results: []transformations.TestResult{{Status: transformations.TestRunStatusFail}},
					},
				}},
			},
		}

		assert.True(t, results.HasFailures())
	})
}

// --- buildTestRequest ---

func TestBuildTestRequest(t *testing.T) {
	t.Run("produces request with correct transformation versionID and test suite", func(t *testing.T) {
		testDefs := []*transformations.TestDefinition{
			{Name: "test-a", Input: []any{map[string]any{"type": "track"}}},
			{Name: "test-b", Input: []any{map[string]any{"type": "identify"}}},
		}

		req := buildTestRequest("trans-ver-1", testDefs, nil)

		require.Len(t, req.Transformations, 1)
		assert.Equal(t, "trans-ver-1", req.Transformations[0].VersionID)
		require.Len(t, req.Transformations[0].TestSuite, 2)
		assert.Equal(t, "test-a", req.Transformations[0].TestSuite[0].Name)
		assert.Equal(t, "test-b", req.Transformations[0].TestSuite[1].Name)
	})

	t.Run("includes library inputs for each provided versionID", func(t *testing.T) {
		req := buildTestRequest("trans-ver-1", nil, []string{"lib-ver-1", "lib-ver-2"})

		require.Len(t, req.Libraries, 2)
		assert.Equal(t, "lib-ver-1", req.Libraries[0].VersionID)
		assert.Equal(t, "lib-ver-2", req.Libraries[1].VersionID)
	})

	t.Run("empty library list produces no library inputs", func(t *testing.T) {
		req := buildTestRequest("trans-ver-1", nil, []string{})

		assert.Empty(t, req.Libraries)
	})

	t.Run("test suite preserves expectedOutput", func(t *testing.T) {
		expected := []any{map[string]any{"type": "track", "processed": true}}
		testDefs := []*transformations.TestDefinition{
			{
				Name:           "with-output",
				Input:          []any{map[string]any{"type": "track"}},
				ExpectedOutput: expected,
			},
		}

		req := buildTestRequest("ver-1", testDefs, nil)

		require.Len(t, req.Transformations[0].TestSuite, 1)
		assert.Equal(t, expected, req.Transformations[0].TestSuite[0].ExpectedOutput)
	})

	t.Run("preserves test definition order", func(t *testing.T) {
		testDefs := []*transformations.TestDefinition{
			{Name: "first"},
			{Name: "second"},
			{Name: "third"},
		}

		req := buildTestRequest("ver-1", testDefs, nil)

		suite := req.Transformations[0].TestSuite
		assert.Equal(t, "first", suite[0].Name)
		assert.Equal(t, "second", suite[1].Name)
		assert.Equal(t, "third", suite[2].Name)
	})
}

// --- getLibraryVersionsForUnit ---

func TestGetLibraryVersionsForUnit(t *testing.T) {
	t.Run("extracts versionIDs for all libraries in the unit", func(t *testing.T) {
		unit := &TestUnit{
			Libraries: []*model.LibraryResource{
				{ID: "lib-1"},
				{ID: "lib-2"},
			},
		}
		versionMap := map[string]string{
			"lib-1": "ver-1",
			"lib-2": "ver-2",
			"lib-3": "ver-3", // not in unit
		}

		versionIDs := getLibraryVersionsForUnit(unit, versionMap)

		require.Len(t, versionIDs, 2)
		assert.Contains(t, versionIDs, "ver-1")
		assert.Contains(t, versionIDs, "ver-2")
		assert.NotContains(t, versionIDs, "ver-3")
	})

	t.Run("returns empty slice when unit has no libraries", func(t *testing.T) {
		unit := &TestUnit{Libraries: []*model.LibraryResource{}}
		versionIDs := getLibraryVersionsForUnit(unit, map[string]string{"lib-1": "ver-1"})

		assert.Empty(t, versionIDs)
	})

	t.Run("skips libraries missing from the version map", func(t *testing.T) {
		unit := &TestUnit{
			Libraries: []*model.LibraryResource{
				{ID: "lib-1"},
				{ID: "lib-missing"},
			},
		}
		versionMap := map[string]string{"lib-1": "ver-1"}

		versionIDs := getLibraryVersionsForUnit(unit, versionMap)

		require.Len(t, versionIDs, 1)
		assert.Equal(t, "ver-1", versionIDs[0])
	})
}
