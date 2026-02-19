package testorchestrator

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pmezard/go-difflib/difflib"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

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

// Display formats and displays test results
func (f *Formatter) Display(results *TestResults) {
	ui.Println(ui.Bold("\nTesting transformations...\n"))

	passed := 0
	failed := 0
	errors := 0
	var failures []failureDetail

	// Display library results first
	if len(results.Libraries) > 0 {
		ui.Printf("  %s\n", ui.Bold("Libraries"))
		for _, lib := range results.Libraries {
			if lib.Pass {
				ui.Printf("    %s %s\n", ui.Color("✓", ui.ColorGreen), lib.HandleName)
			} else {
				msg := lib.HandleName
				if lib.Message != "" {
					msg = fmt.Sprintf("%s: %s", lib.HandleName, lib.Message)
				}
				ui.Printf("    %s %s\n", ui.Color("✗", ui.ColorRed), msg)
				failed++
			}
		}
		ui.Println()
	}

	// Iterate through all transformation results
	for _, trWithDef := range results.Transformations {
		tr := trWithDef.Result
		definitions := trWithDef.Definitions

		// Build map of test name to expected output
		expectedOutputMap := make(map[string][]any)
		for _, def := range definitions {
			expectedOutputMap[def.Name] = def.ExpectedOutput
		}

		// Print transformation name
		testCount := len(tr.TestSuiteResult.Results)
		ui.Printf("  %s (%d test%s)\n", ui.Bold(tr.Name), testCount, pluralize(testCount))

		// Print each test result
		for _, testResult := range tr.TestSuiteResult.Results {
			// Find matching expected output by test name
			expectedOutput := expectedOutputMap[testResult.Name]

			switch testResult.Status {
			case transformations.TestRunStatusPass:
				ui.Printf("    %s %s\n", ui.Color("✓", ui.ColorGreen), testResult.Name)
				passed++
			case transformations.TestRunStatusFail:
				ui.Printf("    %s %s\n", ui.Color("✗", ui.ColorRed), testResult.Name)
				failed++
				// Collect failure details for verbose output
				failures = append(failures, failureDetail{
					transformationName: tr.Name,
					testName:           testResult.Name,
					status:             testResult.Status,
					expectedOutput:     expectedOutput,
					actualOutput:       testResult.ActualOutput,
					errors:             testResult.Errors,
				})
			case transformations.TestRunStatusError:
				ui.Printf("    %s %s\n", ui.Color("⚠", ui.ColorYellow), testResult.Name)
				errors++
				// Collect error details for verbose output
				failures = append(failures, failureDetail{
					transformationName: tr.Name,
					testName:           testResult.Name,
					status:             testResult.Status,
					errors:             testResult.Errors,
				})
			}
		}
		ui.Println()
	}

	// Print separator
	ui.Println(strings.Repeat("-", 60))

	// Print summary
	f.printSummary(passed, failed, errors)

	// Print detailed failures if verbose or hint if not
	if len(failures) > 0 {
		if f.verbose {
			ui.Println(ui.Bold("\nFailure Details:\n"))
			for _, failure := range failures {
				f.printFailureDetail(failure)
			}
		} else {
			ui.Printf("\n%s\n", ui.GreyedOut("Use --verbose to see detailed output and diffs"))
		}
	}
}

// printSummary prints the test summary with colored counts
func (f *Formatter) printSummary(passed, failed, errors int) {
	total := passed + failed + errors

	var parts []string

	if passed > 0 {
		parts = append(parts, fmt.Sprintf("%s passed", ui.Color(fmt.Sprintf("%d", passed), ui.ColorGreen)))
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%s failed", ui.Color(fmt.Sprintf("%d", failed), ui.ColorRed)))
	}
	if errors > 0 {
		parts = append(parts, fmt.Sprintf("%s error%s", ui.Color(fmt.Sprintf("%d", errors), ui.ColorYellow), pluralize(errors)))
	}

	summary := strings.Join(parts, ", ")
	ui.Printf("\nTests: %s, %d total\n", summary, total)
}

// printFailureDetail prints detailed information about a test failure
func (f *Formatter) printFailureDetail(detail failureDetail) {
	// Print header: transformation > test name
	ui.Printf("%s\n", ui.Bold(fmt.Sprintf("%s > %s", detail.transformationName, detail.testName)))

	// If error status, print error messages
	if detail.status == transformations.TestRunStatusError {
		ui.Println(ui.Color("  Error:", ui.ColorRed))
		for _, err := range detail.errors {
			ui.Printf("    %s\n", err.Message)
			if err.Event != nil {
				ui.Printf("    Event: %s\n", f.formatJSON(err.Event))
			}
		}
		ui.Println()
		return
	}

	// For failures, show expected vs actual with diff
	if detail.expectedOutput != nil && len(detail.expectedOutput) > 0 {
		ui.Println(ui.Color("  Expected:", ui.ColorGreen))
		expectedJSON := f.formatJSON(detail.expectedOutput)
		ui.Printf("    %s\n", indentMultiline(expectedJSON, "    "))

		ui.Println(ui.Color("  Actual:", ui.ColorRed))
		actualJSON := f.formatJSON(detail.actualOutput)
		ui.Printf("    %s\n", indentMultiline(actualJSON, "    "))

		// Print diff
		diff := f.generateDiff(expectedJSON, actualJSON)
		if diff != "" {
			ui.Println(ui.Color("  Diff:", ui.ColorYellow))
			ui.Print(indentMultiline(diff, "    "))
		}
	} else {
		// No expected output, just show actual
		ui.Println(ui.Color("  Actual Output:", ui.ColorRed))
		ui.Printf("    %s\n", indentMultiline(f.formatJSON(detail.actualOutput), "    "))
	}

	// Print errors if any
	if len(detail.errors) > 0 {
		ui.Println(ui.Color("  Errors:", ui.ColorYellow))
		for _, err := range detail.errors {
			ui.Printf("    %s\n", err.Message)
			if err.Event != nil {
				ui.Printf("    Event: %s\n", f.formatJSON(err.Event))
			}
		}
	}

	ui.Println()
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
	// Split into lines for difflib
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
		if line != "" || i < len(lines)-1 { // Keep empty lines except trailing
			lines[i] = indent + line
		}
	}
	return strings.TrimRight(strings.Join(lines, "\n"), " \t")
}
