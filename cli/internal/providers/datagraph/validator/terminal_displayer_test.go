package validator

import (
	"bytes"
	"fmt"
	"testing"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/stretchr/testify/assert"
)

func TestDisplayTerminal_AllPassed(t *testing.T) {
	report := &ValidationReport{
		Status: RunStatusExecuted,
		Resources: []*ResourceValidation{
			{ID: "user", DisplayName: "User Model", ResourceType: "model", Issues: nil},
			{ID: "user-orders", DisplayName: "User Orders", ResourceType: "relationship", Issues: nil},
		},
	}

	var buf bytes.Buffer
	NewTerminalDisplayer(&buf).Display(report)
	output := buf.String()

	assert.Contains(t, output, "Data Graph Validation Report")
	assert.Contains(t, output, "MODELS")
	assert.Contains(t, output, "User Model")
	assert.Contains(t, output, "pass")
	assert.Contains(t, output, "RELATIONSHIPS")
	assert.Contains(t, output, "User Orders")
	assert.Contains(t, output, "Result: PASSED")
}

func TestDisplayTerminal_WithErrors(t *testing.T) {
	report := &ValidationReport{
		Status: RunStatusExecuted,
		Resources: []*ResourceValidation{
			{
				ID: "user", DisplayName: "User Model", ResourceType: "model",
				Issues: []dgClient.ValidationIssue{
					{Rule: "model/table-exists", Severity: "error", Message: "Table does not exist"},
				},
			},
		},
	}

	var buf bytes.Buffer
	NewTerminalDisplayer(&buf).Display(report)
	output := buf.String()

	assert.Contains(t, output, "1 error")
	assert.Contains(t, output, "model/table-exists")
	assert.Contains(t, output, "Table does not exist")
	assert.Contains(t, output, "Result: FAILED")
}

func TestDisplayTerminal_WithWarnings(t *testing.T) {
	report := &ValidationReport{
		Status: RunStatusExecuted,
		Resources: []*ResourceValidation{
			{
				ID: "user", DisplayName: "User Model", ResourceType: "model",
				Issues: []dgClient.ValidationIssue{
					{Rule: "model/table-has-recent-data", Severity: "warning", Message: "Table has no data in last 30 days"},
				},
			},
		},
	}

	var buf bytes.Buffer
	NewTerminalDisplayer(&buf).Display(report)
	output := buf.String()

	assert.Contains(t, output, "1 warning")
	assert.Contains(t, output, "Result: PASSED")
}

func TestDisplayTerminal_WithExecutionError(t *testing.T) {
	report := &ValidationReport{
		Status: RunStatusExecuted,
		Resources: []*ResourceValidation{
			{
				ID: "user", DisplayName: "User Model", ResourceType: "model",
				Err: fmt.Errorf("connection refused"),
			},
		},
	}

	var buf bytes.Buffer
	NewTerminalDisplayer(&buf).Display(report)
	output := buf.String()

	assert.Contains(t, output, "connection refused")
	assert.Contains(t, output, "Result: FAILED")
}

func TestDisplayTerminal_FallsBackToID(t *testing.T) {
	report := &ValidationReport{
		Status: RunStatusExecuted,
		Resources: []*ResourceValidation{
			{ID: "user", ResourceType: "model", Issues: nil},
		},
	}

	var buf bytes.Buffer
	NewTerminalDisplayer(&buf).Display(report)
	output := buf.String()

	assert.Contains(t, output, "user")
	assert.Contains(t, output, "pass")
}

func TestDisplayTerminal_MultipleErrorsAndWarnings(t *testing.T) {
	report := &ValidationReport{
		Status: RunStatusExecuted,
		Resources: []*ResourceValidation{
			{
				ID: "user", DisplayName: "User Model", ResourceType: "model",
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
	output := buf.String()

	assert.Contains(t, output, "2 errors")
	assert.Contains(t, output, "1 warning")
}
