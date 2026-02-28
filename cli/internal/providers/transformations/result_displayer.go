package transformations

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pmezard/go-difflib/difflib"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

const (
	indentLevel1  = "  "
	indentLevel2  = "    "
	indentLevel3  = "      "
	separatorLine = "------------------------------------------------------------"
	detailBoxWidth = 60
	headerBoxWidth = 78
	lineWidth = 80

	// Symbols
	symbolPass     = "✓"
	symbolMismatch = "⊗"
	symbolError    = "✕"
	symbolSuite    = "♦"
	symbolTreeNode = "└"
	symbolBullet   = "•"
)

type (
	TransformationTestWithDefinitions = model.TransformationTestWithDefinitions
	TestResults                       = model.TestResults
)

// indent returns a string with the specified indentation level applied
func indent(s string, level int) string {
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

func printIndentedLine(msg string, level int, newLine bool) {
	formatted := indent(msg, level)
	if newLine {
		ui.Println(formatted)
		return
	}

	ui.Printf("%s", formatted)
}

func center(s string, width int) string {
	if len(s) >= width {
		return s
	}
	padding := width - len(s)
	leftPad := padding / 2
	rightPad := padding - leftPad
	return strings.Repeat(" ", leftPad) + s + strings.Repeat(" ", rightPad)
}

func printSection(title string, underlineLen int) {
	ui.Println(ui.Bold(title))
	ui.Println(strings.Repeat("─", underlineLen))
	ui.Println()
}

func printSectionSummary(total int, metrics ...string) {
	parts := make([]string, 0, len(metrics)+1)
	parts = append(parts, fmt.Sprintf("Total  %d", total))
	parts = append(parts, metrics...)

	ui.Println()
	ui.Println(strings.Join(parts, "  "+symbolBullet+"  "))
	ui.Println(separatorLine)
	ui.Println()
}

func printBox(lines []string, width int) {
	ui.Println("┌" + strings.Repeat("─", width))
	for _, line := range lines {
		ui.Printf("│ %s\n", line)
	}
	ui.Println("└" + strings.Repeat("─", width))
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

	rd.displayLibraries(results.Libraries)
	rd.displayTransformations(results.Transformations)

	rd.printSummary()
}

func (rd *ResultDisplayer) printHeader() {
	ui.Println()
	ui.Println("┌" + strings.Repeat("─", headerBoxWidth) + "┐")
	ui.Println("│" + ui.Bold(center("TRANSFORMATION TEST SUITE", headerBoxWidth)) + "│")
	ui.Println("└" + strings.Repeat("─", headerBoxWidth) + "┘")
	ui.Println()
}

func (rd *ResultDisplayer) displayLibraries(libraries []transformations.LibraryTestResult) {
	if len(libraries) == 0 {
		return
	}

	printSection("LIBRARIES", 10)

	for _, lib := range libraries {
		if lib.Pass {
			ui.Printf("%s  %s\n", ui.Color(symbolPass, ui.ColorGreen), lib.HandleName)
			printIndentedLine(ui.Color(symbolPass, ui.ColorGreen)+" syntax ok", 1, true)

			rd.libraryCounter.passed++
		} else {
			ui.Printf("%s  %s\n", ui.Color(symbolError, ui.ColorRed), lib.HandleName)
			printIndentedLine(ui.Color(symbolError, ui.ColorRed)+" syntax error", 1, true)
			if lib.Message != "" {
				printIndentedLine(lib.Message, 3, true)
			}

			rd.libraryCounter.errored++
		}
		ui.Println()
	}

	total := rd.libraryCounter.total()
	printSectionSummary(
		total,
		fmt.Sprintf("Passed  %d", rd.libraryCounter.passed),
		fmt.Sprintf("Errored  %d", rd.libraryCounter.errored),
	)
}

func (rd *ResultDisplayer) displayTransformations(transformations []*TransformationTestWithDefinitions) {
	if len(transformations) == 0 {
		return
	}

	printSection("TEST SUITES", 13)

	for _, trWithDef := range transformations {
		expectedOutputMap := buildExpectedOutputMap(trWithDef.Definitions)
		rd.displayTransformation(trWithDef.Result, expectedOutputMap)
	}

	total := rd.suiteCounter.total()

	printSectionSummary(
		total,
		fmt.Sprintf("Passed  %d", rd.suiteCounter.passed),
		fmt.Sprintf("Mismatch errors %d", rd.suiteCounter.mismatched),
		fmt.Sprintf("Execution errors  %d", rd.suiteCounter.errored),
	)

	rd.printSuiteFailures()
}

func (rd *ResultDisplayer) printSummary() {
	ui.Println(strings.Repeat("=", lineWidth))
	ui.Println(ui.Bold("SUMMARY"))
	ui.Println(strings.Repeat("=", lineWidth))

	if rd.libraryCounter.passed > 0 || rd.libraryCounter.errored > 0 {
		ui.Printf("Libraries       %d passed       %d errored\n",
			rd.libraryCounter.passed,
			rd.libraryCounter.errored,
		)
	}

	if rd.suiteCounter.passed > 0 || rd.suiteCounter.mismatched > 0 || rd.suiteCounter.errored > 0 {
		ui.Printf("Suites          %d passed       %d errored       %d mismatched\n",
			rd.suiteCounter.passed,
			rd.suiteCounter.errored,
			rd.suiteCounter.mismatched,
		)
	}

	ui.Println(strings.Repeat("─", lineWidth))
}

func (rd *ResultDisplayer) displayTransformation(
	result *transformations.TransformationTestResult,
	expectedOutputMap map[string][]any,
) {
	ui.Printf("%s %s\n", symbolSuite, result.Name)

	if len(result.Imports) > 0 {
		importedLibs := strings.Join(result.Imports, ", ")
		printIndentedLine(ui.GreyedOut(fmt.Sprintf("imported libraries: %s", importedLibs)), 1, true)
	}

	for _, testResult := range result.TestSuiteResult.Results {
		expectedOutput := expectedOutputMap[testResult.Name]
		rd.displayTestResult(result.Name, testResult, expectedOutput)
	}
}

func (rd *ResultDisplayer) displayTestResult(
	transformationName string,
	testResult transformations.TestResult,
	expectedOutput []any,
) {
	switch testResult.Status {
	case transformations.TestRunStatusPass:
		ui.Printf("  %s %s  %s\n", symbolTreeNode, ui.Color(symbolPass, ui.ColorGreen), testResult.Name)
		rd.suiteCounter.passed++

	case transformations.TestRunStatusFail:
		ui.Printf("  %s %s  %s\n", symbolTreeNode, ui.Color(symbolMismatch, ui.ColorYellow), testResult.Name)
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
		ui.Printf("  %s %s  %s\n", symbolTreeNode, ui.Color(symbolError, ui.ColorRed), testResult.Name)
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

	printSection("FAILURE DETAILS", 15)

	for _, failure := range rd.suiteCounter.failures {
		rd.printFailureDetail(failure)
	}

	if !rd.verbose {
		ui.Printf("%s", ui.GreyedOut("\nTip: run with --verbose to see full event diffs\n"))
	}
	ui.Println()
}

func (rd *ResultDisplayer) printFailureDetail(detail failureDetail) {
	symbol := symbolMismatch
	if detail.status == transformations.TestRunStatusError {
		symbol = symbolError
	}

	ui.Printf("%s  %s  ›  %s\n",
		ui.Color(symbol, ui.ColorRed),
		detail.transformationName,
		detail.testName,
	)

	if detail.status == transformations.TestRunStatusError {
		rd.printErrorDetail(detail)
	} else {
		rd.printMismatchedOutput(detail)
	}

	ui.Println()
}

func (rd *ResultDisplayer) printErrorDetail(detail failureDetail) {
	var lines []string
	for _, err := range detail.errors {
		messageLines := strings.SplitSeq(err.Message, "\n")
		for line := range messageLines {
			lines = append(lines, line)
		}
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("errored input event can be found at index %d in input file", err.EventIndex))

		if rd.verbose && err.Event != nil {
			eventJSON := formatJSON(err.Event)
			eventLines := strings.SplitSeq(eventJSON, "\n")
			for line := range eventLines {
				lines = append(lines, line)
			}
		}
	}

	printBox(lines, detailBoxWidth)
}

func (rd *ResultDisplayer) printMismatchedOutput(detail failureDetail) {
	var lines []string
	if !rd.verbose || len(detail.expectedOutput) == 0 {
		lines = append(lines, "Actual output mismatched from expected output")
	} else {
		diff := generateDiff(
			formatJSON(detail.expectedOutput),
			formatJSON(detail.actualOutput),
		)
		if diff != "" {
			diffLines := strings.SplitSeq(diff, "\n")
			for line := range diffLines {
				lines = append(lines, line)
			}
		}
	}

	printBox(lines, detailBoxWidth)
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

func generateDiff(expected, actual string) string {
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

func buildExpectedOutputMap(definitions []*transformations.TestDefinition) map[string][]any {
	expectedOutputMap := make(map[string][]any)
	for _, def := range definitions {
		expectedOutputMap[def.Name] = def.ExpectedOutput
	}
	return expectedOutputMap
}
