package validator

import (
	"fmt"
	"io"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

const minTerminalWidth = 60

// TerminalDisplayer renders a validation report to a terminal.
type TerminalDisplayer struct {
	w io.Writer
}

// NewTerminalDisplayer creates a new TerminalDisplayer that writes to w.
func NewTerminalDisplayer(w io.Writer) *TerminalDisplayer {
	return &TerminalDisplayer{w: w}
}

// Display renders the validation report to the terminal.
func (d *TerminalDisplayer) Display(report *ValidationReport) {
	width := ui.GetTerminalWidth()
	if width < minTerminalWidth {
		width = minTerminalWidth
	}
	statusCol := width * 3 / 4

	fmt.Fprintln(d.w)
	fmt.Fprintln(d.w, ui.SectionHeader("Data Graph Validation Report", "=", width))

	models := report.ResourcesByType("model")
	relationships := report.ResourcesByType("relationship")

	if len(models) > 0 {
		d.displaySection("MODELS", models, statusCol, width)
	}

	if len(relationships) > 0 {
		d.displaySection("RELATIONSHIPS", relationships, statusCol, width)
	}

	d.displaySummary(models, relationships, width)
}

func (d *TerminalDisplayer) displaySection(title string, rvs []*ResourceValidation, statusCol, width int) {
	fmt.Fprintln(d.w)
	fmt.Fprintln(d.w, ui.SectionHeader(title, "-", width))

	for _, rv := range rvs {
		d.displayResource(rv, statusCol)
	}
}

func (d *TerminalDisplayer) displayResource(rv *ResourceValidation, statusCol int) {
	name := rv.DisplayName
	if name == "" {
		name = rv.ID
	}
	name = fmt.Sprintf("%s (%s)", name, rv.URN)

	if rv.Err != nil {
		fmt.Fprintln(d.w, ui.PadColumns(
			fmt.Sprintf("  %s  %s", ui.Color(ui.SymbolError, ui.ColorRed), name),
			ui.Color("error", ui.ColorRed),
			statusCol,
		))
		fmt.Fprintf(d.w, "       %s\n", rv.Err.Error())
		return
	}

	if rv.HasErrors() {
		var (
			errorCount   = countBySeverity(rv.Issues, "error")
			warningCount = countBySeverity(rv.Issues, "warning")
		)
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
		fmt.Fprintln(d.w, ui.PadColumns(
			fmt.Sprintf("  %s  %s", ui.Color(ui.SymbolError, ui.ColorRed), name),
			status,
			statusCol,
		))
		d.printIssues(rv.Issues)
		return
	}

	if rv.HasWarnings() {
		warningCount := countBySeverity(rv.Issues, "warning")
		status := fmt.Sprintf("%d warning", warningCount)
		if warningCount > 1 {
			status += "s"
		}
		fmt.Fprintln(d.w, ui.PadColumns(
			fmt.Sprintf("  %s  %s", ui.Color(ui.SymbolWarning, ui.ColorYellow), name),
			status,
			statusCol,
		))
		d.printIssues(rv.Issues)
		return
	}

	fmt.Fprintln(d.w, ui.PadColumns(
		fmt.Sprintf("  %s  %s", ui.Color(ui.SymbolPass, ui.ColorGreen), name),
		"pass",
		statusCol,
	))
}

func (d *TerminalDisplayer) printIssues(issues []dgClient.ValidationIssue) {
	for _, issue := range issues {
		color := ui.ColorYellow
		if issue.Severity == "error" {
			color = ui.ColorRed
		}
		fmt.Fprintf(d.w, "       %s: %s\n", ui.Color(issue.Rule, color), issue.Message)
	}
}

func (d *TerminalDisplayer) displaySummary(models, relationships []*ResourceValidation, width int) {
	fmt.Fprintln(d.w)
	fmt.Fprintln(d.w, ui.SectionHeader("SUMMARY", "=", width))

	if len(models) > 0 {
		p, e, w := countStatuses(models)
		fmt.Fprintf(d.w, "Models:         %d passed   %d errors   %d warnings\n", p, e, w)
	}
	if len(relationships) > 0 {
		p, e, w := countStatuses(relationships)
		fmt.Fprintf(d.w, "Relationships:  %d passed   %d errors   %d warnings\n", p, e, w)
	}

	fmt.Fprintln(d.w, ui.Separator("-", width))

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

	fmt.Fprintln(d.w, ui.Separator("=", width))
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

func countStatuses(rvs []*ResourceValidation) (passed, errors, warnings int) {
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
