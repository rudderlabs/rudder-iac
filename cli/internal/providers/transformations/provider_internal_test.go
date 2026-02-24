package transformations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	transformationsClient "github.com/rudderlabs/rudder-iac/api/client/transformations"
)

func TestBuildTestResultsFromResponse(t *testing.T) {
	t.Run("groups duplicate transformation results by ID", func(t *testing.T) {
		provider := &Provider{}

		// Simulate API response with duplicate transformation entries (one per test)
		// This mimics the actual API behavior where each test case is returned separately
		resp := &transformationsClient.BatchPublishResponse{
			Published: false,
			ValidationOutput: transformationsClient.ValidationOutput{
				Transformations: []transformationsClient.TransformationTestResult{
					{
						ID:        "trans-123",
						Name:      "testlib",
						VersionID: "ver-1",
						Imports:   []string{"adlib"},
						Pass:      true,
						TestSuiteResult: transformationsClient.TestSuiteRunResult{
							Status: transformationsClient.TestRunStatusPass,
							Results: []transformationsClient.TestResult{
								{Name: "Test 1", Status: transformationsClient.TestRunStatusPass},
							},
						},
					},
					{
						ID:        "trans-123",
						Name:      "testlib",
						VersionID: "ver-1",
						Imports:   []string{"adlib"},
						Pass:      false,
						TestSuiteResult: transformationsClient.TestSuiteRunResult{
							Status: transformationsClient.TestRunStatusFail,
							Results: []transformationsClient.TestResult{
								{Name: "Test 2", Status: transformationsClient.TestRunStatusFail},
							},
						},
					},
					{
						ID:        "trans-123",
						Name:      "testlib",
						VersionID: "ver-1",
						Imports:   []string{"adlib"},
						Pass:      true,
						TestSuiteResult: transformationsClient.TestSuiteRunResult{
							Status: transformationsClient.TestRunStatusPass,
							Results: []transformationsClient.TestResult{
								{Name: "Test 3", Status: transformationsClient.TestRunStatusPass},
							},
						},
					},
				},
			},
		}

		// Build test results - this should deduplicate by ID
		results := provider.buildTestResultsFromResponse(resp)

		// Should have exactly 1 transformation (grouped by ID)
		require.Len(t, results.Transformations, 1)

		// The single transformation should have all 3 test results merged
		tr := results.Transformations[0].Result
		assert.Equal(t, "trans-123", tr.ID)
		assert.Equal(t, "testlib", tr.Name)
		assert.Len(t, tr.TestSuiteResult.Results, 3)

		// Verify all test results are present
		testNames := []string{tr.TestSuiteResult.Results[0].Name, tr.TestSuiteResult.Results[1].Name, tr.TestSuiteResult.Results[2].Name}
		assert.Contains(t, testNames, "Test 1")
		assert.Contains(t, testNames, "Test 2")
		assert.Contains(t, testNames, "Test 3")

		// Pass should be false because one test failed
		assert.False(t, tr.Pass)

		// Status should be fail because one test failed
		assert.Equal(t, transformationsClient.TestRunStatusFail, tr.TestSuiteResult.Status)
	})

	t.Run("handles single transformation entry", func(t *testing.T) {
		provider := &Provider{}

		resp := &transformationsClient.BatchPublishResponse{
			Published: true,
			ValidationOutput: transformationsClient.ValidationOutput{
				Transformations: []transformationsClient.TransformationTestResult{
					{
						ID:        "trans-456",
						Name:      "single-test",
						VersionID: "ver-2",
						Pass:      true,
						TestSuiteResult: transformationsClient.TestSuiteRunResult{
							Status: transformationsClient.TestRunStatusPass,
							Results: []transformationsClient.TestResult{
								{Name: "Only test", Status: transformationsClient.TestRunStatusPass},
							},
						},
					},
				},
			},
		}

		results := provider.buildTestResultsFromResponse(resp)

		require.Len(t, results.Transformations, 1)
		tr := results.Transformations[0].Result
		assert.Equal(t, "trans-456", tr.ID)
		assert.Len(t, tr.TestSuiteResult.Results, 1)
		assert.True(t, tr.Pass)
	})

	t.Run("handles multiple different transformations", func(t *testing.T) {
		provider := &Provider{}

		resp := &transformationsClient.BatchPublishResponse{
			Published: true,
			ValidationOutput: transformationsClient.ValidationOutput{
				Transformations: []transformationsClient.TransformationTestResult{
					{
						ID:        "trans-1",
						Name:      "first",
						VersionID: "ver-1",
						Pass:      true,
						TestSuiteResult: transformationsClient.TestSuiteRunResult{
							Status: transformationsClient.TestRunStatusPass,
							Results: []transformationsClient.TestResult{
								{Name: "Test A", Status: transformationsClient.TestRunStatusPass},
							},
						},
					},
					{
						ID:        "trans-2",
						Name:      "second",
						VersionID: "ver-2",
						Pass:      true,
						TestSuiteResult: transformationsClient.TestSuiteRunResult{
							Status: transformationsClient.TestRunStatusPass,
							Results: []transformationsClient.TestResult{
								{Name: "Test B", Status: transformationsClient.TestRunStatusPass},
							},
						},
					},
				},
			},
		}

		results := provider.buildTestResultsFromResponse(resp)

		require.Len(t, results.Transformations, 2)

		// Both transformations should be present
		ids := []string{results.Transformations[0].Result.ID, results.Transformations[1].Result.ID}
		assert.Contains(t, ids, "trans-1")
		assert.Contains(t, ids, "trans-2")
	})

	t.Run("preserves imports from first entry when deduplicating", func(t *testing.T) {
		provider := &Provider{}

		resp := &transformationsClient.BatchPublishResponse{
			Published: false,
			ValidationOutput: transformationsClient.ValidationOutput{
				Transformations: []transformationsClient.TransformationTestResult{
					{
						ID:        "trans-123",
						Name:      "testlib",
						VersionID: "ver-1",
						Imports:   []string{"lib1", "lib2"},
						Pass:      true,
						TestSuiteResult: transformationsClient.TestSuiteRunResult{
							Status: transformationsClient.TestRunStatusPass,
							Results: []transformationsClient.TestResult{
								{Name: "Test 1", Status: transformationsClient.TestRunStatusPass},
							},
						},
					},
					{
						ID:        "trans-123",
						Name:      "testlib",
						VersionID: "ver-1",
						Imports:   []string{"lib1", "lib2"},
						Pass:      true,
						TestSuiteResult: transformationsClient.TestSuiteRunResult{
							Status: transformationsClient.TestRunStatusPass,
							Results: []transformationsClient.TestResult{
								{Name: "Test 2", Status: transformationsClient.TestRunStatusPass},
							},
						},
					},
				},
			},
		}

		results := provider.buildTestResultsFromResponse(resp)

		require.Len(t, results.Transformations, 1)
		tr := results.Transformations[0].Result
		assert.Equal(t, []string{"lib1", "lib2"}, tr.Imports)
	})
}
