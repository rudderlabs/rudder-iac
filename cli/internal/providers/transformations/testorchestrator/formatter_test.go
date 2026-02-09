package testorchestrator

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

// Helper function to wrap test results with definitions
func wrapResults(result *transformations.TransformationTestResult, definitions []transformations.TestDefinition) *TransformationTestWithDefinitions {
	return &TransformationTestWithDefinitions{
		Result:      result,
		Definitions: definitions,
	}
}

func TestFormatter_Display_AllPassed(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	formatter := NewFormatter(false)
	results := &TestResults{
		Transformations: []*TransformationTestWithDefinitions{
			wrapResults(&transformations.TransformationTestResult{
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
	assert.Contains(t, output, "Testing transformations")
	assert.Contains(t, output, "test-transformation")
	assert.Contains(t, output, "✓ test-case-1")
	assert.Contains(t, output, "✓ test-case-2")
	assert.Contains(t, output, "2 passed")
	assert.Contains(t, output, "2 total")
	assert.NotContains(t, output, "Use --verbose")
}

func TestFormatter_Display_WithFailures(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	formatter := NewFormatter(false)
	results := &TestResults{
		Transformations: []*TransformationTestWithDefinitions{
			wrapResults(&transformations.TransformationTestResult{
				Name: "test-transformation",
				TestSuiteResult: transformations.TestSuiteRunResult{
					Status: transformations.TestRunStatusFail,
					Results: []transformations.TestResult{
						{Name: "test-case-1", Status: transformations.TestRunStatusPass},
						{Name: "test-case-2", Status: transformations.TestRunStatusFail, ActualOutput: []any{map[string]any{"foo": "bar"}}},
					},
				},
			}, nil),
		},
	}

	formatter.Display(results)

	output := buf.String()
	assert.Contains(t, output, "✓ test-case-1")
	assert.Contains(t, output, "✗ test-case-2")
	assert.Contains(t, output, "1 passed")
	assert.Contains(t, output, "1 failed")
	assert.Contains(t, output, "2 total")
	assert.Contains(t, output, "Use --verbose to see detailed output")
}

func TestFormatter_FormatJSON(t *testing.T) {
	formatter := NewFormatter(false)

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

func TestFormatter_Display_VerboseMode_WithExpectedOutput(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	formatter := NewFormatter(true)

	// Define test definitions with expected output
	testDefinitions := []transformations.TestDefinition{
		{
			Name:           "test-with-expected",
			Input:          []any{map[string]any{"type": "track"}},
			ExpectedOutput: []any{map[string]any{"status": "success"}},
		},
	}

	results := &TestResults{
		Transformations: []*TransformationTestWithDefinitions{
			wrapResults(&transformations.TransformationTestResult{
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
	assert.Contains(t, output, "Expected:")
	assert.Contains(t, output, "success")
	assert.Contains(t, output, "Actual:")
	assert.Contains(t, output, "failed")
	assert.Contains(t, output, "Diff:")
}
