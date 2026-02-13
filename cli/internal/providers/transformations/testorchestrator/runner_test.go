package testorchestrator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
)

// Helper to convert test cases to test definitions for testing
func toTestDefinitions(testCases []TestCase) []transformations.TestDefinition {
	defs := make([]transformations.TestDefinition, len(testCases))
	for i, tc := range testCases {
		defs[i] = transformations.TestDefinition{
			Name:           tc.Name,
			Input:          tc.InputEvents,
			ExpectedOutput: tc.ExpectedOutput,
		}
	}
	return defs
}

func TestHasFailures(t *testing.T) {
	t.Run("no failures", func(t *testing.T) {
		results := &TestResults{
			Transformations: []*TransformationTestWithDefinitions{
				{
					Result: &transformations.TransformationTestResult{
						TestSuiteResult: transformations.TestSuiteRunResult{
							Results: []transformations.TestResult{
								{Status: transformations.TestRunStatusPass},
								{Status: transformations.TestRunStatusPass},
							},
						},
					},
				},
			},
		}

		assert.False(t, results.HasFailures())
	})

	t.Run("has fail status", func(t *testing.T) {
		results := &TestResults{
			Transformations: []*TransformationTestWithDefinitions{
				{
					Result: &transformations.TransformationTestResult{
						TestSuiteResult: transformations.TestSuiteRunResult{
							Results: []transformations.TestResult{
								{Status: transformations.TestRunStatusPass},
								{Status: transformations.TestRunStatusFail},
							},
						},
					},
				},
			},
		}

		assert.True(t, results.HasFailures())
	})

	t.Run("has error status", func(t *testing.T) {
		results := &TestResults{
			Transformations: []*TransformationTestWithDefinitions{
				{
					Result: &transformations.TransformationTestResult{
						TestSuiteResult: transformations.TestSuiteRunResult{
							Results: []transformations.TestResult{
								{Status: transformations.TestRunStatusError},
							},
						},
					},
				},
			},
		}

		assert.True(t, results.HasFailures())
	})

	t.Run("empty results", func(t *testing.T) {
		results := &TestResults{
			Transformations: []*TransformationTestWithDefinitions{},
		}

		assert.False(t, results.HasFailures())
	})

	t.Run("multiple transformations with mixed results", func(t *testing.T) {
		results := &TestResults{
			Transformations: []*TransformationTestWithDefinitions{
				{
					Result: &transformations.TransformationTestResult{
						TestSuiteResult: transformations.TestSuiteRunResult{
							Results: []transformations.TestResult{
								{Status: transformations.TestRunStatusPass},
							},
						},
					},
				},
				{
					Result: &transformations.TransformationTestResult{
						TestSuiteResult: transformations.TestSuiteRunResult{
							Results: []transformations.TestResult{
								{Status: transformations.TestRunStatusFail},
							},
						},
					},
				},
			},
		}

		assert.True(t, results.HasFailures())
	})
}

func TestGetLibraryVersionsForUnit(t *testing.T) {
	t.Run("extracts version IDs for unit libraries", func(t *testing.T) {
		runner := &Runner{}

		unit := &TestUnit{
			Libraries: []*model.LibraryResource{
				{ID: "lib-1"},
				{ID: "lib-2"},
			},
		}

		libraryVersionMap := map[string]string{
			"lib-1": "ver-1",
			"lib-2": "ver-2",
			"lib-3": "ver-3", // Not used by this unit
		}

		versionIDs := runner.getLibraryVersionsForUnit(unit, libraryVersionMap)

		require.Len(t, versionIDs, 2)
		assert.Contains(t, versionIDs, "ver-1")
		assert.Contains(t, versionIDs, "ver-2")
		assert.NotContains(t, versionIDs, "ver-3")
	})

	t.Run("returns empty for unit with no libraries", func(t *testing.T) {
		runner := &Runner{}

		unit := &TestUnit{
			Libraries: []*model.LibraryResource{},
		}

		libraryVersionMap := map[string]string{
			"lib-1": "ver-1",
		}

		versionIDs := runner.getLibraryVersionsForUnit(unit, libraryVersionMap)

		assert.Empty(t, versionIDs)
	})

	t.Run("skips libraries not in version map", func(t *testing.T) {
		runner := &Runner{}

		unit := &TestUnit{
			Libraries: []*model.LibraryResource{
				{ID: "lib-1"},
				{ID: "lib-missing"},
			},
		}

		libraryVersionMap := map[string]string{
			"lib-1": "ver-1",
		}

		versionIDs := runner.getLibraryVersionsForUnit(unit, libraryVersionMap)

		require.Len(t, versionIDs, 1)
		assert.Contains(t, versionIDs, "ver-1")
	})
}

