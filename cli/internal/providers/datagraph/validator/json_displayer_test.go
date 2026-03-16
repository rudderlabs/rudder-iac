package validator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisplayJSON(t *testing.T) {
	report := &ValidationReport{
		Status: RunStatusExecuted,
		Resources: []*ResourceValidation{
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
	NewJSONDisplayer(&buf).Display(report)
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
	report := &ValidationReport{
		Status: RunStatusExecuted,
		Resources: []*ResourceValidation{
			{
				ID: "user", DisplayName: "User Model", ResourceType: "model",
				Err: fmt.Errorf("api timeout"),
			},
		},
	}

	var buf bytes.Buffer
	NewJSONDisplayer(&buf).Display(report)
	output := buf.String()

	var parsed map[string]any
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	resources := parsed["resources"].([]any)
	first := resources[0].(map[string]any)
	assert.Equal(t, "error", first["status"])
	assert.Equal(t, "api timeout", first["error"])
}

func TestDisplayJSON_WithWarning(t *testing.T) {
	report := &ValidationReport{
		Status: RunStatusExecuted,
		Resources: []*ResourceValidation{
			{
				ID: "user", DisplayName: "User Model", ResourceType: "model",
				Issues: []dgClient.ValidationIssue{
					{Rule: "model/data-freshness", Severity: "warning", Message: "Stale data"},
				},
			},
		},
	}

	var buf bytes.Buffer
	NewJSONDisplayer(&buf).Display(report)
	output := buf.String()

	var parsed map[string]any
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	resources := parsed["resources"].([]any)
	first := resources[0].(map[string]any)
	assert.Equal(t, "warning", first["status"])
}
