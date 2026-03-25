package workspace

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/charmbracelet/bubbles/table"
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestPrintWorkspaceInfoJSON(t *testing.T) {
	dataPlaneURL := "https://dataplane.example.com"
	workspace := &client.Workspace{
		ID:           "ws_123",
		Name:         "Prod",
		Environment:  "PRODUCTION",
		Status:       "ACTIVE",
		Region:       "US",
		DataPlaneURL: &dataPlaneURL,
	}

	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	require.NoError(t, printWorkspaceInfoJSON(cmd, workspace))

	var out map[string]string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out))

	assert.Equal(t, map[string]string{
		"workspaceID":  "ws_123",
		"name":         "Prod",
		"environment":  "PRODUCTION",
		"status":       "ACTIVE",
		"region":       "US",
		"dataPlaneURL": "https://dataplane.example.com",
	}, out)
}

func TestPrintWorkspaceInfoJSON_WithoutDataPlaneURL(t *testing.T) {
	workspace := &client.Workspace{
		ID:          "ws_456",
		Name:        "Dev",
		Environment: "DEVELOPMENT",
		Status:      "ACTIVE",
		Region:      "EU",
	}

	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	require.NoError(t, printWorkspaceInfoJSON(cmd, workspace))

	var out map[string]string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out))

	assert.Equal(t, map[string]string{
		"workspaceID":  "ws_456",
		"name":         "Dev",
		"environment":  "DEVELOPMENT",
		"status":       "ACTIVE",
		"region":       "EU",
		"dataPlaneURL": "",
	}, out)
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
