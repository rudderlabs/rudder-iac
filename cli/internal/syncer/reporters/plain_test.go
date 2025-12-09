package reporters_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/reporters"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/stretchr/testify/assert"
)

func TestPlainSyncReporter(t *testing.T) {
	var buf bytes.Buffer
	r := &reporters.PlainSyncReporter{}
	r.SetWriter(&buf)

	confirmed, err := r.AskConfirmation()
	assert.NoError(t, err)
	assert.False(t, confirmed)

	r.TaskCompleted("task1", "Task 1", nil)
	r.TaskCompleted("task2", "Task 2", fmt.Errorf("some error"))
	assert.Equal(t, buf.String(), ui.Success("Task 1")+"\n"+ui.Failure("Task 2")+"\n")
}
