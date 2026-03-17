package validator

import (
	"bytes"
	"fmt"
	"testing"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/stretchr/testify/assert"
)

func TestDisplayJSON(t *testing.T) {
	report := &ValidationReport{
		Status: RunStatusExecuted,
		Resources: []*ResourceValidation{
			{
				ID: "user", URN: "data-graph-model:user", DisplayName: "User Model", ResourceType: "model",
				Issues: []dgClient.ValidationIssue{
					{Rule: "model/table-exists", Severity: "error", Message: "Table does not exist"},
				},
			},
			{
				ID: "purchase", URN: "data-graph-model:purchase", DisplayName: "Purchase", ResourceType: "model",
				Issues: nil,
			},
			{
				ID: "order", URN: "data-graph-model:order", DisplayName: "Order", ResourceType: "model",
				Issues: []dgClient.ValidationIssue{
					{Rule: "model/table-has-recent-data", Severity: "warning", Message: "Stale data"},
				},
			},
			{
				ID: "user-orders", URN: "data-graph-relationship:user-orders", DisplayName: "User Orders", ResourceType: "relationship",
				Err: fmt.Errorf("api timeout"),
			},
		},
	}

	var buf bytes.Buffer
	NewJSONDisplayer(&buf).Display(report)

	expected := `{
  "resources": [
    {
      "id": "user",
      "urn": "data-graph-model:user",
      "displayName": "User Model",
      "resourceType": "model",
      "status": "failed",
      "issues": [
        {
          "rule": "model/table-exists",
          "severity": "error",
          "message": "Table does not exist"
        }
      ]
    },
    {
      "id": "purchase",
      "urn": "data-graph-model:purchase",
      "displayName": "Purchase",
      "resourceType": "model",
      "status": "passed"
    },
    {
      "id": "order",
      "urn": "data-graph-model:order",
      "displayName": "Order",
      "resourceType": "model",
      "status": "warning",
      "issues": [
        {
          "rule": "model/table-has-recent-data",
          "severity": "warning",
          "message": "Stale data"
        }
      ]
    },
    {
      "id": "user-orders",
      "urn": "data-graph-relationship:user-orders",
      "displayName": "User Orders",
      "resourceType": "relationship",
      "status": "error",
      "error": "api timeout"
    }
  ]
}
`
	assert.Equal(t, expected, buf.String())
}

func TestDisplayJSON_NoResources(t *testing.T) {
	report := &ValidationReport{
		Status:    RunStatusNoResources,
		Resources: nil,
	}

	var buf bytes.Buffer
	NewJSONDisplayer(&buf).Display(report)

	expected := `{
  "resources": []
}
`
	assert.Equal(t, expected, buf.String())
}
