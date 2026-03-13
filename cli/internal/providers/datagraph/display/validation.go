package display

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/validations"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

const (
	lineWidth    = 80
	statusColumn = 60

	symbolPass    = "✓"
	symbolWarning = "⚠"
	symbolError   = "✕"
)

// ValidationDisplayer formats and displays validation results
type ValidationDisplayer struct {
	w          io.Writer
	jsonOutput bool
}

// NewValidationDisplayer creates a new displayer
func NewValidationDisplayer(w io.Writer, jsonOutput bool) *ValidationDisplayer {
	return &ValidationDisplayer{w: w, jsonOutput: jsonOutput}
}

// Display renders validation results to the terminal or as JSON
func (d *ValidationDisplayer) Display(results *validations.ValidationResults) {
	if d.jsonOutput {
		d.displayJSON(results)
		return
	}
	d.displayTerminal(results)
}

func (d *ValidationDisplayer) displayJSON(results *validations.ValidationResults) {
	type jsonIssue struct {
		Rule     string `json:"rule"`
		Severity string `json:"severity"`
		Message  string `json:"message"`
	}
	type jsonResource struct {
		ID           string      `json:"id"`
		DisplayName  string      `json:"displayName"`
		ResourceType string      `json:"resourceType"`
		Status       string      `json:"status"`
		Issues       []jsonIssue `json:"issues,omitempty"`
		Error        string      `json:"error,omitempty"`
	}
	type jsonOutput struct {
		Status    string         `json:"status"`
		Resources []jsonResource `json:"resources"`
	}

	out := jsonOutput{
		Status:    "executed",
		Resources: make([]jsonResource, 0, len(results.Resources)),
	}

	for _, rv := range results.Resources {
		jr := jsonResource{
			ID:           rv.ID,
			DisplayName:  rv.DisplayName,
			ResourceType: rv.ResourceType,
		}

		if rv.Err != nil {
			jr.Status = "error"
			jr.Error = rv.Err.Error()
		} else if rv.HasErrors() {
			jr.Status = "failed"
		} else if rv.HasWarnings() {
			jr.Status = "warning"
		} else {
			jr.Status = "passed"
		}

		for _, issue := range rv.Issues {
			jr.Issues = append(jr.Issues, jsonIssue{
				Rule:     issue.Rule,
				Severity: issue.Severity,
				Message:  issue.Message,
			})
		}

		out.Resources = append(out.Resources, jr)
	}

	data, _ := json.MarshalIndent(out, "", "  ")
	fmt.Fprintln(d.w, string(data))
}

func (d *ValidationDisplayer) displayTerminal(results *validations.ValidationResults) {
	fmt.Fprintln(d.w)
	fmt.Fprintln(d.w, ui.Bold("Data Graph Validation Report"))
	d.printSeparator("=")

	models := results.ResourcesByType("model")
	relationships := results.ResourcesByType("relationship")

	if len(models) > 0 {
		d.displaySection("MODELS", models)
	}

	if len(relationships) > 0 {
		d.displaySection("RELATIONSHIPS", relationships)
	}

	d.displaySummary(models, relationships)
}

func (d *ValidationDisplayer) displaySection(title string, rvs []*validations.ResourceValidation) {
	fmt.Fprintln(d.w)
	fmt.Fprintln(d.w, ui.Bold(title))
	d.printSeparator("-")

	for _, rv := range rvs {
		d.displayResource(rv)
	}
}

func (d *ValidationDisplayer) displayResource(rv *validations.ResourceValidation) {
	name := rv.DisplayName
	if name == "" {
		name = rv.ID
	}

	if rv.Err != nil {
		d.printWithPadding(
			fmt.Sprintf("  %s  %s", ui.Color(symbolError, ui.ColorRed), name),
			ui.Color("error", ui.ColorRed),
			statusColumn,
		)
		fmt.Fprintf(d.w, "       %s\n", rv.Err.Error())
		return
	}

	if rv.HasErrors() {
		errorCount := countBySeverity(rv.Issues, "error")
		warningCount := countBySeverity(rv.Issues, "warning")
		status := fmt.Sprintf("%d error", errorCount)
		if errorCount > 1 {
			status += "s"
		}
		if warningCount > 0 {
			status += fmt.Sprintf("  %d warning", warningCount)
			if warningCount > 1 {
				status += "s"
			}
		}
		d.printWithPadding(
			fmt.Sprintf("  %s  %s", ui.Color(symbolError, ui.ColorRed), name),
			status,
			statusColumn,
		)
		d.printIssues(rv.Issues)
		return
	}

	if rv.HasWarnings() {
		warningCount := countBySeverity(rv.Issues, "warning")
		status := fmt.Sprintf("%d warning", warningCount)
		if warningCount > 1 {
			status += "s"
		}
		d.printWithPadding(
			fmt.Sprintf("  %s  %s", ui.Color(symbolWarning, ui.ColorYellow), name),
			status,
			statusColumn,
		)
		d.printIssues(rv.Issues)
		return
	}

	d.printWithPadding(
		fmt.Sprintf("  %s  %s", ui.Color(symbolPass, ui.ColorGreen), name),
		"pass",
		statusColumn,
	)
}

func (d *ValidationDisplayer) printIssues(issues []dgClient.ValidationIssue) {
	for _, issue := range issues {
		color := ui.ColorYellow
		if issue.Severity == "error" {
			color = ui.ColorRed
		}
		fmt.Fprintf(d.w, "       %s: %s\n", ui.Color(issue.Rule, color), issue.Message)
	}
}

func (d *ValidationDisplayer) displaySummary(models, relationships []*validations.ResourceValidation) {
	fmt.Fprintln(d.w)
	fmt.Fprintln(d.w, ui.Bold("SUMMARY"))
	d.printSeparator("=")

	if len(models) > 0 {
		p, e, w := countStatuses(models)
		fmt.Fprintf(d.w, "Models:         %d passed   %d errors   %d warnings\n", p, e, w)
	}
	if len(relationships) > 0 {
		p, e, w := countStatuses(relationships)
		fmt.Fprintf(d.w, "Relationships:  %d passed   %d errors   %d warnings\n", p, e, w)
	}

	d.printSeparator("-")

	hasFailures := false
	for _, rv := range append(models, relationships...) {
		if rv.HasErrors() {
			hasFailures = true
			break
		}
	}

	if hasFailures {
		fmt.Fprintln(d.w, ui.Color("Result: FAILED", ui.ColorRed))
	} else {
		fmt.Fprintln(d.w, ui.Color("Result: PASSED", ui.ColorGreen))
	}

	d.printSeparator("=")
}

func (d *ValidationDisplayer) printSeparator(char string) {
	fmt.Fprintf(d.w, "%s\n", strings.Repeat(char, lineWidth))
}

func (d *ValidationDisplayer) printWithPadding(leftText, rightText string, rightTextStart int) {
	padding := max(rightTextStart-len(leftText), 1)
	fmt.Fprintf(d.w, "%s%s%s\n", leftText, strings.Repeat(" ", padding), rightText)
}

func countBySeverity(issues []dgClient.ValidationIssue, severity string) int {
	count := 0
	for _, i := range issues {
		if i.Severity == severity {
			count++
		}
	}
	return count
}

func countStatuses(rvs []*validations.ResourceValidation) (passed, errors, warnings int) {
	for _, rv := range rvs {
		if rv.HasErrors() {
			errors++
		} else if rv.HasWarnings() {
			warnings++
		} else {
			passed++
		}
	}
	return
}
