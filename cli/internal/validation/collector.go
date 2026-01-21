package validation

import (
	"sort"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// diagnosticCollector aggregates diagnostics during validation.
// It is an internal helper used by the validation engine to collect
// and manage diagnostics from multiple rules across multiple files.
// Not exposed in the public ValidationEngine API.
type diagnosticCollector struct {
	diagnostics []Diagnostic
}

// newDiagnosticCollector creates a new empty diagnostic collector.
func newDiagnosticCollector() *diagnosticCollector {
	return &diagnosticCollector{
		diagnostics: make([]Diagnostic, 0),
	}
}

// add adds one or more diagnostics to the collector.
func (dc *diagnosticCollector) add(diagnostics ...Diagnostic) {
	dc.diagnostics = append(dc.diagnostics, diagnostics...)
}

// getAll returns all collected diagnostics.
func (dc *diagnosticCollector) getAll() []Diagnostic {
	return dc.diagnostics
}

// getErrors returns only diagnostics with Error severity.
func (dc *diagnosticCollector) getErrors() []Diagnostic {
	errors := make([]Diagnostic, 0)
	for _, d := range dc.diagnostics {
		if d.Severity == rules.Error {
			errors = append(errors, d)
		}
	}
	return errors
}

// getWarnings returns only diagnostics with Warning severity.
func (dc *diagnosticCollector) getWarnings() []Diagnostic {
	warnings := make([]Diagnostic, 0)
	for _, d := range dc.diagnostics {
		if d.Severity == rules.Warning {
			warnings = append(warnings, d)
		}
	}
	return warnings
}

// hasErrors returns true if any Error-severity diagnostics exist.
func (dc *diagnosticCollector) hasErrors() bool {
	for _, d := range dc.diagnostics {
		if d.Severity == rules.Error {
			return true
		}
	}
	return false
}

// sortDiagnostics sorts diagnostics by file path, then line, then column.
// This provides consistent ordering for output.
func (dc *diagnosticCollector) sortDiagnostics() {
	sort.Slice(dc.diagnostics, func(i, j int) bool {

		var (
			d1 = dc.diagnostics[i]
			d2 = dc.diagnostics[j]
		)

		if d1.File != d2.File {
			return d1.File < d2.File
		}

		if d1.Position.Line != d2.Position.Line {
			return d1.Position.Line < d2.Position.Line
		}

		return d1.Position.Column < d2.Position.Column
	})
}
