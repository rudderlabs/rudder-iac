package validation

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/pathindex"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
)

func TestDiagnosticCollector_Add(t *testing.T) {
	collector := newDiagnosticCollector()
	assert.Empty(t, collector.getAll())

	// Add single diagnostic
	d1 := Diagnostic{
		RuleID:   "test/rule1",
		Severity: rules.Error,
		Message:  "error message",
		File:     "file1.yaml",
		Position: pathindex.Position{Line: 1, Column: 1},
	}
	collector.add(d1)
	assert.Len(t, collector.getAll(), 1)

	// Add multiple diagnostics at once
	d2 := Diagnostic{
		RuleID:   "test/rule2",
		Severity: rules.Warning,
		Message:  "warning message",
		File:     "file2.yaml",
		Position: pathindex.Position{Line: 2, Column: 2},
	}
	d3 := Diagnostic{
		RuleID:   "test/rule3",
		Severity: rules.Info,
		Message:  "info message",
		File:     "file3.yaml",
		Position: pathindex.Position{Line: 3, Column: 3},
	}
	collector.add(d2, d3)
	assert.Len(t, collector.getAll(), 3)
}

func TestDiagnosticCollector_GetErrors(t *testing.T) {
	collector := newDiagnosticCollector()

	// Add diagnostics with different severities
	collector.add(
		Diagnostic{RuleID: "error1", Severity: rules.Error, File: "file1.yaml"},
		Diagnostic{RuleID: "warning1", Severity: rules.Warning, File: "file2.yaml"},
		Diagnostic{RuleID: "error2", Severity: rules.Error, File: "file3.yaml"},
		Diagnostic{RuleID: "info1", Severity: rules.Info, File: "file4.yaml"},
	)

	errors := collector.getErrors()
	assert.Len(t, errors, 2)
	assert.Equal(t, "error1", errors[0].RuleID)
	assert.Equal(t, "error2", errors[1].RuleID)
}

func TestDiagnosticCollector_GetWarnings(t *testing.T) {
	collector := newDiagnosticCollector()

	// Add diagnostics with different severities
	collector.add(
		Diagnostic{RuleID: "error1", Severity: rules.Error, File: "file1.yaml"},
		Diagnostic{RuleID: "warning1", Severity: rules.Warning, File: "file2.yaml"},
		Diagnostic{RuleID: "warning2", Severity: rules.Warning, File: "file3.yaml"},
		Diagnostic{RuleID: "info1", Severity: rules.Info, File: "file4.yaml"},
	)

	warnings := collector.getWarnings()
	assert.Len(t, warnings, 2)
	assert.Equal(t, "warning1", warnings[0].RuleID)
	assert.Equal(t, "warning2", warnings[1].RuleID)
}

func TestDiagnosticCollector_HasErrors(t *testing.T) {
	collector := newDiagnosticCollector()
	assert.False(t, collector.hasErrors())

	// Add warning - should not have errors
	collector.add(Diagnostic{RuleID: "warning1", Severity: rules.Warning})
	assert.False(t, collector.hasErrors())

	// Add error - should have errors
	collector.add(Diagnostic{RuleID: "error1", Severity: rules.Error})
	assert.True(t, collector.hasErrors())
}

func TestDiagnosticCollector_SortDiagnostics(t *testing.T) {
	collector := newDiagnosticCollector()

	// Add diagnostics in unsorted order
	collector.add(
		Diagnostic{
			RuleID:   "d3",
			File:     "b.yaml",
			Position: pathindex.Position{Line: 1, Column: 1},
		},
		Diagnostic{
			RuleID:   "d1",
			File:     "a.yaml",
			Position: pathindex.Position{Line: 2, Column: 1},
		},
		Diagnostic{
			RuleID:   "d2",
			File:     "a.yaml",
			Position: pathindex.Position{Line: 1, Column: 2},
		},
		Diagnostic{
			RuleID:   "d4",
			File:     "a.yaml",
			Position: pathindex.Position{Line: 1, Column: 1},
		},
	)

	collector.sortDiagnostics()
	diagnostics := collector.getAll()

	// Should be sorted by file, then line, then column
	assert.Equal(t, "d4", diagnostics[0].RuleID) // a.yaml:1:1
	assert.Equal(t, "d2", diagnostics[1].RuleID) // a.yaml:1:2
	assert.Equal(t, "d1", diagnostics[2].RuleID) // a.yaml:2:1
	assert.Equal(t, "d3", diagnostics[3].RuleID) // b.yaml:1:1
}

func TestDiagnosticCollector_EmptyCollector(t *testing.T) {
	collector := newDiagnosticCollector()

	assert.Empty(t, collector.getAll())
	assert.Empty(t, collector.getErrors())
	assert.Empty(t, collector.getWarnings())
	assert.False(t, collector.hasErrors())

	// Sorting empty collector should not panic
	collector.sortDiagnostics()
	assert.Empty(t, collector.getAll())
}
