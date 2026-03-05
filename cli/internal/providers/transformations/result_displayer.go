package transformations

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/samber/lo"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

const (
	// Display formatting
	indentLevel1 = "  "
	indentLevel2 = "    "
	lineWidth    = 80
	statusColumn = 70

	// Section titles
	sectionLibraryTests        = "LIBRARY TESTS"
	sectionTransformationTests = "TRANSFORMATION TESTS"
	sectionFailures            = "FAILURES"
	sectionSummary             = "SUMMARY"

	// Failure types
	failureTypeExecutionError = "EXECUTION ERROR"
	failureTypeOutputMismatch = "OUTPUT MISMATCH"

	// Status labels
	syntaxStatusOK     = "syntax ok"
	syntaxStatusError  = "syntax error"
	testStatusPassed   = "passed"
	testStatusMismatch = "output mismatch"
	testStatusError    = "execution error"

	// Message labels
	labelFullStackTrace  = "Full stack trace:"
	labelImportedLibs    = "imported libraries: %s"
	labelInputEvent      = "Input event at index %d:"
	labelErrorOccurred1  = "This error occurred 1 time (input index: %d)"
	labelErrorOccurredN  = "This error occurred %d times (input indices: %s)"
	labelResultFailed    = "Result: FAILED (exit code 1)"
	labelResultPassed    = "Result: PASSED"
	labelLegend          = "Legend: %s passed  %s execution error  %s output mismatch"
	labelVerboseTip      = "Tip: Run with --verbose for full diffs and input payloads"
	labelSuiteField      = "Suite:       %s"
	labelTestField       = "Test:        %s"
	labelErrorField      = "Error: %s"
	labelLibrarySummary  = "Libraries: %d passed, %d errored"
	labelTestCaseSummary = "Test cases: %d passed, %d errored, %d mismatched"

	// Symbols
	symbolPass     = "✓"
	symbolMismatch = "⊗"
	symbolError    = "✕"
)

type (
	TransformationTestWithDefinitions = model.TransformationTestWithDefinitions
	TestResults                       = model.TestResults
)

func indent(s string, level int) string {
	switch level {
	case 1:
		return indentLevel1 + s
	case 2:
		return indentLevel2 + s
	default:
		return s
	}
}

func newLine() {
	ui.Println()
}

/*
	printIndentedLine("error message", 1, true)
	// Output:   error message
	printIndentedLine("prefix: ", 0, false)
	// Output: prefix: (no newline)
*/
func printIndentedLine(msg string, level int, newLine bool) {
	indented := indent(msg, level)
	if newLine {
		ui.Println(indented)
		return
	}

	ui.Printf("%s", indented)
}

/*
	printWithPadding("✓ test-case-1", "passed", 30)
	// Output: ✓ test-case-1                 passed
	printWithPadding("✓ test-case-1", "passed", 10)
	// Output: ✓ test-case-1 passed
*/
func printWithPadding(leftText, rightText string, rightTextStart int) {
	padding := max(rightTextStart - len(leftText), 1)
	ui.Printf("%s%s%s\n", leftText, strings.Repeat(" ", padding), rightText)
}

/*
printSection("TEST RESULTS")
	// Output:
	// TEST RESULTS (bold)
	// --------------------------------------------------------------------------------
	// (blank line)
*/
func printSection(title string) {
	ui.Println(ui.Bold(title))
	printSeparator("-")
	newLine()
}

func printSeparator(char string) {
	ui.Printf("%s\n", strings.Repeat(char, lineWidth))
}

/*
	printFullStackTrace([]string{"  at line 42", "  in function foo()"}, 2)
	// Output:
	// (blank line)
	//     Full stack trace:
	//       at line 42
	//       in function foo()
*/
func printFullStackTrace(lines []string, indent int) {
	newLine()
	printIndentedLine(labelFullStackTrace, indent, true)

	for _, line := range lines {
		printIndentedLine(line, indent, true)
	}
}

type failureDetail struct {
	transformationName string
	testName           string
	status             transformations.TestRunStatus
	expectedOutput     []any
	actualOutput       []any
	errors             []transformations.TestError
}

