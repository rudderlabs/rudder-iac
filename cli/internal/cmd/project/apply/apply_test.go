package apply

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/stretchr/testify/assert"
)

func TestWorkspaceHeader(t *testing.T) {
	tests := []struct {
		name      string
		workspace *client.Workspace
		expected  string
	}{
		{
			name: "formats workspace name and ID",
			workspace: &client.Workspace{
				Name: "Production",
				ID:   "ws_123abc",
			},
			expected: "Workspace: Production (ws_123abc)\n",
		},
		{
			name: "handles empty fields",
			workspace: &client.Workspace{
				Name: "",
				ID:   "",
			},
			expected: "Workspace:  ()\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, workspaceHeader(tt.workspace))
		})
	}
}
