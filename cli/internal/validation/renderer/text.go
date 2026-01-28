package renderer

import (
	"fmt"
	"io"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

type TextRenderer struct {
	w io.Writer
}

func NewTextRenderer(w io.Writer) Renderer {
	return &TextRenderer{
		w: w,
	}
}

func (r *TextRenderer) Render(diagnostics validation.Diagnostics) error {
	var errorCount, warningCount int

	for _, d := range diagnostics {
		switch d.Severity {
		case rules.Error:
			errorCount += 1

		case rules.Warning:
			warningCount += 1
		}
		r.renderDiagnostic(r.w, d)
	}

	if errorCount > 0 || warningCount > 0 {
		fmt.Fprintln(r.w) // blank line between the last diagnostic and summary
		fmt.Fprintf(r.w, "Found %d error(s), %d warning(s)\n", errorCount, warningCount)
	}

	return nil
}

// renderDiagnostic simply renders the information in the diagnostic to the writer
// in a human readable format. eFor xample:
//
// error[project/version-valid]: version must be one of the supported versions: rudder/v1, rudder/v0.1, rudder/0.1
//   --> empty/malformed.yaml:1:1
//   |
// 1 | version: rudder/v1.1
//   | ^^^^^^^^^^^^^^^^^^^^

// Found 1 error(s), 0 warning(s)
func (r *TextRenderer) renderDiagnostic(w io.Writer, d validation.Diagnostic) {
	fmt.Fprintf(w, "\n")

	// error[rule-id]: message
	fmt.Fprintf(w, "%s[%s]: %s\n", d.Severity.String(), d.RuleID, d.Message)

	// file:line:column
	fmt.Fprintf(w, "  --> %s:%d:%d\n", d.File, d.Position.Line, d.Position.Column)

	if d.Position.LineText != "" {
		lineNumWidth := len(fmt.Sprintf("%d", d.Position.Line))
		padding := strings.Repeat(" ", lineNumWidth)

		fmt.Fprintf(w, "   %s |\n", padding) // empty gutter line
		fmt.Fprintf(w, "   %d | %s\n", d.Position.Line, d.Position.LineText)
		squiggly := strings.Repeat("^", len(d.Position.LineText)) // squiggly line underneath
		fmt.Fprintf(w, "   %s | %s\n", padding, squiggly)         // empty gutter line
	}
}
