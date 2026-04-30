package apply

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkspaceBanner(t *testing.T) {
	assert.Equal(t, "Workspace: Test Workspace (ws_123)", workspaceBanner("Test Workspace", "ws_123"))
}
