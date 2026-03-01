package transformations

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

func wrapResults(result *transformations.TransformationTestResult, definitions []*transformations.TestDefinition) *model.TransformationTestWithDefinitions {
	return &model.TransformationTestWithDefinitions{
		Result:      result,
		Definitions: definitions,
	}
}

func TestIndent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		level    int
		expected string
	}{
		{
			name:     "level 1",
			input:    "test",
			level:    1,
			expected: "  test",
		},
		{
			name:     "level 2",
			input:    "test",
			level:    2,
			expected: "    test",
		},
		{
			name:     "level 3",
			input:    "test",
			level:    3,
			expected: "      test",
		},
		{
			name:     "level 0 or invalid",
			input:    "test",
			level:    0,
			expected: "test",
		},
		{
			name:     "negative level",
			input:    "test",
			level:    -1,
			expected: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indent(tt.input, tt.level)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCenter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		expected string
	}{
		{
			name:     "center short string",
			input:    "test",
			width:    10,
			expected: "   test   ",
		},
		{
			name:     "center with odd padding",
			input:    "abc",
			width:    10,
			expected: "   abc    ",
		},
		{
			name:     "string equals width",
			input:    "test",
			width:    4,
			expected: "test",
		},
		{
			name:     "string longer than width",
			input:    "testing",
			width:    4,
			expected: "testing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := center(tt.input, tt.width)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPrintIndentedLine(t *testing.T) {
	tests := []struct {
		name     string
		msg      string
		level    int
		newLine  bool
		expected string
	}{
		{
			name:     "with newline level 1",
			msg:      "test message",
			level:    1,
			newLine:  true,
			expected: "  test message\n",
		},
		{
			name:     "without newline level 2",
			msg:      "test message",
			level:    2,
			newLine:  false,
			expected: "    test message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			ui.SetWriter(&buf)
			defer ui.RestoreWriter()

			printIndentedLine(tt.msg, tt.level, tt.newLine)
			assert.Equal(t, tt.expected, buf.String())
		})
	}
}

func TestPrintSection(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	printSection("TEST TITLE", 10)

	output := buf.String()
	assert.Contains(t, output, "TEST TITLE")
	assert.Contains(t, output, "──────────")
}

func TestPrintSectionSummary(t *testing.T) {
	tests := []struct {
		name     string
		total    int
		metrics  []string
		expected []string
	}{
		{
			name:    "single metric",
			total:   10,
			metrics: []string{"Passed  5"},
			expected: []string{
				"Total  10",
				"•",
				"Passed  5",
			},
		},
		{
			name:    "multiple metrics",
			total:   20,
			metrics: []string{"Passed  10", "Failed  5", "Skipped  5"},
			expected: []string{
				"Total  20",
				"•",
				"Passed  10",
				"Failed  5",
				"Skipped  5",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			ui.SetWriter(&buf)
			defer ui.RestoreWriter()

			printSectionSummary(tt.total, tt.metrics...)
			output := buf.String()

			for _, exp := range tt.expected {
				assert.Contains(t, output, exp)
			}
		})
	}
}

func TestPrintBox(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	lines := []string{"line 1", "line 2", "line 3"}
	printBox(lines, 30)

	output := buf.String()
	assert.Contains(t, output, "┌")
	assert.Contains(t, output, "└")
	assert.Contains(t, output, "│ line 1")
	assert.Contains(t, output, "│ line 2")
	assert.Contains(t, output, "│ line 3")
}

func TestSuiteCounterTotal(t *testing.T) {
	counter := suiteCounter{
		passed:     5,
		mismatched: 3,
		errored:    2,
	}
	assert.Equal(t, 10, counter.total())
}

func TestLibraryCounterTotal(t *testing.T) {
	counter := libraryCounter{
		passed:  8,
		errored: 2,
	}
	assert.Equal(t, 10, counter.total())
}

func TestFormatJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "nil value",
			input:    nil,
			expected: "null",
		},
		{
			name:  "simple object",
			input: map[string]any{"key": "value"},
			expected: `{
  "key": "value"
}`,
		},
		{
			name:  "array",
			input: []any{"item1", "item2"},
			expected: `[
  "item1",
  "item2"
]`,
		},
		{
			name:  "nested object",
			input: map[string]any{"outer": map[string]any{"inner": "value"}},
			expected: `{
  "outer": {
    "inner": "value"
  }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateDiff(t *testing.T) {
	expected := `[
  {
    "anonymousId": "sample_anonymous_id",
    "context": {
      "app": {
        "name": "RudderLabs JavaScript SDK",
        "namespace": "com.rudderlabs.javascript",
        "version": "3.7.6"
      },
    },
    "event": "Product Click",
    "integrations": {
      "All": true
    },
    "messageId": "1",
    "type": "track",
  }
]`
	actual := `[
  {
    "anonymousId": "sample_anonymous_id",
    "context": {
      "app": {
        "installType": "npm",
        "name": "RudderLabs JavaScript SDK",
        "namespace": "com.rudderlabs.javascript",
        "version": "3.7.6"
      },
    },
    "event": "Product Click",
    "integrations": {
      "All": true
    },
    "messageId": "1",
    "type": "track",
  }
]`

	diff := generateDiff(expected, actual)
	assert.NotEmpty(t, diff)
	assert.Contains(t, diff, "Expected")
	assert.Contains(t, diff, "Actual")
	assert.Contains(t, diff, "installType")
}

func TestBuildExpectedOutputMap(t *testing.T) {
	definitions := []*transformations.TestDefinition{
		{
			Name:           "test1",
			ExpectedOutput: []any{map[string]any{"result": "ok"}},
		},
		{
			Name:           "test2",
			ExpectedOutput: []any{map[string]any{"result": "failed"}},
		},
	}

	result := buildExpectedOutputMap(definitions)

	assert.Len(t, result, 2)
	assert.Contains(t, result, "test1")
	assert.Contains(t, result, "test2")
	assert.Equal(t, definitions[0].ExpectedOutput, result["test1"])
	assert.Equal(t, definitions[1].ExpectedOutput, result["test2"])
}

func TestNewResultDisplayer(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
	}{
		{
			name:    "verbose mode",
			verbose: true,
		},
		{
			name:    "non-verbose mode",
			verbose: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			displayer := NewResultDisplayer(tt.verbose)
			assert.NotNil(t, displayer)
			assert.Equal(t, tt.verbose, displayer.verbose)
		})
	}
}

func TestResultDisplayer_Display_AllPassed(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	formatter := NewResultDisplayer(false)
	results := &model.TestResults{
		Transformations: []*model.TransformationTestWithDefinitions{
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
	assert.Contains(t, output, "TRANSFORMATION TEST SUITE")
	assert.Contains(t, output, "TEST SUITES")
	assert.Contains(t, output, "test-transformation")
	assert.Contains(t, output, "✓  test-case-1")
	assert.Contains(t, output, "✓  test-case-2")
	assert.Contains(t, output, "Total  2")
	assert.Contains(t, output, "Passed  2")
	assert.Contains(t, output, "Mismatch errors 0")
	assert.Contains(t, output, "Execution errors  0")
	assert.NotContains(t, output, "FAILURE DETAILS")
}

func TestResultDisplayer_Display_WithFailures_NonVerbose(t *testing.T) {
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

	results := &model.TestResults{
		Transformations: []*model.TransformationTestWithDefinitions{
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
	assert.Contains(t, output, "✓  test-case-1")
	assert.Contains(t, output, "⊗  test-with-expected")
	assert.Contains(t, output, "Total  2")
	assert.Contains(t, output, "Passed  1")
	assert.Contains(t, output, "Mismatch errors 1")
	assert.Contains(t, output, "FAILURE DETAILS")
	assert.Contains(t, output, "test-transformation  ›  test-with-expected")
	assert.Contains(t, output, "Actual output mismatched from expected output")
	assert.Contains(t, output, "Tip: run with --verbose to see full event diffs")
	assert.NotContains(t, output, "success")
	assert.NotContains(t, output, "failed")
}

func TestResultDisplayer_Display_WithFailures_Verbose(t *testing.T) {
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

	results := &model.TestResults{
		Transformations: []*model.TransformationTestWithDefinitions{
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
	assert.Contains(t, output, "FAILURE DETAILS")
	assert.Contains(t, output, "test-transformation  ›  test-with-expected")
	assert.Contains(t, output, "│")
	assert.Contains(t, output, "success")
	assert.Contains(t, output, "failed")
	assert.NotContains(t, output, "Tip:")
}

func TestResultDisplayer_Display_WithImportedLibraries(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	formatter := NewResultDisplayer(false)
	results := &model.TestResults{
		Transformations: []*model.TransformationTestWithDefinitions{
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
	assert.Contains(t, output, "imported libraries")
	assert.Contains(t, output, "myLib, anotherLib")
}

func TestResultDisplayer_Display_WithErrors(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	formatter := NewResultDisplayer(false)
	results := &model.TestResults{
		Transformations: []*model.TransformationTestWithDefinitions{
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
								{
									Message:    "Transformation execution failed",
									EventIndex: 0,
								},
							},
						},
					},
				},
			}, nil),
		},
	}

	formatter.Display(results)

	output := buf.String()
	assert.Contains(t, output, "✕  error-test")
	assert.Contains(t, output, "Execution errors  1")
	assert.Contains(t, output, "FAILURE DETAILS")
	assert.Contains(t, output, "error-transformation  ›  error-test")
	assert.Contains(t, output, "│")
	assert.Contains(t, output, "Transformation execution failed")
	assert.Contains(t, output, "errored input event can be found at index 0")
}

func TestResultDisplayer_Display_WithErrors_Verbose(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	formatter := NewResultDisplayer(true)
	results := &model.TestResults{
		Transformations: []*model.TransformationTestWithDefinitions{
			wrapResults(&transformations.TransformationTestResult{
				ID:   "tr-error-verbose",
				Name: "error-transformation",
				TestSuiteResult: transformations.TestSuiteRunResult{
					Status: transformations.TestRunStatusError,
					Results: []transformations.TestResult{
						{
							Name:   "error-test",
							Status: transformations.TestRunStatusError,
							Errors: []transformations.TestError{
								{
									Message:    "Transformation execution failed",
									EventIndex: 0,
									Event:      map[string]any{"type": "track", "event": "Error Event"},
								},
							},
						},
					},
				},
			}, nil),
		},
	}

	formatter.Display(results)

	output := buf.String()
	assert.Contains(t, output, "FAILURE DETAILS")
	assert.Contains(t, output, "Transformation execution failed")
	assert.Contains(t, output, "errored input event can be found at index 0")
	assert.Contains(t, output, "track")
	assert.Contains(t, output, "Error Event")
}

func TestResultDisplayer_Display_WithLibraries(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	formatter := NewResultDisplayer(false)
	results := &model.TestResults{
		Libraries: []transformations.LibraryTestResult{
			{HandleName: "lib-pass", Pass: true},
			{HandleName: "lib-fail", Pass: false, Message: "Syntax error on line 5"},
		},
	}

	formatter.Display(results)

	output := buf.String()
	assert.Contains(t, output, "LIBRARIES")
	assert.Contains(t, output, "lib-pass")
	assert.Contains(t, output, "syntax ok")
	assert.Contains(t, output, "lib-fail")
	assert.Contains(t, output, "syntax error")
	assert.Contains(t, output, "Syntax error on line 5")
	assert.Contains(t, output, "Total  2")
	assert.Contains(t, output, "Passed  1")
	assert.Contains(t, output, "Errored  1")
}

func TestResultDisplayer_Display_EmptyResults(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	formatter := NewResultDisplayer(false)
	results := &model.TestResults{}

	formatter.Display(results)

	output := buf.String()
	assert.Contains(t, output, "TRANSFORMATION TEST SUITE")
	assert.Contains(t, output, "SUMMARY")
	assert.NotContains(t, output, "LIBRARIES")
	assert.NotContains(t, output, "TEST SUITES")
}

func TestResultDisplayer_Display_MixedResults(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	formatter := NewResultDisplayer(false)
	results := &model.TestResults{
		Libraries: []transformations.LibraryTestResult{
			{HandleName: "lib-1", Pass: true},
			{HandleName: "lib-2", Pass: false, Message: "error"},
		},
		Transformations: []*model.TransformationTestWithDefinitions{
			wrapResults(&transformations.TransformationTestResult{
				ID:   "tr-1",
				Name: "transformation-1",
				TestSuiteResult: transformations.TestSuiteRunResult{
					Status: transformations.TestRunStatusPass,
					Results: []transformations.TestResult{
						{Name: "test-1", Status: transformations.TestRunStatusPass},
					},
				},
			}, nil),
		},
	}

	formatter.Display(results)

	output := buf.String()
	assert.Contains(t, output, "LIBRARIES")
	assert.Contains(t, output, "TEST SUITES")
	assert.Contains(t, output, "SUMMARY")
	assert.Contains(t, output, "Libraries")
	assert.Contains(t, output, "Suites")
}

func TestResultDisplayer_Display_MultilineErrorMessage(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	formatter := NewResultDisplayer(false)
	results := &model.TestResults{
		Transformations: []*model.TransformationTestWithDefinitions{
			wrapResults(&transformations.TransformationTestResult{
				ID:   "tr-multiline",
				Name: "test-transformation",
				TestSuiteResult: transformations.TestSuiteRunResult{
					Status: transformations.TestRunStatusError,
					Results: []transformations.TestResult{
						{
							Name:   "error-test",
							Status: transformations.TestRunStatusError,
							Errors: []transformations.TestError{
								{
									Message:    "Line 1 error\nLine 2 error\nLine 3 error",
									EventIndex: 2,
								},
							},
						},
					},
				},
			}, nil),
		},
	}

	formatter.Display(results)

	output := buf.String()
	assert.Contains(t, output, "Line 1 error")
	assert.Contains(t, output, "Line 2 error")
	assert.Contains(t, output, "Line 3 error")
	assert.Contains(t, output, "errored input event can be found at index 2")
}

func TestResultDisplayer_Display_NoExpectedOutputInVerboseMode(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	formatter := NewResultDisplayer(true)
	results := &model.TestResults{
		Transformations: []*model.TransformationTestWithDefinitions{
			wrapResults(&transformations.TransformationTestResult{
				ID:   "tr-no-expected",
				Name: "test-transformation",
				TestSuiteResult: transformations.TestSuiteRunResult{
					Status: transformations.TestRunStatusFail,
					Results: []transformations.TestResult{
						{
							Name:         "test-no-expected",
							Status:       transformations.TestRunStatusFail,
							ActualOutput: []any{map[string]any{"status": "failed"}},
						},
					},
				},
			}, nil),
		},
	}

	formatter.Display(results)

	output := buf.String()
	assert.Contains(t, output, "Actual output mismatched from expected output")
	assert.NotContains(t, output, "failed")
}

func TestResultDisplayer_PrintSummary_NoResults(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	formatter := NewResultDisplayer(false)
	results := &model.TestResults{}

	formatter.Display(results)

	output := buf.String()
	assert.Contains(t, output, "SUMMARY")
	assert.NotContains(t, output, "Libraries")
	assert.NotContains(t, output, "Suites")
}
