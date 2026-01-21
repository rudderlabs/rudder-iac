package renderer

import (
	"fmt"
	"io"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

type TextRenderer struct {
	out io.Writer
	err io.Writer
}

func NewTextRenderer(out, err io.Writer) Renderer {
	return &TextRenderer{
		out: out,
		err: err,
	}
}

func (r *TextRenderer) Render(diagnostics []validation.Diagnostic) error {
	var errorCount, warningCount int

	for _, d := range diagnostics {
		switch d.Severity {
		case rules.Error:
			r.renderDiagnostic(r.err, "error", d)
			errorCount++
		case rules.Warning:
			r.renderDiagnostic(r.out, "warning", d)
			warningCount++
		}
	}

	if errorCount > 0 || warningCount > 0 {
		fmt.Fprintf(r.out, "Found %d error(s), %d warning(s)\n", errorCount, warningCount)
	}

	return nil
}

func (r *TextRenderer) renderDiagnostic(w io.Writer, level string, d validation.Diagnostic) {
	fmt.Fprintf(w, "\n")

	// Header: error[rule-id]: message
	fmt.Fprintf(w, "%s[%s]: %s\n", level, d.RuleID, d.Message)

	// Location: --> file:line:column
	fmt.Fprintf(w, "  --> %s:%d:%d\n", d.File, d.Position.Line, d.Position.Column)

	// Line text with squiggly underline (only if LineText is available)
	if d.Position.LineText != "" {
		lineNumWidth := len(fmt.Sprintf("%d", d.Position.Line))
		padding := strings.Repeat(" ", lineNumWidth)

		// Empty gutter line
		fmt.Fprintf(w, "   %s |\n", padding)

		// Line with content
		fmt.Fprintf(w, "   %d | %s\n", d.Position.Line, d.Position.LineText)

		// Squiggly line underneath
		squiggly := strings.Repeat("^", len(d.Position.LineText))
		fmt.Fprintf(w, "   %s | %s\n", padding, squiggly)
	}

	fmt.Fprintln(w) // Blank line between diagnostics
}
