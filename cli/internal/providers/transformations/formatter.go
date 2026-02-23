package transformations

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pmezard/go-difflib/difflib"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

const (
	indentLevel1  = "  "
	indentLevel2  = "    "
	indentLevel3  = "      "
	separatorLine = "------------------------------------------------------------"
)

// indent returns a string with the specified indentation level applied
func indent(level int, s string) string {
	switch level {
	case 1:
		return indentLevel1 + s
	case 2:
		return indentLevel2 + s
	case 3:
		return indentLevel3 + s
	default:
		return s
	}
}

// TransformationTestWithDefinitions combines test results with their original definitions
type TransformationTestWithDefinitions struct {
	Result      *transformations.TransformationTestResult
	Definitions []*transformations.TestDefinition
}

// TestResults contains the results of all test executions with their definitions
type TestResults struct {
	Pass            bool
	Message         string
	Libraries       []transformations.LibraryTestResult
	Transformations []*TransformationTestWithDefinitions
}

// Formatter formats and displays test results
type Formatter struct {
	verbose bool
}

// NewFormatter creates a new result formatter
func NewFormatter(verbose bool) *Formatter {
	return &Formatter{
		verbose: verbose,
	}
}

// failureDetail holds information about a failed test for detailed display
type failureDetail struct {
	transformationName string
	testName           string
	status             transformations.TestRunStatus
	expectedOutput     []any
	actualOutput       []any
	errors             []transformations.TestError
}

// testCounts tracks test result statistics
type testCounts struct {
	passed   int
	failed   int
	errors   int
	failures []failureDetail
}

// Display formats and displays test results
func (f *Formatter) Display(results *TestResults) {
	ui.Println(ui.Bold("\nTransformation Test Results\n"))

	counts := f.displayLibraries(results.Libraries)
	counts = f.displayTransformations(results.Transformations, counts)

	ui.Println(separatorLine)
	f.printSummary(counts)
	f.printFailures(counts.failures)
}

// displayLibraries displays library test results and returns updated counts
func (f *Formatter) displayLibraries(libraries []transformations.LibraryTestResult) testCounts {
	counts := testCounts{}

	if len(libraries) == 0 {
		return counts
	}

	ui.Printf("%s%s\n", indentLevel1, ui.Bold("Libraries"))
	for _, lib := range libraries {
		if lib.Pass {
			ui.Printf("%s%s %s\n", indentLevel2, ui.Color("✓", ui.ColorGreen), lib.HandleName)
			continue
		}

		msg := lib.HandleName
		if lib.Message != "" {
			msg = fmt.Sprintf("%s: %s", lib.HandleName, lib.Message)
		}
		ui.Printf("%s%s %s\n", indentLevel2, ui.Color("✗", ui.ColorRed), msg)
		counts.failed++
	}
	ui.Println()

	return counts
}

// displayTransformations displays transformation test results and returns updated counts
func (f *Formatter) displayTransformations(transformations []*TransformationTestWithDefinitions, counts testCounts) testCounts {
	for _, trWithDef := range transformations {
		expectedOutputMap := buildExpectedOutputMap(trWithDef.Definitions)
		counts = f.displayTransformation(trWithDef.Result, expectedOutputMap, counts)
	}
	return counts
}

// displayTransformation displays a single transformation's test results
func (f *Formatter) displayTransformation(
	result *transformations.TransformationTestResult,
	expectedOutputMap map[string][]any,
	counts testCounts,
) testCounts {
	ui.Printf("%s%s\n", indentLevel1, ui.Bold(result.Name))

	if len(result.Imports) > 0 {
		ui.Printf("%sImported libraries: %s\n", indentLevel2, ui.GreyedOut(strings.Join(result.Imports, ", ")))
	}

	testCount := len(result.TestSuiteResult.Results)
	ui.Printf("%sTests: %d total\n", indentLevel2, testCount)

	for _, testResult := range result.TestSuiteResult.Results {
		expectedOutput := expectedOutputMap[testResult.Name]
		counts = f.displayTestResult(result.Name, testResult, expectedOutput, counts)
	}

	ui.Println()
	return counts
}

// displayTestResult displays a single test result and updates counts
func (f *Formatter) displayTestResult(
	transformationName string,
	testResult transformations.TestResult,
	expectedOutput []any,
	counts testCounts,
) testCounts {
	switch testResult.Status {
	case transformations.TestRunStatusPass:
		ui.Printf("%s%s %s\n", indentLevel3, ui.Color("✓", ui.ColorGreen), testResult.Name)
		counts.passed++

	case transformations.TestRunStatusFail:
		ui.Printf("%s%s %s\n", indentLevel3, ui.Color("✗", ui.ColorYellow), testResult.Name)
		counts.failed++
		counts.failures = append(counts.failures, failureDetail{
			transformationName: transformationName,
			testName:           testResult.Name,
			status:             testResult.Status,
			expectedOutput:     expectedOutput,
			actualOutput:       testResult.ActualOutput,
			errors:             testResult.Errors,
		})

	case transformations.TestRunStatusError:
		ui.Printf("%s%s %s\n", indentLevel3, ui.Color("✕", ui.ColorRed), testResult.Name)
		counts.errors++
		counts.failures = append(counts.failures, failureDetail{
			transformationName: transformationName,
			testName:           testResult.Name,
			status:             testResult.Status,
			errors:             testResult.Errors,
		})
	}

	return counts
}

