package ui

import (
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/engine"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/location"
)

// CLIDiagnostic wraps an engine.Diagnostic and provides Rust/Cargo-style terminal rendering
type CLIDiagnostic struct {
	Diagnostic engine.Diagnostic
}

// NewCLIDiagnostic creates a new CLIDiagnostic from an engine.Diagnostic
func NewCLIDiagnostic(d engine.Diagnostic) *CLIDiagnostic {
	return &CLIDiagnostic{Diagnostic: d}
}

// Render produces Rust/Cargo-style formatted output
//
// Output format:
//
//	error[datacatalog/events/required-ids]: property 'id' is mandatory
//	  --> path/to/file.yaml:10:5
//	   |
//	10 |     id: some_value
//	   |     ^^^^^^^^^^^^^^
func (c *CLIDiagnostic) Render() string {
	var sb strings.Builder

	// Line 1: Severity header with rule ID and message
	sb.WriteString(c.renderHeader())
	sb.WriteString("\n")

	// Line 2: File location pointer
	sb.WriteString(c.renderLocation())
	sb.WriteString("\n")

	// Line 3: Separator pipe
	sb.WriteString(c.renderSeparator())
	sb.WriteString("\n")

	// Line 4: Code fragment with line number (if available)
	sb.WriteString(c.renderFragment())
	sb.WriteString("\n")

	// Line 5: Underline pointer (if fragment exists)
	sb.WriteString(c.renderUnderline())

	return sb.String()
}

// renderHeader produces: "error[rule-id]: message"
func (c *CLIDiagnostic) renderHeader() string {
	severityStr := string(c.Diagnostic.Severity)

	// Format: severity[rule]: message
	if c.Diagnostic.Rule != "" {
		return fmt.Sprintf("%s[%s]: %s",
			severityStr,
			c.Diagnostic.Rule,
			c.Diagnostic.Message,
		)
	}
	// For parse errors without rule ID
	return fmt.Sprintf("%s: %s",
		severityStr,
		c.Diagnostic.Message,
	)
}

// renderLocation produces: "  --> path/to/file.yaml:10:5"
func (c *CLIDiagnostic) renderLocation() string {
	pos := c.normalizePosition(c.Diagnostic.Position)
	return fmt.Sprintf("  --> %s:%d:%d",
		c.Diagnostic.File,
		pos.Line,
		pos.Column,
	)
}

// renderSeparator produces: "   |"
func (c *CLIDiagnostic) renderSeparator() string {
	return "   |"
}

// renderFragment produces: "10 | <line content>"
func (c *CLIDiagnostic) renderFragment() string {
	pos := c.normalizePosition(c.Diagnostic.Position)

	if c.Diagnostic.Fragment == "" {
		return fmt.Sprintf("%2d | %s",
			pos.Line,
			"<source unavailable>",
		)
	}

	// Fragment already contains the full line with indentation
	return fmt.Sprintf("%2d | %s",
		pos.Line,
		c.Diagnostic.Fragment,
	)
}

// renderUnderline produces: "   | ^^^^^^^^" (in RED, aligned with content)
func (c *CLIDiagnostic) renderUnderline() string {
	if c.Diagnostic.Fragment == "" {
		return "   |"
	}

	// Calculate leading spaces in fragment to align underline
	trimmed := strings.TrimLeft(c.Diagnostic.Fragment, " \t")
	leadingSpaces := len(c.Diagnostic.Fragment) - len(trimmed)

	// Underline length matches trimmed content length
	underlineLen := len(trimmed)
	if underlineLen == 0 {
		underlineLen = 1 // Minimum one caret
	}

	padding := strings.Repeat(" ", leadingSpaces)
	underline := strings.Repeat("^", underlineLen)

	// Only the squiggly line is colored red
	return fmt.Sprintf("   | %s%s", padding, Color(underline, ColorRed))
}

// normalizePosition ensures Line and Column are at least 1
func (c *CLIDiagnostic) normalizePosition(pos location.Position) location.Position {
	normalized := pos
	if normalized.Line <= 0 {
		normalized.Line = 1
	}
	if normalized.Column <= 0 {
		normalized.Column = 1
	}
	return normalized
}

// DiagnosticsRenderer handles rendering multiple diagnostics
type DiagnosticsRenderer struct {
	diagnostics []engine.Diagnostic
}

// NewDiagnosticsRenderer creates a new renderer for multiple diagnostics
func NewDiagnosticsRenderer(diagnostics []engine.Diagnostic) *DiagnosticsRenderer {
	return &DiagnosticsRenderer{diagnostics: diagnostics}
}

// Render produces the complete output for all diagnostics
func (r *DiagnosticsRenderer) Render() string {
	if len(r.diagnostics) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, d := range r.diagnostics {
		cliDiag := NewCLIDiagnostic(d)
		sb.WriteString(cliDiag.Render())

		// Add blank line between diagnostics (but not after the last one)
		if i < len(r.diagnostics)-1 {
			sb.WriteString("\n\n")
		}
	}
	return sb.String()
}

// HasErrors returns true if any diagnostic has error severity
func (r *DiagnosticsRenderer) HasErrors() bool {
	for _, d := range r.diagnostics {
		if d.Severity == validation.SeverityError {
			return true
		}
	}
	return false
}

// Summary returns a summary line like "Found 3 error(s) and 2 warning(s)"
func (r *DiagnosticsRenderer) Summary() string {
	var errors, warnings, infos int
	for _, d := range r.diagnostics {
		switch d.Severity {
		case validation.SeverityError:
			errors++
		case validation.SeverityWarning:
			warnings++
		case validation.SeverityInfo:
			infos++
		}
	}

	if errors == 0 && warnings == 0 && infos == 0 {
		return Success("Validation passed with no issues")
	}

	parts := []string{}
	if errors > 0 {
		parts = append(parts, fmt.Sprintf("%d error(s)", errors))
	}
	if warnings > 0 {
		parts = append(parts, fmt.Sprintf("%d warning(s)", warnings))
	}
	if infos > 0 {
		parts = append(parts, fmt.Sprintf("%d info(s)", infos))
	}

	return fmt.Sprintf("Found %s", strings.Join(parts, ", "))
}

// Count returns the total number of diagnostics
func (r *DiagnosticsRenderer) Count() int {
	return len(r.diagnostics)
}
