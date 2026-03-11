package display

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/testorchestrator"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

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

	printSection("TEST TITLE")

	output := buf.String()
	assert.Contains(t, output, "TEST TITLE")
	assert.Contains(t, output, "--------")
}

func TestPrintSeparator(t *testing.T) {
	tests := []struct {
		name string
		char string
	}{
		{
			name: "dash separator",
			char: "-",
		},
		{
			name: "equals separator",
			char: "=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			ui.SetWriter(&buf)
			defer ui.RestoreWriter()

			printSeparator(tt.char)
			output := buf.String()

			assert.Contains(t, output, tt.char)
			// Should be lineWidth (80) characters
			assert.Equal(t, 81, len(output)) // 80 chars + newline
		})
	}
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
	expected := `{
  "status": "success"
}`
	actual := `{
  "status": "failed",
  "error": "test"
}`

	diff := generateDiff(expected, actual, 0)
	assert.NotEmpty(t, diff)
	assert.Contains(t, diff, "Expected")
	assert.Contains(t, diff, "Actual")
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
			assert.Equal(t, 0, displayer.suiteCounter.total())
			assert.Equal(t, 0, displayer.libraryCounter.total())
		})
	}
}

func TestResultDisplayer_Display_AllPassed(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	displayer := NewResultDisplayer(false)
	results := &testorchestrator.TestResults{
		Transformations: []*testorchestrator.TransformationTestWithDefinitions{
			{
				Result: &transformations.TransformationTestResult{
					Name: "test-transformation",
					TestSuiteResult: transformations.TestSuiteRunResult{
						Results: []transformations.TestResult{
							{Name: "test-case-1", Status: transformations.TestRunStatusPass},
							{Name: "test-case-2", Status: transformations.TestRunStatusPass},
						},
					},
				},
			},
		},
	}

	displayer.Display(results)

	output := buf.String()
	assert.Contains(t, output, sectionTransformationTests)
	assert.Contains(t, output, "test-transformation")
	assert.Contains(t, output, symbolPass)
	assert.Contains(t, output, "test-case-1")
	assert.Contains(t, output, "test-case-2")
	assert.Contains(t, output, testStatusPassed)
	assert.Contains(t, output, sectionSummary)
	assert.Contains(t, output, labelResultPassed)
}

func TestResultDisplayer_Display_WithMismatchFailures(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	displayer := NewResultDisplayer(false)
	testDefinitions := []*transformations.TestDefinition{
		{
			Name:           "test-mismatch",
			ExpectedOutput: []any{map[string]any{"status": "success"}},
		},
	}

	results := &testorchestrator.TestResults{
		Transformations: []*testorchestrator.TransformationTestWithDefinitions{
			{
				Result: &transformations.TransformationTestResult{
					Name: "test-transformation",
					TestSuiteResult: transformations.TestSuiteRunResult{
						Results: []transformations.TestResult{
							{Name: "test-pass", Status: transformations.TestRunStatusPass},
							{
								Name:         "test-mismatch",
								Status:       transformations.TestRunStatusFail,
								ActualOutput: []any{map[string]any{"status": "failed"}},
							},
						},
					},
				},
				Definitions: testDefinitions,
			},
		},
	}

	displayer.Display(results)

	output := buf.String()
	assert.Contains(t, output, symbolMismatch)
	assert.Contains(t, output, "test-mismatch")
	assert.Contains(t, output, testStatusMismatch)
	assert.Contains(t, output, sectionFailures)
	assert.Contains(t, output, failureTypeOutputMismatch)
	assert.Contains(t, output, labelResultFailed)
}