// printSummary prints the test summary with colored counts
func (f *Formatter) printSummary(counts testCounts) {
	total := counts.passed + counts.failed + counts.errors

	var parts []string

	if counts.passed > 0 {
		parts = append(parts, fmt.Sprintf("%s passed", ui.Color(fmt.Sprintf("%d", counts.passed), ui.ColorGreen)))
	}
	if counts.failed > 0 {
		parts = append(parts, fmt.Sprintf("%s failed", ui.Color(fmt.Sprintf("%d", counts.failed), ui.ColorYellow)))
	}
	if counts.errors > 0 {
		parts = append(parts, fmt.Sprintf("%s error%s", ui.Color(fmt.Sprintf("%d", counts.errors), ui.ColorRed), pluralize(counts.errors)))
	}

	summary := strings.Join(parts, ", ")
	ui.Printf("\nTests: %s, %d total\n", summary, total)
}

// printFailures prints detailed failure information
func (f *Formatter) printFailures(failures []failureDetail) {
	if len(failures) == 0 {
		return
	}

	ui.Println(ui.Bold("\nFailure Details:\n"))
	for _, failure := range failures {
		f.printFailureDetail(failure)
	}

	if !f.verbose {
		ui.Printf("\n%s\n", ui.GreyedOut("Use --verbose to see additional event details"))
	}
}

// printFailureDetail prints detailed information about a single failure
func (f *Formatter) printFailureDetail(detail failureDetail) {
	ui.Printf("%s\n", ui.Bold(fmt.Sprintf("%s > %s", detail.transformationName, detail.testName)))

	if detail.status == transformations.TestRunStatusError {
		f.printErrorDetail(detail)
		return
	}

	f.printFailureOutput(detail)
	f.printFailureErrors(detail)
	ui.Println()
}

// printErrorDetail prints error information for tests with error status
func (f *Formatter) printErrorDetail(detail failureDetail) {
	ui.Println(ui.Color(indent(1, "Error:"), ui.ColorRed))
	for _, err := range detail.errors {
		ui.Printf("%s%s\n", indentLevel2, err.Message)
		if f.verbose && err.Event != nil {
			ui.Printf("%sEvent: %s\n", indentLevel2, f.formatJSON(err.Event))
		}
	}
	ui.Println()
}

// printFailureOutput prints expected vs actual output comparison
func (f *Formatter) printFailureOutput(detail failureDetail) {
	if len(detail.expectedOutput) == 0 {
		return
	}

	if !f.verbose {
		ui.Println(ui.Color(indent(1, "Actual output mismatched from expected output"), ui.ColorYellow))
		return
	} else {

		diff := f.generateDiff(
			f.formatJSON(detail.expectedOutput),
			f.formatJSON(detail.actualOutput),
		)
		if diff != "" {
			ui.Println(ui.Color(indent(1, "Diff:"), ui.ColorYellow))
			ui.Print(indentMultiline(diff, indentLevel2))
		}
	}
}

// printFailureErrors prints error messages for failed tests (verbose mode only)
func (f *Formatter) printFailureErrors(detail failureDetail) {
	if len(detail.errors) == 0 || !f.verbose {
		return
	}

	ui.Println(ui.Color(indent(1, "Errors:"), ui.ColorYellow))
	for _, err := range detail.errors {
		ui.Printf("%s%s\n", indentLevel2, err.Message)
		if err.Event != nil {
			ui.Printf("%sEvent: %s\n", indentLevel2, f.formatJSON(err.Event))
		}
	}
}

// formatJSON converts any value to a pretty-printed JSON string
func (f *Formatter) formatJSON(v any) string {
	if v == nil {
		return "null"
	}

	jsonBytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("<error formatting JSON: %v>", err)
	}
	return string(jsonBytes)
}

// generateDiff generates a unified diff between expected and actual output
func (f *Formatter) generateDiff(expected, actual string) string {
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")

	diff := difflib.UnifiedDiff{
		A:        expectedLines,
		B:        actualLines,
		FromFile: "Expected",
		ToFile:   "Actual",
		Context:  3,
	}

	result, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		return ""
	}

	return result
}

// buildExpectedOutputMap creates a map from test name to expected output
func buildExpectedOutputMap(definitions []*transformations.TestDefinition) map[string][]any {
	expectedOutputMap := make(map[string][]any)
	for _, def := range definitions {
		expectedOutputMap[def.Name] = def.ExpectedOutput
	}
	return expectedOutputMap
}

// pluralize returns "s" if count != 1, otherwise empty string
func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// indentMultiline indents each line of a string with the given prefix
func indentMultiline(s, indent string) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line != "" || i < len(lines)-1 {
			lines[i] = indent + line
		}
	}
	return strings.TrimRight(strings.Join(lines, "\n"), " \t")
}
