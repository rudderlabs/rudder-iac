package validator

import (
	"bytes"
	"fmt"
	"testing"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/stretchr/testify/assert"
)

// Non-TTY terminal width is 80, statusCol = 60.
// lipgloss renders no ANSI codes in non-TTY, so ui.Bold/ui.Color are identity functions.
// Padding is computed via lipgloss.Width (display width), not len (byte count).

const (
	doubleLine = "================================================================================\n"
	singleLine = "--------------------------------------------------------------------------------\n"
)

func TestDisplayTerminal_AllPassed(t *testing.T) {
	report := &ValidationReport{
		Status: RunStatusExecuted,
		Resources: []*ResourceValidation{
			{ID: "user", URN: "data-graph-model:user", DisplayName: "User Model", ResourceType: "model", Issues: nil},
			{ID: "user-orders", URN: "data-graph-relationship:user-orders", DisplayName: "User Orders", ResourceType: "relationship", Issues: nil},
		},
	}

	var buf bytes.Buffer
	NewTerminalDisplayer(&buf).Display(report)

	expected := "\n" +
		"Data Graph Validation Report\n" +
		doubleLine +
		"\n" +
		"MODELS\n" +
		singleLine +
		"  ✓  User Model (data-graph-model:user)                     pass\n" +
		"\n" +
		"RELATIONSHIPS\n" +
		singleLine +
		"  ✓  User Orders (data-graph-relationship:user-orders)      pass\n" +
		"\n" +
		"SUMMARY\n" +
		doubleLine +
		"Models:         1 passed   0 errors   0 warnings\n" +
		"Relationships:  1 passed   0 errors   0 warnings\n" +
		singleLine +
		"Result: PASSED\n" +
		doubleLine

	assert.Equal(t, expected, buf.String())
}

func TestDisplayTerminal_WithErrors(t *testing.T) {
	report := &ValidationReport{
		Status: RunStatusExecuted,
		Resources: []*ResourceValidation{
			{
				ID: "user", URN: "data-graph-model:user", DisplayName: "User Model", ResourceType: "model",
				Issues: []dgClient.ValidationIssue{
					{Rule: "model/table-exists", Severity: "error", Message: "Table does not exist"},
				},
			},
		},
	}

	var buf bytes.Buffer
	NewTerminalDisplayer(&buf).Display(report)

	expected := "\n" +
		"Data Graph Validation Report\n" +
		doubleLine +
		"\n" +
		"MODELS\n" +
		singleLine +
		"  ✕  User Model (data-graph-model:user)                     1 error\n" +
		"       model/table-exists: Table does not exist\n" +
		"\n" +
		"SUMMARY\n" +
		doubleLine +
		"Models:         0 passed   1 errors   0 warnings\n" +
		singleLine +
		"Result: FAILED\n" +
		doubleLine

	assert.Equal(t, expected, buf.String())
}

func TestDisplayTerminal_WithWarnings(t *testing.T) {
	report := &ValidationReport{
		Status: RunStatusExecuted,
		Resources: []*ResourceValidation{
			{
				ID: "user", URN: "data-graph-model:user", DisplayName: "User Model", ResourceType: "model",
				Issues: []dgClient.ValidationIssue{
					{Rule: "model/table-has-recent-data", Severity: "warning", Message: "Table has no data in last 30 days"},
				},
			},
		},
	}

	var buf bytes.Buffer
	NewTerminalDisplayer(&buf).Display(report)

	expected := "\n" +
		"Data Graph Validation Report\n" +
		doubleLine +
		"\n" +
		"MODELS\n" +
		singleLine +
		"  ⚠  User Model (data-graph-model:user)                     1 warning\n" +
		"       model/table-has-recent-data: Table has no data in last 30 days\n" +
		"\n" +
		"SUMMARY\n" +
		doubleLine +
		"Models:         0 passed   0 errors   1 warnings\n" +
		singleLine +
		"Result: PASSED\n" +
		doubleLine

	assert.Equal(t, expected, buf.String())
}

func TestDisplayTerminal_WithExecutionError(t *testing.T) {
	report := &ValidationReport{
		Status: RunStatusExecuted,
		Resources: []*ResourceValidation{
			{
				ID: "user", URN: "data-graph-model:user", DisplayName: "User Model", ResourceType: "model",
				Err: fmt.Errorf("connection refused"),
			},
		},
	}

	var buf bytes.Buffer
	NewTerminalDisplayer(&buf).Display(report)

	expected := "\n" +
		"Data Graph Validation Report\n" +
		doubleLine +
		"\n" +
		"MODELS\n" +
		singleLine +
		"  ✕  User Model (data-graph-model:user)                     error\n" +
		"       connection refused\n" +
		"\n" +
		"SUMMARY\n" +
		doubleLine +
		"Models:         0 passed   1 errors   0 warnings\n" +
		singleLine +
		"Result: FAILED\n" +
		doubleLine

	assert.Equal(t, expected, buf.String())
}

func TestDisplayTerminal_FallsBackToID(t *testing.T) {
	report := &ValidationReport{
		Status: RunStatusExecuted,
		Resources: []*ResourceValidation{
			{ID: "user", URN: "data-graph-model:user", ResourceType: "model", Issues: nil},
		},
	}

	var buf bytes.Buffer
	NewTerminalDisplayer(&buf).Display(report)

	expected := "\n" +
		"Data Graph Validation Report\n" +
		doubleLine +
		"\n" +
		"MODELS\n" +
		singleLine +
		"  ✓  user (data-graph-model:user)                           pass\n" +
		"\n" +
		"SUMMARY\n" +
		doubleLine +
		"Models:         1 passed   0 errors   0 warnings\n" +
		singleLine +
		"Result: PASSED\n" +
		doubleLine

	assert.Equal(t, expected, buf.String())
}

func TestDisplayTerminal_MultipleErrorsAndWarnings(t *testing.T) {
	report := &ValidationReport{
		Status: RunStatusExecuted,
		Resources: []*ResourceValidation{
			{
				ID: "user", URN: "data-graph-model:user", DisplayName: "User Model", ResourceType: "model",
				Issues: []dgClient.ValidationIssue{
					{Rule: "model/table-exists", Severity: "error", Message: "Table does not exist"},
					{Rule: "model/column-exists", Severity: "error", Message: "Column missing"},
					{Rule: "model/data-freshness", Severity: "warning", Message: "Stale data"},
				},
			},
		},
	}

	var buf bytes.Buffer
	NewTerminalDisplayer(&buf).Display(report)

	expected := "\n" +
		"Data Graph Validation Report\n" +
		doubleLine +
		"\n" +
		"MODELS\n" +
		singleLine +
		"  ✕  User Model (data-graph-model:user)                     2 errors  1 warning\n" +
		"       model/table-exists: Table does not exist\n" +
		"       model/column-exists: Column missing\n" +
		"       model/data-freshness: Stale data\n" +
		"\n" +
		"SUMMARY\n" +
		doubleLine +
		"Models:         0 passed   1 errors   0 warnings\n" +
		singleLine +
		"Result: FAILED\n" +
		doubleLine

	assert.Equal(t, expected, buf.String())
}
