package transformations

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

// Helper function to wrap test results with definitions
func wrapResults(result *transformations.TransformationTestResult, definitions []*transformations.TestDefinition) *TransformationTestWithDefinitions {
	return &TransformationTestWithDefinitions{
		Result:      result,
		Definitions: definitions,
	}
}

func TestFormatter_Display_AllPassed(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	formatter := NewResultDisplayer(false)
	results := &TestResults{
		Transformations: []*TransformationTestWithDefinitions{
			wrapResults(&transformations.TransformationTestResult{
				ID:   "tr-123",
				Name: "test-transformation",
				TestSuiteResult: transformations.TestSuiteRunResult{
					Status: transformations.TestRunStatusPass,
					Results: []transformations.TestResult{
						{Name: "test-case-1", Status: transformations.TestRunStatusPass},
						{Name: "test-case-2", Status: transformations.TestRunStatusPass},
					},
				},
			}, nil),
		},
	}

	formatter.Display(results)

	output := buf.String()
	assert.Contains(t, output, "Transformation Test Results")
	assert.Contains(t, output, "test-transformation")
	assert.Contains(t, output, "✓ test-case-1")
	assert.Contains(t, output, "✓ test-case-2")
	assert.Contains(t, output, "2 passed")
	assert.Contains(t, output, "2 total")
	assert.NotContains(t, output, "Failure Details")
}

func TestFormatter_Display_WithFailures_NonVerbose(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	formatter := NewResultDisplayer(false)
	testDefinitions := []*transformations.TestDefinition{
		{
			Name:           "test-with-expected",
			ExpectedOutput: []any{map[string]any{"status": "success"}},
		},
	}

	results := &TestResults{
		Transformations: []*TransformationTestWithDefinitions{
			wrapResults(&transformations.TransformationTestResult{
				ID:   "tr-456",
				Name: "test-transformation",
				TestSuiteResult: transformations.TestSuiteRunResult{
					Status: transformations.TestRunStatusFail,
					Results: []transformations.TestResult{
						{Name: "test-case-1", Status: transformations.TestRunStatusPass},
						{
							Name:         "test-with-expected",
							Status:       transformations.TestRunStatusFail,
							ActualOutput: []any{map[string]any{"status": "failed"}},
						},
					},
				},
			}, testDefinitions),
		},
	}

	formatter.Display(results)

	output := buf.String()
	assert.Contains(t, output, "✓ test-case-1")
	assert.Contains(t, output, "✗ test-with-expected")
	assert.Contains(t, output, "1 passed")
	assert.Contains(t, output, "1 failed")
	assert.Contains(t, output, "2 total")
	assert.Contains(t, output, "Failure Details")
	assert.Contains(t, output, "test-transformation > test-with-expected")
	assert.Contains(t, output, "Actual output mismatched from expected output")
	assert.Contains(t, output, "Use --verbose to see additional event details")
	// Should NOT show diff/expected/actual in non-verbose mode
	assert.NotContains(t, output, "Diff:")
	assert.NotContains(t, output, `"success"`)
	assert.NotContains(t, output, `"failed"`)
}

func TestFormatter_Display_WithFailures_Verbose(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	formatter := NewResultDisplayer(true)
	testDefinitions := []*transformations.TestDefinition{
		{
			Name:           "test-with-expected",
			ExpectedOutput: []any{map[string]any{"status": "success"}},
		},
	}

	results := &TestResults{
		Transformations: []*TransformationTestWithDefinitions{
			wrapResults(&transformations.TransformationTestResult{
				ID:   "tr-verbose",
				Name: "test-transformation",
				TestSuiteResult: transformations.TestSuiteRunResult{
					Status: transformations.TestRunStatusFail,
					Results: []transformations.TestResult{
						{
							Name:         "test-with-expected",
							Status:       transformations.TestRunStatusFail,
							ActualOutput: []any{map[string]any{"status": "failed"}},
						},
					},
				},
			}, testDefinitions),
		},
	}

	formatter.Display(results)

	output := buf.String()
	assert.Contains(t, output, "Failure Details")
	assert.Contains(t, output, "test-transformation > test-with-expected")
	assert.Contains(t, output, "Diff:")
	assert.Contains(t, output, "success")
	assert.Contains(t, output, "failed")
	// Should NOT show the hint in verbose mode
	assert.NotContains(t, output, "Use --verbose")
}