func TestBuildTestRequest(t *testing.T) {
	t.Run("builds request with transformation and libraries", func(t *testing.T) {
		runner := &Runner{}

		testCases := []TestCase{
			{
				Name:           "test1",
				InputEvents:    []any{map[string]any{"type": "track"}},
				ExpectedOutput: []any{map[string]any{"type": "track", "modified": true}},
			},
			{
				Name:           "test2",
				InputEvents:    []any{map[string]any{"type": "identify"}},
				ExpectedOutput: nil,
			},
		}

		libraryVersionIDs := []string{"lib-ver-1", "lib-ver-2"}

		req := runner.buildTestRequest("trans-ver-1", toTestDefinitions(testCases), libraryVersionIDs)

		require.NotNil(t, req)
		require.Len(t, req.Transformations, 1)
		assert.Equal(t, "trans-ver-1", req.Transformations[0].VersionID)
		require.Len(t, req.Transformations[0].TestSuite, 2)

		// Verify first test case
		assert.Equal(t, "test1", req.Transformations[0].TestSuite[0].Name)
		assert.Equal(t, testCases[0].InputEvents, req.Transformations[0].TestSuite[0].Input)
		assert.Equal(t, testCases[0].ExpectedOutput, req.Transformations[0].TestSuite[0].ExpectedOutput)

		// Verify second test case
		assert.Equal(t, "test2", req.Transformations[0].TestSuite[1].Name)
		assert.Equal(t, testCases[1].InputEvents, req.Transformations[0].TestSuite[1].Input)
		assert.Nil(t, req.Transformations[0].TestSuite[1].ExpectedOutput)

		// Verify libraries
		require.Len(t, req.Libraries, 2)
		assert.Equal(t, "lib-ver-1", req.Libraries[0].VersionID)
		assert.Equal(t, "lib-ver-2", req.Libraries[1].VersionID)
	})

	t.Run("builds request with no libraries", func(t *testing.T) {
		runner := &Runner{}

		testCases := []TestCase{
			{
				Name:        "test1",
				InputEvents: []any{map[string]any{"type": "track"}},
			},
		}

		req := runner.buildTestRequest("trans-ver-1", toTestDefinitions(testCases), []string{})

		require.NotNil(t, req)
		require.Len(t, req.Transformations, 1)
		assert.Equal(t, "trans-ver-1", req.Transformations[0].VersionID)
		assert.Len(t, req.Transformations[0].TestSuite, 1)
		assert.Empty(t, req.Libraries)
	})

	t.Run("builds request with empty test cases", func(t *testing.T) {
		runner := &Runner{}

		req := runner.buildTestRequest("trans-ver-1", toTestDefinitions([]TestCase{}), []string{"lib-ver-1"})

		require.NotNil(t, req)
		require.Len(t, req.Transformations, 1)
		assert.Equal(t, "trans-ver-1", req.Transformations[0].VersionID)
		assert.Empty(t, req.Transformations[0].TestSuite)
		require.Len(t, req.Libraries, 1)
		assert.Equal(t, "lib-ver-1", req.Libraries[0].VersionID)
	})

	t.Run("preserves test case order", func(t *testing.T) {
		runner := &Runner{}

		testCases := []TestCase{
			{Name: "test-a", InputEvents: []any{}},
			{Name: "test-b", InputEvents: []any{}},
			{Name: "test-c", InputEvents: []any{}},
		}

		req := runner.buildTestRequest("trans-ver-1", toTestDefinitions(testCases), []string{})

		require.Len(t, req.Transformations[0].TestSuite, 3)
		assert.Equal(t, "test-a", req.Transformations[0].TestSuite[0].Name)
		assert.Equal(t, "test-b", req.Transformations[0].TestSuite[1].Name)
		assert.Equal(t, "test-c", req.Transformations[0].TestSuite[2].Name)
	})
}

func TestTestUnitIsLibraryModified(t *testing.T) {
	t.Run("returns true for modified library", func(t *testing.T) {
		unit := &TestUnit{
			ModifiedLibraryURNs: map[string]bool{
				"transformation-library:lib-1": true,
			},
		}

		assert.True(t, unit.IsLibraryModified("transformation-library:lib-1"))
	})

	t.Run("returns false for unmodified library", func(t *testing.T) {
		unit := &TestUnit{
			ModifiedLibraryURNs: map[string]bool{
				"transformation-library:lib-1": true,
			},
		}

		assert.False(t, unit.IsLibraryModified("transformation-library:lib-2"))
	})

	t.Run("returns false for empty modified set", func(t *testing.T) {
		unit := &TestUnit{
			ModifiedLibraryURNs: map[string]bool{},
		}

		assert.False(t, unit.IsLibraryModified("transformation-library:lib-1"))
	})
}