type suiteCounter struct {
	passed     int
	mismatched int
	errored    int
	failures   []failureDetail
}

type libraryCounter struct {
	passed  int
	errored int
}

func (c *suiteCounter) total() int {
	return c.passed + c.mismatched + c.errored
}

func (lc *libraryCounter) total() int {
	return lc.passed + lc.errored
}

type ResultDisplayer struct {
	verbose        bool
	suiteCounter   suiteCounter
	libraryCounter libraryCounter
}

func NewResultDisplayer(verbose bool) *ResultDisplayer {
	return &ResultDisplayer{
		verbose: verbose,
	}
}

// Display formats and displays test results
func (rd *ResultDisplayer) Display(results *TestResults) {
	rd.printHeader()
	rd.displayDefaultTestSuiteWarning(results)

	rd.displayLibraries(results.Libraries)
	rd.displayTransformations(results.Transformations)

	rd.printSummary()
}

func (rd *ResultDisplayer) displayDefaultTestSuiteWarning(results *TestResults) {
	names := results.DefaultSuiteTransformationNames()
	if len(names) == 0 {
		return
	}

	ui.Println(ui.Color("WARNING: Default test suite used for test", ui.ColorYellow))
	ui.Printf("No test suites defined for transformations:\n - %s\n", strings.Join(names, "\n - "))
	ui.Println(ui.GreyedOut("Use `rudder-cli transformations show-default-events` to view default events."))
	newLine()
}

/*
(blank line)
Transformation Test Suite v1.0.0 (bold)
================================================================================
(blank line)
*/
func (rd *ResultDisplayer) printHeader() {
	newLine()
	ui.Println(ui.Bold("Transformation Test Suite v1.0.0"))
	printSeparator("=")
	newLine()
}

func (rd *ResultDisplayer) displayLibraries(libraries []transformations.LibraryTestResult) {
	if len(libraries) == 0 {
		return
	}

	uniqueLibs := lo.UniqBy(libraries, func(lib transformations.LibraryTestResult) string {
		return lib.HandleName
	})

	passed := lo.Filter(uniqueLibs, func(lib transformations.LibraryTestResult, _ int) bool {
		return lib.Pass
	})
	failed := lo.Filter(uniqueLibs, func(lib transformations.LibraryTestResult, _ int) bool {
		return !lib.Pass
	})

	printSection(sectionLibraryTests)

	for _, lib := range passed {
		printWithPadding(
			fmt.Sprintf("%s  %s", ui.Color(symbolPass, ui.ColorGreen), lib.HandleName),
			syntaxStatusOK,
			statusColumn,
		)
		rd.libraryCounter.passed++
	}

	for _, lib := range failed {
		printWithPadding(
			fmt.Sprintf("%s  %s", ui.Color(symbolError, ui.ColorRed), lib.HandleName),
			syntaxStatusError,
			statusColumn,
		)

		// Print the first line of the error message
		lines := strings.Split(lib.Message, "\n")
		printIndentedLine(lines[0], 2, true)

		if rd.verbose && len(lines) > 1 {
			printFullStackTrace(lines[1:], 2)
		}

		newLine()
		rd.libraryCounter.errored++
	}

	newLine()
	ui.Printf(labelLibrarySummary+"\n", rd.libraryCounter.passed, rd.libraryCounter.errored)
	printSeparator("-")
	newLine()
}

func (rd *ResultDisplayer) displayTransformations(transformations []*TransformationTestWithDefinitions) {
	if len(transformations) == 0 {
		return
	}

	printSection(sectionTransformationTests)

	for _, trWithDef := range transformations {
		expectedOutputMap := buildExpectedOutputMap(trWithDef.Definitions)
		rd.displayTransformation(trWithDef.Result, expectedOutputMap)
	}

	ui.Printf(labelTestCaseSummary+"\n",
		rd.suiteCounter.passed,
		rd.suiteCounter.errored,
		rd.suiteCounter.mismatched,
	)
	printSeparator("-")
	newLine()

	rd.printSuiteFailures()
}

