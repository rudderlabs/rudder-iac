package workspace

import (
	"testing"

	"github.com/charmbracelet/bubbles/table"
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/stretchr/testify/assert"
)

func TestWorkspaceInfoRows_WithDataPlaneURL(t *testing.T) {
	dataPlaneURL := "https://dataplane.example.com"
	workspace := &client.Workspace{
		ID:           "ws_123",
		Name:         "Prod",
		Environment:  "PRODUCTION",
		Status:       "ACTIVE",
		Region:       "US",
		DataPlaneURL: &dataPlaneURL,
	}

	assert.Equal(t, []table.Row{
		{"WorkspaceID", "ws_123"},
		{"Name", "Prod"},
		{"Environment", "PRODUCTION"},
		{"Status", "ACTIVE"},
		{"Region", "US"},
		{"Data plane URL", "https://dataplane.example.com"},
	}, workspaceInfoRows(workspace))
}

func TestWorkspaceInfoRows_WithoutDataPlaneURL(t *testing.T) {
	workspace := &client.Workspace{
		ID:          "ws_456",
		Name:        "Dev",
		Environment: "DEVELOPMENT",
		Status:      "ACTIVE",
		Region:      "EU",
	}

	assert.Equal(t, []table.Row{
		{"WorkspaceID", "ws_456"},
		{"Name", "Dev"},
		{"Environment", "DEVELOPMENT"},
		{"Status", "ACTIVE"},
		{"Region", "EU"},
		{"Data plane URL", ""},
	}, workspaceInfoRows(workspace))
}
