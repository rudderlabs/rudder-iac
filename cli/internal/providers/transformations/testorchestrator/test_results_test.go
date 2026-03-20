package testorchestrator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
)

func TestRunStatus(t *testing.T) {
	t.Run("RunStatusExecuted is the default run status", func(t *testing.T) {
		r := &TestResults{Status: RunStatusExecuted}
		assert.Equal(t, RunStatusExecuted, r.Status)
	})
}

func TestHasFailures(t *testing.T) {
	tests := []struct {
		name     string
		results  *TestResults
		expected bool
	}{
		{
			name:     "no libraries and no transformations",
			results:  &TestResults{},
			expected: false,
		},
		{
			name: "one library fails",
			results: &TestResults{
				Libraries: []transformations.LibraryTestResult{
					{HandleName: "lib1", Pass: true},
					{HandleName: "lib2", Pass: false},
				},
			},
			expected: true,
		},
		{
			name: "all transformation test results pass",
			results: &TestResults{
				Transformations: []*TransformationTestWithDefinitions{
					{
						Result: &transformations.TransformationTestResult{
							TestSuiteResult: transformations.TestSuiteRunResult{
								Results: []transformations.TestResult{
									{Name: "test1", Status: transformations.TestRunStatusPass},
									{Name: "test2", Status: transformations.TestRunStatusPass},
								},
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "one transformation test result fails",
			results: &TestResults{
				Transformations: []*TransformationTestWithDefinitions{
					{
						Result: &transformations.TransformationTestResult{
							TestSuiteResult: transformations.TestSuiteRunResult{
								Results: []transformations.TestResult{
									{Name: "test1", Status: transformations.TestRunStatusPass},
									{Name: "test2", Status: transformations.TestRunStatusFail},
								},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "one transformation test result errors",
			results: &TestResults{
				Transformations: []*TransformationTestWithDefinitions{
					{
						Result: &transformations.TransformationTestResult{
							TestSuiteResult: transformations.TestSuiteRunResult{
								Results: []transformations.TestResult{
									{Name: "test1", Status: transformations.TestRunStatusError},
								},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "library fails and transformations pass",
			results: &TestResults{
				Libraries: []transformations.LibraryTestResult{
					{HandleName: "lib1", Pass: false},
				},
				Transformations: []*TransformationTestWithDefinitions{
					{
						Result: &transformations.TransformationTestResult{
							TestSuiteResult: transformations.TestSuiteRunResult{
								Results: []transformations.TestResult{
									{Name: "test1", Status: transformations.TestRunStatusPass},
								},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "libraries pass and one transformation fails",
			results: &TestResults{
				Libraries: []transformations.LibraryTestResult{
					{HandleName: "lib1", Pass: true},
				},
				Transformations: []*TransformationTestWithDefinitions{
					{
						Result: &transformations.TransformationTestResult{
							TestSuiteResult: transformations.TestSuiteRunResult{
								Results: []transformations.TestResult{
									{Name: "test1", Status: transformations.TestRunStatusFail},
								},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "multiple transformations - only one has a failure",
			results: &TestResults{
				Transformations: []*TransformationTestWithDefinitions{
					{
						Result: &transformations.TransformationTestResult{
							TestSuiteResult: transformations.TestSuiteRunResult{
								Results: []transformations.TestResult{
									{Name: "test1", Status: transformations.TestRunStatusPass},
								},
							},
						},
					},
					{
						Result: &transformations.TransformationTestResult{
							TestSuiteResult: transformations.TestSuiteRunResult{
								Results: []transformations.TestResult{
									{Name: "test2", Status: transformations.TestRunStatusFail},
								},
							},
						},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.results.HasFailures())
		})
	}
}

func TestTestResults_DefaultSuiteTransformationNames(t *testing.T) {
	tests := []struct {
		name     string
		results  *TestResults
		expected []string
	}{
		{
			name:     "returns empty when no transformations exist in test results",
			results:  &TestResults{},
			expected: []string{},
		},
		{
			name: "excludes transformation with custom test suite",
			results: &TestResults{
				Transformations: []*TransformationTestWithDefinitions{
					{
						Result: &transformations.TransformationTestResult{Name: "my-transform"},
						Definitions: []*transformations.TestDefinition{
							{Name: "custom-suite"},
						},
					},
				},
			},
			expected: []string{},
		},
		{
			name: "excludes transformation with multiple test suites even when one is default-events",
			results: &TestResults{
				Transformations: []*TransformationTestWithDefinitions{
					{
						Result: &transformations.TransformationTestResult{Name: "my-transform"},
						Definitions: []*transformations.TestDefinition{
							{Name: "default-events"},
							{Name: "custom-suite"},
						},
					},
				},
			},
			expected: []string{},
		},
		{
			name: "excludes transformation with empty test definitions",
			results: &TestResults{
				Transformations: []*TransformationTestWithDefinitions{
					{
						Result:      &transformations.TransformationTestResult{Name: "my-transform"},
						Definitions: []*transformations.TestDefinition{},
					},
				},
			},
			expected: []string{},
		},
		{
			name: "returns only transformations with single default-events suite",
			results: &TestResults{
				Transformations: []*TransformationTestWithDefinitions{
					{
						Result: &transformations.TransformationTestResult{Name: "default-only"},
						Definitions: []*transformations.TestDefinition{
							{Name: "default-events"},
						},
					},
					{
						Result: &transformations.TransformationTestResult{Name: "custom-only"},
						Definitions: []*transformations.TestDefinition{
							{Name: "my-custom-suite"},
						},
					},
					{
						Result: &transformations.TransformationTestResult{Name: "multi-suite"},
						Definitions: []*transformations.TestDefinition{
							{Name: "default-events"},
							{Name: "extra-suite"},
						},
					},
					{
						Result: &transformations.TransformationTestResult{Name: "another-default"},
						Definitions: []*transformations.TestDefinition{
							{Name: "default-events"},
						},
					},
				},
			},
			expected: []string{"default-only", "another-default"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.results.DefaultSuiteTransformationNames())
		})
	}
}