func (rd *ResultDisplayer) printSummary() {
	ui.Println(ui.Bold(sectionSummary))
	printSeparator("=")

	if rd.libraryCounter.total() > 0 {
		ui.Printf("Libraries       %d passed       %d errored\n",
			rd.libraryCounter.passed,
			rd.libraryCounter.errored,
		)
	}

	if rd.suiteCounter.total() > 0 {
		ui.Printf("Test cases      %d passed       %d errored       %d mismatched\n",
			rd.suiteCounter.passed,
			rd.suiteCounter.errored,
			rd.suiteCounter.mismatched,
		)
	}

	printSeparator("-")

	hasFailures := rd.libraryCounter.errored > 0 || rd.suiteCounter.errored > 0 || rd.suiteCounter.mismatched > 0
	if hasFailures {
		ui.Println(ui.Color(labelResultFailed, ui.ColorRed))
	} else {
		ui.Println(ui.Color(labelResultPassed, ui.ColorGreen))
	}

	newLine()
	ui.Printf(labelLegend+"\n",
		ui.Color(symbolPass, ui.ColorGreen),
		ui.Color(symbolError, ui.ColorRed),
		ui.Color(symbolMismatch, ui.ColorYellow),
	)

	newLine()
	if !rd.verbose {
		ui.Println(labelVerboseTip)
	}
	printSeparator("=")
}

func (rd *ResultDisplayer) displayTransformation(
	result *transformations.TransformationTestResult,
	expectedOutputMap map[string][]any,
) {
	ui.Println(result.Name)

	if len(result.Imports) > 0 {
		importedLibs := strings.Join(result.Imports, ", ")
		printIndentedLine(ui.GreyedOut(fmt.Sprintf(labelImportedLibs, importedLibs)), 1, true)
	}

	for _, testResult := range result.TestSuiteResult.Results {
		expectedOutput := expectedOutputMap[testResult.Name]
		rd.displayTestResult(result.Name, testResult, expectedOutput)
	}

	newLine()
}

func (rd *ResultDisplayer) displayTestResult(
	transformationName string,
	testResult transformations.TestResult,
	expectedOutput []any,
) {
	switch testResult.Status {
	case transformations.TestRunStatusPass:
		printWithPadding(
			fmt.Sprintf("  %s %s", ui.Color(symbolPass, ui.ColorGreen), testResult.Name),
			testStatusPassed,
			statusColumn,
		)
		rd.suiteCounter.passed++

	case transformations.TestRunStatusFail:
		printWithPadding(
			fmt.Sprintf("  %s %s", ui.Color(symbolMismatch, ui.ColorYellow), testResult.Name),
			testStatusMismatch,
			statusColumn,
		)
		rd.suiteCounter.mismatched++

		rd.suiteCounter.failures = append(rd.suiteCounter.failures, failureDetail{
			transformationName: transformationName,
			testName:           testResult.Name,
			status:             testResult.Status,
			expectedOutput:     expectedOutput,
			actualOutput:       testResult.ActualOutput,
			errors:             testResult.Errors,
		})

	case transformations.TestRunStatusError:
		printWithPadding(
			fmt.Sprintf("  %s %s", ui.Color(symbolError, ui.ColorRed), testResult.Name),
			testStatusError,
			statusColumn,
		)
		rd.suiteCounter.errored++

		rd.suiteCounter.failures = append(rd.suiteCounter.failures, failureDetail{
			transformationName: transformationName,
			testName:           testResult.Name,
			status:             testResult.Status,
			errors:             testResult.Errors,
		})
	}
}

func (rd *ResultDisplayer) printSuiteFailures() {
	if len(rd.suiteCounter.failures) == 0 {
		return
	}

	printSection(sectionFailures)

	for i, failure := range rd.suiteCounter.failures {
		rd.printFailureDetail(failure, i+1)
	}
}

func (rd *ResultDisplayer) printFailureDetail(detail failureDetail, failureNum int) {
	var failureType string
	if detail.status == transformations.TestRunStatusError {
		failureType = failureTypeExecutionError
	} else {
		failureType = failureTypeOutputMismatch
	}

	ui.Printf("[%d/%d]  %s\n", failureNum, len(rd.suiteCounter.failures), failureType)
	printSeparator("-")

	ui.Printf(labelSuiteField+"\n", detail.transformationName)
	ui.Printf(labelTestField+"\n", detail.testName)

	if detail.status == transformations.TestRunStatusError {
		rd.printErrorDetail(detail)
	} else {
		rd.printMismatchedOutput(detail)
	}

	printSeparator("=")
}

