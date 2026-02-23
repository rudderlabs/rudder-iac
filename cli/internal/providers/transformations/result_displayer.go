package transformations

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pmezard/go-difflib/difflib"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/testorchestrator"
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

// Type aliases to testorchestrator types for backwards compatibility
type (
	TransformationTestWithDefinitions = testorchestrator.TransformationTestWithDefinitions
	TestResults                       = testorchestrator.TestResults
)

// ResultDisplayer formats and displays test results
type ResultDisplayer struct {
	verbose bool
}

// NewResultDisplayer creates a new result displayer
func NewResultDisplayer(verbose bool) *ResultDisplayer {
	return &ResultDisplayer{
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
func (rd *ResultDisplayer) Display(results *TestResults) {
	ui.Println(ui.Bold("\nTransformation Test Results\n"))

	counts := rd.displayLibraries(results.Libraries)
	counts = rd.displayTransformations(results.Transformations, counts)

	ui.Println(separatorLine)
	rd.printSummary(counts)
	rd.printFailures(counts.failures)
}

// displayLibraries displays library test results and returns updated counts
func (rd *ResultDisplayer) displayLibraries(libraries []transformations.LibraryTestResult) testCounts {
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
func (rd *ResultDisplayer) displayTransformations(transformations []*TransformationTestWithDefinitions, counts testCounts) testCounts {
	for _, trWithDef := range transformations {
		expectedOutputMap := buildExpectedOutputMap(trWithDef.Definitions)
		counts = rd.displayTransformation(trWithDef.Result, expectedOutputMap, counts)
	}
	return counts
}

// displayTransformation displays a single transformation's test results
func (rd *ResultDisplayer) displayTransformation(
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
		counts = rd.displayTestResult(result.Name, testResult, expectedOutput, counts)
	}

	ui.Println()
	return counts
}

// displayTestResult displays a single test result and updates counts
func (rd *ResultDisplayer) displayTestResult(
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
func (rd *ResultDisplayer) printSummary(counts testCounts) {
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
func (rd *ResultDisplayer) printFailures(failures []failureDetail) {
	if len(failures) == 0 {
		return
	}

	ui.Println(ui.Bold("\nFailure Details:\n"))
	for _, failure := range failures {
		rd.printFailureDetail(failure)
	}

	if !rd.verbose {
		ui.Printf("\n%s\n", ui.GreyedOut("Use --verbose to see additional event details"))
	}
}

// printFailureDetail prints detailed information about a single failure
func (rd *ResultDisplayer) printFailureDetail(detail failureDetail) {
	ui.Printf("%s\n", ui.Bold(fmt.Sprintf("%s > %s", detail.transformationName, detail.testName)))

	if detail.status == transformations.TestRunStatusError {
		rd.printErrorDetail(detail)
		return
	}

	rd.printFailureOutput(detail)
	rd.printFailureErrors(detail)
	ui.Println()
}

// printErrorDetail prints error information for tests with error status
func (rd *ResultDisplayer) printErrorDetail(detail failureDetail) {
	ui.Println(ui.Color(indent(1, "Error:"), ui.ColorRed))
	for _, err := range detail.errors {
		ui.Printf("%s%s\n", indentLevel2, err.Message)
		if rd.verbose && err.Event != nil {
			ui.Printf("%sEvent: %s\n", indentLevel2, rd.formatJSON(err.Event))
		}
	}
	ui.Println()
}

// printFailureOutput prints expected vs actual output comparison
func (rd *ResultDisplayer) printFailureOutput(detail failureDetail) {
	if len(detail.expectedOutput) == 0 {
		return
	}

	if !rd.verbose {
		ui.Println(ui.Color(indent(1, "Actual output mismatched from expected output"), ui.ColorYellow))
		return
	} else {

		diff := rd.generateDiff(
			rd.formatJSON(detail.expectedOutput),
			rd.formatJSON(detail.actualOutput),
		)
		if diff != "" {
			ui.Println(ui.Color(indent(1, "Diff:"), ui.ColorYellow))
			ui.Print(indentMultiline(diff, indentLevel2))
		}
	}
}

// printFailureErrors prints error messages for failed tests (verbose mode only)
func (rd *ResultDisplayer) printFailureErrors(detail failureDetail) {
	if len(detail.errors) == 0 || !rd.verbose {
		return
	}

	ui.Println(ui.Color(indent(1, "Errors:"), ui.ColorYellow))
	for _, err := range detail.errors {
		ui.Printf("%s%s\n", indentLevel2, err.Message)
		if err.Event != nil {
			ui.Printf("%sEvent: %s\n", indentLevel2, rd.formatJSON(err.Event))
		}
	}
}

// formatJSON converts any value to a pretty-printed JSON string
func (rd *ResultDisplayer) formatJSON(v any) string {
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
func (rd *ResultDisplayer) generateDiff(expected, actual string) string {
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
