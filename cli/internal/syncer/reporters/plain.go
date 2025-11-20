package reporters

import "github.com/rudderlabs/rudder-iac/cli/internal/ui"

type PlainSyncReporter struct {
	planReporter
}

// AskConfirmation in PlainSyncReporter is a no-op and always returns false, since in non-interactive environments we cannot prompt the user.
// This typically runs in CI/CD pipelines or when output is being redirected to a file,
// which assumes non-interactive execution with --confirm=false for the apply command.
func (r *PlainSyncReporter) AskConfirmation() (bool, error) { return false, nil }

func (r *PlainSyncReporter) TaskCompleted(taskId string, description string, err error) {
	if err != nil {
		ui.PrintFailure(description)
	} else {
		ui.PrintSuccess(description)
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
