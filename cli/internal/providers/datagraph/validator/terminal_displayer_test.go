package validator

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/stretchr/testify/assert"
)

const testTerminalWidth = 80 // ui.GetTerminalWidth() returns 80 in non-TTY

// buildPaddedLine replicates the padding logic of printWithPadding.
func buildPaddedLine(left, right string, statusCol int) string {
	padding := max(statusCol-len(left), 1)
	return fmt.Sprintf("%s%s%s\n", left, strings.Repeat(" ", padding), right)
}

func separator(char string) string {
	return strings.Repeat(char, testTerminalWidth) + "\n"
}

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

	statusCol := testTerminalWidth * 3 / 4

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(ui.Bold("Data Graph Validation Report") + "\n")
	sb.WriteString(separator("="))
	// MODELS section
	sb.WriteString("\n")
	sb.WriteString(ui.Bold("MODELS") + "\n")
	sb.WriteString(separator("-"))
	sb.WriteString(buildPaddedLine(
		fmt.Sprintf("  %s  %s", ui.Color(symbolPass, ui.ColorGreen), "User Model"),
		"pass",
		statusCol,
	))
	// RELATIONSHIPS section
	sb.WriteString("\n")
	sb.WriteString(ui.Bold("RELATIONSHIPS") + "\n")
	sb.WriteString(separator("-"))
	sb.WriteString(buildPaddedLine(
		fmt.Sprintf("  %s  %s", ui.Color(symbolPass, ui.ColorGreen), "User Orders"),
		"pass",
		statusCol,
	))
	// SUMMARY
	sb.WriteString("\n")
	sb.WriteString(ui.Bold("SUMMARY") + "\n")
	sb.WriteString(separator("="))
	sb.WriteString("Models:         1 passed   0 errors   0 warnings\n")
	sb.WriteString("Relationships:  1 passed   0 errors   0 warnings\n")
	sb.WriteString(separator("-"))
	sb.WriteString(ui.Color("Result: PASSED", ui.ColorGreen) + "\n")
	sb.WriteString(separator("="))

	assert.Equal(t, sb.String(), buf.String())
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

	statusCol := testTerminalWidth * 3 / 4

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(ui.Bold("Data Graph Validation Report") + "\n")
	sb.WriteString(separator("="))
	// MODELS section
	sb.WriteString("\n")
	sb.WriteString(ui.Bold("MODELS") + "\n")
	sb.WriteString(separator("-"))
	sb.WriteString(buildPaddedLine(
		fmt.Sprintf("  %s  %s", ui.Color(symbolError, ui.ColorRed), "User Model"),
		"1 error",
		statusCol,
	))
	sb.WriteString(fmt.Sprintf("       %s: %s\n", ui.Color("model/table-exists", ui.ColorRed), "Table does not exist"))
	// SUMMARY
	sb.WriteString("\n")
	sb.WriteString(ui.Bold("SUMMARY") + "\n")
	sb.WriteString(separator("="))
	sb.WriteString("Models:         0 passed   1 errors   0 warnings\n")
	sb.WriteString(separator("-"))
	sb.WriteString(ui.Color("Result: FAILED", ui.ColorRed) + "\n")
	sb.WriteString(separator("="))

	assert.Equal(t, sb.String(), buf.String())
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

	statusCol := testTerminalWidth * 3 / 4

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(ui.Bold("Data Graph Validation Report") + "\n")
	sb.WriteString(separator("="))
	// MODELS section
	sb.WriteString("\n")
	sb.WriteString(ui.Bold("MODELS") + "\n")
	sb.WriteString(separator("-"))
	sb.WriteString(buildPaddedLine(
		fmt.Sprintf("  %s  %s", ui.Color(symbolWarning, ui.ColorYellow), "User Model"),
		"1 warning",
		statusCol,
	))
	sb.WriteString(fmt.Sprintf("       %s: %s\n", ui.Color("model/table-has-recent-data", ui.ColorYellow), "Table has no data in last 30 days"))
	// SUMMARY
	sb.WriteString("\n")
	sb.WriteString(ui.Bold("SUMMARY") + "\n")
	sb.WriteString(separator("="))
	sb.WriteString("Models:         0 passed   0 errors   1 warnings\n")
	sb.WriteString(separator("-"))
	sb.WriteString(ui.Color("Result: PASSED", ui.ColorGreen) + "\n")
	sb.WriteString(separator("="))

	assert.Equal(t, sb.String(), buf.String())
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

	statusCol := testTerminalWidth * 3 / 4

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(ui.Bold("Data Graph Validation Report") + "\n")
	sb.WriteString(separator("="))
	// MODELS section
	sb.WriteString("\n")
	sb.WriteString(ui.Bold("MODELS") + "\n")
	sb.WriteString(separator("-"))
	sb.WriteString(buildPaddedLine(
		fmt.Sprintf("  %s  %s", ui.Color(symbolError, ui.ColorRed), "User Model"),
		ui.Color("error", ui.ColorRed),
		statusCol,
	))
	sb.WriteString("       connection refused\n")
	// SUMMARY
	sb.WriteString("\n")
	sb.WriteString(ui.Bold("SUMMARY") + "\n")
	sb.WriteString(separator("="))
	sb.WriteString("Models:         0 passed   1 errors   0 warnings\n")
	sb.WriteString(separator("-"))
	sb.WriteString(ui.Color("Result: FAILED", ui.ColorRed) + "\n")
	sb.WriteString(separator("="))

	assert.Equal(t, sb.String(), buf.String())
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

	statusCol := testTerminalWidth * 3 / 4

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(ui.Bold("Data Graph Validation Report") + "\n")
	sb.WriteString(separator("="))
	// MODELS section
	sb.WriteString("\n")
	sb.WriteString(ui.Bold("MODELS") + "\n")
	sb.WriteString(separator("-"))
	sb.WriteString(buildPaddedLine(
		fmt.Sprintf("  %s  %s", ui.Color(symbolPass, ui.ColorGreen), "user"),
		"pass",
		statusCol,
	))
	// SUMMARY
	sb.WriteString("\n")
	sb.WriteString(ui.Bold("SUMMARY") + "\n")
	sb.WriteString(separator("="))
	sb.WriteString("Models:         1 passed   0 errors   0 warnings\n")
	sb.WriteString(separator("-"))
	sb.WriteString(ui.Color("Result: PASSED", ui.ColorGreen) + "\n")
	sb.WriteString(separator("="))

	assert.Equal(t, sb.String(), buf.String())
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

	statusCol := testTerminalWidth * 3 / 4

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(ui.Bold("Data Graph Validation Report") + "\n")
	sb.WriteString(separator("="))
	// MODELS section
	sb.WriteString("\n")
	sb.WriteString(ui.Bold("MODELS") + "\n")
	sb.WriteString(separator("-"))
	sb.WriteString(buildPaddedLine(
		fmt.Sprintf("  %s  %s", ui.Color(symbolError, ui.ColorRed), "User Model"),
		"2 errors  1 warning",
		statusCol,
	))
	sb.WriteString(fmt.Sprintf("       %s: %s\n", ui.Color("model/table-exists", ui.ColorRed), "Table does not exist"))
	sb.WriteString(fmt.Sprintf("       %s: %s\n", ui.Color("model/column-exists", ui.ColorRed), "Column missing"))
	sb.WriteString(fmt.Sprintf("       %s: %s\n", ui.Color("model/data-freshness", ui.ColorYellow), "Stale data"))
	// SUMMARY
	sb.WriteString("\n")
	sb.WriteString(ui.Bold("SUMMARY") + "\n")
	sb.WriteString(separator("="))
	sb.WriteString("Models:         0 passed   1 errors   0 warnings\n")
	sb.WriteString(separator("-"))
	sb.WriteString(ui.Color("Result: FAILED", ui.ColorRed) + "\n")
	sb.WriteString(separator("="))

	assert.Equal(t, sb.String(), buf.String())
}
