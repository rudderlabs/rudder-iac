package reporters_test

import (
	"bytes"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/reporters"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/stretchr/testify/assert"
)

func TestPlainSyncReporter(t *testing.T) {
	r := &reporters.PlainSyncReporter{}

	var buf bytes.Buffer
	ui.SetWriter(&buf)
	defer ui.ResetWriter()

	confirmed, err := r.AskConfirmation()
	assert.NoError(t, err)
	assert.False(t, confirmed)

	r.TaskCompleted("task1", "Task 1", nil)
	assert.Equal(t, buf.String(), ui.Success("Task 1")+"\n")
}