type errorGroup struct {
	message    string
	indices    []int
	event      any
	eventIndex int
}

func (rd *ResultDisplayer) printErrorDetail(detail failureDetail) {
	errorGroups, groupOrder := groupErrorsByMessage(detail.errors)

	for groupIdx, msgKey := range groupOrder {
		group := errorGroups[msgKey]
		lines := strings.Split(group.message, "\n")

		newLine()
		ui.Printf(labelErrorField+"\n", lines[0])
		if len(lines) > 1 {
			ui.Println(lines[1])
		}

		if rd.verbose && len(lines) > 2 {
			printFullStackTrace(lines[2:], 0)
		}

		newLine()
		if len(group.indices) == 1 {
			ui.Printf(labelErrorOccurred1+"\n", group.indices[0])
		} else {
			indicesStr := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(group.indices)), ", "), "[]")
			ui.Printf(labelErrorOccurredN+"\n", len(group.indices), indicesStr)
		}

		if rd.verbose && group.event != nil {
			newLine()
			ui.Printf(labelInputEvent+"\n", group.eventIndex)
			ui.Println(formatJSON(group.event))
		}

		if groupIdx < len(groupOrder)-1 {
			newLine()
			printSeparator("=")
		}
	}
}

func groupErrorsByMessage(errors []transformations.TestError) (map[string]*errorGroup, []string) {
	errorGroups := make(map[string]*errorGroup)
	var groupOrder []string

	for _, err := range errors {
		if _, exists := errorGroups[err.Message]; !exists {
			errorGroups[err.Message] = &errorGroup{
				message:    err.Message,
				indices:    []int{},
				event:      err.Event,
				eventIndex: err.EventIndex,
			}
			groupOrder = append(groupOrder, err.Message)
		}
		errorGroups[err.Message].indices = append(errorGroups[err.Message].indices, err.EventIndex)
	}

	return errorGroups, groupOrder
}

func (rd *ResultDisplayer) printMismatchedOutput(detail failureDetail) {
	if len(detail.expectedOutput) == 0 {
		newLine()
		ui.Println("Actual output mismatched from expected output")
		return
	}

	contextLines := 0
	if rd.verbose {
		contextLines = 3
	}

	diff := generateDiff(
		formatJSON(detail.expectedOutput),
		formatJSON(detail.actualOutput),
		contextLines,
	)

	newLine()
	ui.Println(diff)
}

func formatJSON(v any) string {
	if v == nil {
		return "null"
	}

	jsonBytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("<error formatting JSON: %v>", err)
	}
	return string(jsonBytes)
}

/*
generateDiff creates a unified diff between expected and actual strings.
contextLines controls how many unchanged lines to show around changes (0 = none, 3 = typical).

Example:

	expected := `{
	  "status": "success"
	}`
	actual := `{
	  "status": "failed",
	  "error": "timeout"
	}`
	diff := generateDiff(expected, actual, 0)
	// Returns:
	// --- Expected
	// +++ Actual
	// @@ -2 +2,2 @@
	// -  "status": "success"
	// +  "status": "failed",
	// +  "error": "timeout"
*/
func generateDiff(expected, actual string, contextLines int) string {
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")

	// Add trailing newlines to each line for difflib
	for i := range expectedLines {
		expectedLines[i] += "\n"
	}
	for i := range actualLines {
		actualLines[i] += "\n"
	}

	diff := difflib.UnifiedDiff{
		A:        expectedLines,
		B:        actualLines,
		FromFile: "Expected",
		ToFile:   "Actual",
		Context:  contextLines,
	}

	result, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		return ""
	}

	return result
}

func buildExpectedOutputMap(definitions []*transformations.TestDefinition) map[string][]any {
	expectedOutputMap := make(map[string][]any)
	for _, def := range definitions {
		expectedOutputMap[def.Name] = def.ExpectedOutput
	}
	return expectedOutputMap
}
