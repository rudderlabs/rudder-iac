package telemetry

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rudderlabs/rudder-iac/api/client"
)

func TestEventPropsWorkspace(t *testing.T) {
	t.Cleanup(func() { workspaceID.Store("") })

	t.Run("a command that never looked the workspace up names none", func(t *testing.T) {
		workspaceID.Store("")
		SetWorkspace(nil)
		props := eventProps("validate", nil)
		assert.NotContains(t, props, "workspace_id")
		assert.Equal(t, false, props["errored"])
	})

	t.Run("a command that signed in names its workspace", func(t *testing.T) {
		SetWorkspace(&client.Workspace{ID: "ws-123"})
		props := eventProps("apply", errors.New("boom"), KV{K: "dryRun", V: true})
		assert.Equal(t, "ws-123", props["workspace_id"])
		assert.Equal(t, "apply", props["command"])
		assert.Equal(t, true, props["errored"])
		assert.Equal(t, true, props["dryRun"])
	})
}
