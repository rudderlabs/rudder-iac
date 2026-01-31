package validation

import (
	"sort"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/pathindex"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// Diagnostic represents a validation issue found in a spec file.
// It contains complete information including the resolved position (line, column)
// in the file where the issue was detected. Diagnostics are created by the
// validation engine by converting ValidationResults and resolving their Reference
// fields using PathIndexer.
type Diagnostic struct {
	// RuleID is the unique identifier of the rule that generated this diagnostic
	// Convention: "provider/kind/rule-name" (e.g., "datacatalog/properties/unique-name")
	RuleID string

	// Severity indicates the level of the diagnostic (Error, Warning, Info)
	Severity rules.Severity

	// Message is the human-readable description of the issue
	Message string

	// File is the absolute path to the spec file where the issue was found
	File string

	// Position contains the resolved location information (line, column, lineText)
	// This is populated by the engine using PathIndexer, so renderers don't need
	// to do any file I/O or position lookup
	Position pathindex.Position

	// Examples provides valid and invalid usage examples for this rule
	// Can be nil if the rule doesn't provide examples
	Examples rules.Examples
}

// Diagnostics is a collection of Diagnostic items with utility methods.
type Diagnostics []Diagnostic

// HasErrors returns true if any diagnostic in the collection has Error severity.
func (d Diagnostics) HasErrors() bool {
	for _, diag := range d {
		if diag.Severity == rules.Error {
			return true
		}
	}
	return false
}

// HasWarnings returns true if any diagnostic in the collection has Warning severity.
func (d Diagnostics) HasWarnings() bool {
	for _, diag := range d {
		if diag.Severity == rules.Warning {
			return true
		}
	}
	return false
}

func (d Diagnostics) Sort() {
	sort.Slice(d, func(i, j int) bool {
		if d[i].File != d[j].File {
			return d[i].File < d[j].File
		}

		if d[i].Position.Line != d[j].Position.Line {
			return d[i].Position.Line < d[j].Position.Line
		}

		return d[i].Position.Column < d[j].Position.Column
	})
}

// Errors returns only the diagnostics with Error severity.
func (d Diagnostics) Errors() Diagnostics {
	errors := make(Diagnostics, 0)
	for _, diag := range d {
		if diag.Severity == rules.Error {
			errors = append(errors, diag)
		}
	}
	return errors
}

// Warnings returns only the diagnostics with Warning severity.
func (d Diagnostics) Warnings() Diagnostics {
	warnings := make(Diagnostics, 0)
	for _, diag := range d {
		if diag.Severity == rules.Warning {
			warnings = append(warnings, diag)
		}
	}
	return warnings
}
