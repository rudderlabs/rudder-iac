package display

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/validations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisplayTerminal_AllPassed(t *testing.T) {
	results := &validations.ValidationResults{
		Status: validations.RunStatusExecuted,
		Resources: []*validations.ResourceValidation{
			{ID: "user", DisplayName: "User Model", ResourceType: "model", Issues: nil},
			{ID: "user-orders", DisplayName: "User Orders", ResourceType: "relationship", Issues: nil},
		},
	}

	var buf bytes.Buffer
	NewValidationDisplayer(&buf, false).Display(results)
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
	results := &validations.ValidationResults{
		Status: validations.RunStatusExecuted,
		Resources: []*validations.ResourceValidation{
			{
				ID: "user", DisplayName: "User Model", ResourceType: "model",
				Issues: []dgClient.ValidationIssue{
					{Rule: "model/table-exists", Severity: "error", Message: "Table does not exist"},
				},
			},
		},
	}

	var buf bytes.Buffer
	NewValidationDisplayer(&buf, false).Display(results)
	output := buf.String()

	assert.Contains(t, output, "1 error")
	assert.Contains(t, output, "model/table-exists")
	assert.Contains(t, output, "Table does not exist")
	assert.Contains(t, output, "Result: FAILED")
}

func TestDisplayTerminal_WithWarnings(t *testing.T) {
	results := &validations.ValidationResults{
		Status: validations.RunStatusExecuted,
		Resources: []*validations.ResourceValidation{
			{
				ID: "user", DisplayName: "User Model", ResourceType: "model",
				Issues: []dgClient.ValidationIssue{
					{Rule: "model/table-has-recent-data", Severity: "warning", Message: "Table has no data in last 30 days"},
				},
			},
		},
	}

	var buf bytes.Buffer
	NewValidationDisplayer(&buf, false).Display(results)
	output := buf.String()

	assert.Contains(t, output, "1 warning")
	assert.Contains(t, output, "Result: PASSED")
}

func TestDisplayTerminal_WithExecutionError(t *testing.T) {
	results := &validations.ValidationResults{
		Status: validations.RunStatusExecuted,
		Resources: []*validations.ResourceValidation{
			{
				ID: "user", DisplayName: "User Model", ResourceType: "model",
				Err: fmt.Errorf("connection refused"),
			},
		},
	}

	var buf bytes.Buffer
	NewValidationDisplayer(&buf, false).Display(results)
	output := buf.String()

	assert.Contains(t, output, "connection refused")
	assert.Contains(t, output, "Result: FAILED")
}

func TestDisplayJSON(t *testing.T) {
	results := &validations.ValidationResults{
		Status: validations.RunStatusExecuted,
		Resources: []*validations.ResourceValidation{
			{
				ID: "user", DisplayName: "User Model", ResourceType: "model",
				Issues: []dgClient.ValidationIssue{
					{Rule: "model/table-exists", Severity: "error", Message: "Table does not exist"},
				},
			},
			{
				ID: "purchase", DisplayName: "Purchase", ResourceType: "model",
				Issues: nil,
			},
		},
	}

	var buf bytes.Buffer
	NewValidationDisplayer(&buf, true).Display(results)
	output := buf.String()

	var parsed map[string]any
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "executed", parsed["status"])

	resources := parsed["resources"].([]any)
	assert.Len(t, resources, 2)

	first := resources[0].(map[string]any)
	assert.Equal(t, "user", first["id"])
	assert.Equal(t, "failed", first["status"])

	second := resources[1].(map[string]any)
	assert.Equal(t, "purchase", second["id"])
	assert.Equal(t, "passed", second["status"])
}

func TestDisplayJSON_WithError(t *testing.T) {
	results := &validations.ValidationResults{
		Status: validations.RunStatusExecuted,
		Resources: []*validations.ResourceValidation{
			{
				ID: "user", DisplayName: "User Model", ResourceType: "model",
				Err: fmt.Errorf("api timeout"),
			},
		},
	}

	var buf bytes.Buffer
	NewValidationDisplayer(&buf, true).Display(results)
	output := buf.String()

	var parsed map[string]any
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	resources := parsed["resources"].([]any)
	first := resources[0].(map[string]any)
	assert.Equal(t, "error", first["status"])
	assert.Equal(t, "api timeout", first["error"])
}