func TestResultDisplayer_Display_WithExecutionErrors(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	displayer := NewResultDisplayer(false)
	results := &testorchestrator.TestResults{
		Transformations: []*testorchestrator.TransformationTestWithDefinitions{
			{
				Result: &transformations.TransformationTestResult{
					Name: "error-transformation",
					TestSuiteResult: transformations.TestSuiteRunResult{
						Results: []transformations.TestResult{
							{
								Name:   "error-test",
								Status: transformations.TestRunStatusError,
								Errors: []transformations.TestError{
									{
										Message:    "Execution failed",
										EventIndex: 0,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	displayer.Display(results)

	output := buf.String()
	assert.Contains(t, output, symbolError)
	assert.Contains(t, output, "error-test")
	assert.Contains(t, output, testStatusError)
	assert.Contains(t, output, sectionFailures)
	assert.Contains(t, output, failureTypeExecutionError)
	assert.Contains(t, output, "Execution failed")
}

func TestResultDisplayer_Display_WithMultilineErrors(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	displayer := NewResultDisplayer(false)
	results := &testorchestrator.TestResults{
		Transformations: []*testorchestrator.TransformationTestWithDefinitions{
			{
				Result: &transformations.TransformationTestResult{
					Name: "test-transformation",
					TestSuiteResult: transformations.TestSuiteRunResult{
						Results: []transformations.TestResult{
							{
								Name:   "error-test",
								Status: transformations.TestRunStatusError,
								Errors: []transformations.TestError{
									{
										Message:    "Line 1 error\nLine 2 error\nLine 3 stack\nLine 4 stack",
										EventIndex: 2,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	displayer.Display(results)

	output := buf.String()
	assert.Contains(t, output, "Line 1 error")
	assert.Contains(t, output, "Line 2 error")
	// Stack trace should not be shown in non-verbose mode
	assert.NotContains(t, output, "Line 3 stack")
}

func TestResultDisplayer_Display_WithMultilineErrors_Verbose(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	displayer := NewResultDisplayer(true)
	results := &testorchestrator.TestResults{
		Transformations: []*testorchestrator.TransformationTestWithDefinitions{
			{
				Result: &transformations.TransformationTestResult{
					Name: "test-transformation",
					TestSuiteResult: transformations.TestSuiteRunResult{
						Results: []transformations.TestResult{
							{
								Name:   "error-test",
								Status: transformations.TestRunStatusError,
								Errors: []transformations.TestError{
									{
										Message:    "Line 1 error\nLine 2 error\nLine 3 stack\nLine 4 stack",
										EventIndex: 2,
										Event:      map[string]any{"type": "track"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	displayer.Display(results)

	output := buf.String()
	assert.Contains(t, output, "Line 1 error")
	assert.Contains(t, output, "Line 2 error")
	assert.Contains(t, output, labelFullStackTrace)
	assert.Contains(t, output, "Line 3 stack")
	assert.Contains(t, output, "Line 4 stack")
	assert.Contains(t, output, "track")
}

func TestResultDisplayer_Display_WithGroupedErrors(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	displayer := NewResultDisplayer(false)
	results := &testorchestrator.TestResults{
		Transformations: []*testorchestrator.TransformationTestWithDefinitions{
			{
				Result: &transformations.TransformationTestResult{
					Name: "test-transformation",
					TestSuiteResult: transformations.TestSuiteRunResult{
						Results: []transformations.TestResult{
							{
								Name:   "error-test",
								Status: transformations.TestRunStatusError,
								Errors: []transformations.TestError{
									{Message: "Same error", EventIndex: 0},
									{Message: "Same error", EventIndex: 1},
									{Message: "Different error", EventIndex: 2},
								},
							},
						},
					},
				},
			},
		},
	}

	displayer.Display(results)

	output := buf.String()
	assert.Contains(t, output, "Same error")
	assert.Contains(t, output, "Different error")
	// Should show grouped error occurrences
	assert.Contains(t, output, "2 times")
	assert.Contains(t, output, "1 time")
}

func TestResultDisplayer_Display_WithImportedLibraries(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	displayer := NewResultDisplayer(false)
	results := &testorchestrator.TestResults{
		Transformations: []*testorchestrator.TransformationTestWithDefinitions{
			{
				Result: &transformations.TransformationTestResult{
					Name:    "transformation-with-libs",
					Imports: []string{"myLib", "anotherLib"},
					TestSuiteResult: transformations.TestSuiteRunResult{
						Results: []transformations.TestResult{
							{Name: "test-1", Status: transformations.TestRunStatusPass},
						},
					},
				},
			},
		},
	}

	displayer.Display(results)

	output := buf.String()
	assert.Contains(t, output, "transformation-with-libs")
	assert.Contains(t, output, "myLib, anotherLib")
}

func TestResultDisplayer_DisplayLibraries_AllPassed(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	displayer := NewResultDisplayer(false)
	results := &testorchestrator.TestResults{
		Libraries: []transformations.LibraryTestResult{
			{HandleName: "lib-1", Pass: true},
			{HandleName: "lib-2", Pass: true},
		},
	}

	displayer.Display(results)

	output := buf.String()
	assert.Contains(t, output, sectionLibraryTests)
	assert.Contains(t, output, "lib-1")
	assert.Contains(t, output, "lib-2")
	assert.Contains(t, output, syntaxStatusOK)
	assert.Contains(t, output, symbolPass)
}

func TestResultDisplayer_DisplayLibraries_WithFailures(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	displayer := NewResultDisplayer(false)
	results := &testorchestrator.TestResults{
		Libraries: []transformations.LibraryTestResult{
			{HandleName: "lib-pass", Pass: true},
			{HandleName: "lib-fail", Pass: false, Message: "Syntax error on line 5"},
		},
	}

	displayer.Display(results)

	output := buf.String()
	assert.Contains(t, output, sectionLibraryTests)
	assert.Contains(t, output, "lib-pass")
	assert.Contains(t, output, "lib-fail")
	assert.Contains(t, output, syntaxStatusError)
	assert.Contains(t, output, "Syntax error on line 5")
	assert.Contains(t, output, symbolError)
}

func TestResultDisplayer_DisplayLibraries_WithMultilineError_Verbose(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	displayer := NewResultDisplayer(true)
	results := &testorchestrator.TestResults{
		Libraries: []transformations.LibraryTestResult{
			{HandleName: "lib-fail", Pass: false, Message: "Error line 1\nStack line 2\nStack line 3"},
		},
	}

	displayer.Display(results)

	output := buf.String()
	assert.Contains(t, output, "Error line 1")
	assert.Contains(t, output, labelFullStackTrace)
	assert.Contains(t, output, "Stack line 2")
	assert.Contains(t, output, "Stack line 3")
}

func TestResultDisplayer_Display_EmptyResults(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	displayer := NewResultDisplayer(false)
	results := &testorchestrator.TestResults{}

	displayer.Display(results)

	output := buf.String()
	assert.Contains(t, output, sectionSummary)
	assert.NotContains(t, output, sectionLibraryTests)
	assert.NotContains(t, output, sectionTransformationTests)
}

func TestResultDisplayer_Display_MixedResults(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	displayer := NewResultDisplayer(false)
	results := &testorchestrator.TestResults{
		Libraries: []transformations.LibraryTestResult{
			{HandleName: "lib-1", Pass: true},
		},
		Transformations: []*testorchestrator.TransformationTestWithDefinitions{
			{
				Result: &transformations.TransformationTestResult{
					Name: "transformation-1",
					TestSuiteResult: transformations.TestSuiteRunResult{
						Results: []transformations.TestResult{
							{Name: "test-1", Status: transformations.TestRunStatusPass},
						},
					},
				},
			},
		},
	}

	displayer.Display(results)

	output := buf.String()
	assert.Contains(t, output, sectionLibraryTests)
	assert.Contains(t, output, sectionTransformationTests)
	assert.Contains(t, output, sectionSummary)
}

func TestResultDisplayer_VerboseTipShownWhenNonVerbose(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	displayer := NewResultDisplayer(false)
	results := &testorchestrator.TestResults{}

	displayer.Display(results)

	output := buf.String()
	assert.Contains(t, output, labelVerboseTip)
}

func TestResultDisplayer_VerboseTipNotShownWhenVerbose(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.RestoreWriter()

	displayer := NewResultDisplayer(true)
	results := &testorchestrator.TestResults{}

	displayer.Display(results)

	output := buf.String()
	assert.NotContains(t, output, labelVerboseTip)
}

func TestGroupErrorsByMessage(t *testing.T) {
	errors := []transformations.TestError{
		{Message: "Error A", EventIndex: 0, Event: "event0"},
		{Message: "Error A", EventIndex: 1, Event: "event1"},
		{Message: "Error B", EventIndex: 2, Event: "event2"},
		{Message: "Error A", EventIndex: 3, Event: "event3"},
	}

	groups, order := groupErrorsByMessage(errors)

	assert.Len(t, groups, 2)
	assert.Len(t, order, 2)
	assert.Equal(t, "Error A", order[0])
	assert.Equal(t, "Error B", order[1])
	assert.Equal(t, []int{0, 1, 3}, groups["Error A"].indices)
	assert.Equal(t, []int{2}, groups["Error B"].indices)
	assert.Equal(t, "event0", groups["Error A"].event)
}
