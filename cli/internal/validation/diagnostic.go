package validation

import (
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
