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

func TestWorkspaceInfoRows(t *testing.T) {
	dataPlaneURL := "https://dataplane.example.com"

	tests := []struct {
		name      string
		workspace *client.Workspace
		expected  []table.Row
	}{
		{
			name: "with data plane URL",
			workspace: &client.Workspace{
				ID:           "ws_123",
				Name:         "Prod",
				Environment:  "PRODUCTION",
				Status:       "ACTIVE",
				Region:       "US",
				DataPlaneURL: &dataPlaneURL,
			},
			expected: []table.Row{
				{"WorkspaceID", "ws_123"},
				{"Name", "Prod"},
				{"Environment", "PRODUCTION"},
				{"Status", "ACTIVE"},
				{"Region", "US"},
				{"Data plane URL", "https://dataplane.example.com"},
			},
		},
		{
			name: "without data plane URL",
			workspace: &client.Workspace{
				ID:          "ws_456",
				Name:        "Dev",
				Environment: "DEVELOPMENT",
				Status:      "ACTIVE",
				Region:      "EU",
			},
			expected: []table.Row{
				{"WorkspaceID", "ws_456"},
				{"Name", "Dev"},
				{"Environment", "DEVELOPMENT"},
				{"Status", "ACTIVE"},
				{"Region", "EU"},
				{"Data plane URL", ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, workspaceInfoRows(tt.workspace))
		})
	}
}

func TestPrintWorkspaceInfoJSON(t *testing.T) {
	dataPlaneURL := "https://dataplane.example.com"

	tests := []struct {
		name      string
		workspace *client.Workspace
		expected  map[string]string
	}{
		{
			name: "with data plane URL",
			workspace: &client.Workspace{
				ID:           "ws_123",
				Name:         "Prod",
				Environment:  "PRODUCTION",
				Status:       "ACTIVE",
				Region:       "US",
				DataPlaneURL: &dataPlaneURL,
			},
			expected: map[string]string{
				"workspaceID":  "ws_123",
				"name":         "Prod",
				"environment":  "PRODUCTION",
				"status":       "ACTIVE",
				"region":       "US",
				"dataPlaneURL": "https://dataplane.example.com",
			},
		},
		{
			name: "without data plane URL",
			workspace: &client.Workspace{
				ID:          "ws_456",
				Name:        "Dev",
				Environment: "DEVELOPMENT",
				Status:      "ACTIVE",
				Region:      "EU",
			},
			expected: map[string]string{
				"workspaceID":  "ws_456",
				"name":         "Dev",
				"environment":  "DEVELOPMENT",
				"status":       "ACTIVE",
				"region":       "EU",
				"dataPlaneURL": "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			var buf bytes.Buffer
			cmd.SetOut(&buf)

			require.NoError(t, printWorkspaceInfoJSON(cmd, tt.workspace))

			var out map[string]string
			require.NoError(t, json.Unmarshal(buf.Bytes(), &out))

			assert.Equal(t, tt.expected, out)
		})
	}
}
