package reporters

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

type PlainSyncReporter struct {
	planReporter
}

// AskConfirmation in PlainSyncReporter is a no-op and always returns false, since in non-interactive environments we cannot prompt the user.
// This typically runs in CI/CD pipelines or when output is being redirected to a file,
// which assumes non-interactive execution with --confirm=false for the apply command.
func (r *PlainSyncReporter) AskConfirmation() (bool, error) { return false, nil }

func (r *PlainSyncReporter) TaskCompleted(_ string, description string, err error) {
	if err != nil {
		fmt.Fprintln(r.getWriter(), ui.Failure(description))
	} else {
		fmt.Fprintln(r.getWriter(), ui.Success(description))
	}
}

func (r *PlainSyncReporter) SyncStarted(totalTasks int) {
	// no output expected
}

func (r *PlainSyncReporter) SyncCompleted() {
	// no output expected
}

func (r *PlainSyncReporter) TaskStarted(taskId string, description string) {
	// no output expected
}