func TestFormatter_Display_WithImportedLibraries(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	formatter := NewResultDisplayer(false)
	results := &TestResults{
		Transformations: []*TransformationTestWithDefinitions{
			wrapResults(&transformations.TransformationTestResult{
				ID:      "tr-789",
				Name:    "transformation-with-libs",
				Imports: []string{"myLib", "anotherLib"},
				TestSuiteResult: transformations.TestSuiteRunResult{
					Status: transformations.TestRunStatusPass,
					Results: []transformations.TestResult{
						{Name: "test-case-1", Status: transformations.TestRunStatusPass},
					},
				},
			}, nil),
		},
	}

	formatter.Display(results)

	output := buf.String()
	assert.Contains(t, output, "transformation-with-libs")
	assert.Contains(t, output, "Imported libraries")
	assert.Contains(t, output, "myLib, anotherLib")
	assert.Contains(t, output, "Tests: 1 total")
}

func TestFormatter_Display_WithErrors(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	formatter := NewResultDisplayer(false)
	results := &TestResults{
		Transformations: []*TransformationTestWithDefinitions{
			wrapResults(&transformations.TransformationTestResult{
				ID:   "tr-error",
				Name: "error-transformation",
				TestSuiteResult: transformations.TestSuiteRunResult{
					Status: transformations.TestRunStatusError,
					Results: []transformations.TestResult{
						{
							Name:   "error-test",
							Status: transformations.TestRunStatusError,
							Errors: []transformations.TestError{
								{Message: "Transformation execution failed"},
							},
						},
					},
				},
			}, nil),
		},
	}

	formatter.Display(results)

	output := buf.String()
	assert.Contains(t, output, "✕ error-test")
	assert.Contains(t, output, "1 error")
	assert.Contains(t, output, "Failure Details")
	assert.Contains(t, output, "error-transformation > error-test")
	assert.Contains(t, output, "Error:")
	assert.Contains(t, output, "Transformation execution failed")
}

func TestFormatter_Display_WithLibraries(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	formatter := NewResultDisplayer(false)
	results := &TestResults{
		Libraries: []transformations.LibraryTestResult{
			{HandleName: "lib-pass", Pass: true},
			{HandleName: "lib-fail", Pass: false, Message: "Syntax error"},
		},
	}

	formatter.Display(results)

	output := buf.String()
	assert.Contains(t, output, "Libraries")
	assert.Contains(t, output, "✓ lib-pass")
	assert.Contains(t, output, "✗ lib-fail: Syntax error")
	assert.Contains(t, output, "1 failed")
}

func TestFormatter_FormatJSON(t *testing.T) {
	formatter := NewResultDisplayer(false)

	t.Run("nil value", func(t *testing.T) {
		result := formatter.formatJSON(nil)
		assert.Equal(t, "null", result)
	})

	t.Run("simple object", func(t *testing.T) {
		obj := map[string]any{"key": "value"}
		result := formatter.formatJSON(obj)
		assert.Contains(t, result, "key")
		assert.Contains(t, result, "value")
	})

	t.Run("array", func(t *testing.T) {
		arr := []any{"item1", "item2"}
		result := formatter.formatJSON(arr)
		assert.Contains(t, result, "item1")
		assert.Contains(t, result, "item2")
	})
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		expected string
	}{
		{"zero", 0, "s"},
		{"one", 1, ""},
		{"two", 2, "s"},
		{"many", 100, "s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pluralize(tt.count)
			assert.Equal(t, tt.expected, result)
		})
	}
}
