package reporters

import "github.com/rudderlabs/rudder-iac/cli/internal/ui"

// ProgressSyncReporter implements SyncReporter interface and provides progress reporting using a UI task reporter.
// It displays spinners for running tasks as well as a progress bar for overall sync progress.
type ProgressSyncReporter struct {
	planReporter
	reporter *ui.TaskReporter
}

func (r *ProgressSyncReporter) AskConfirmation() (bool, error) {
	return ui.Confirm("Do you want to apply these changes?")
}

func (r *ProgressSyncReporter) SyncStarted(totalTasks int) {
	r.reporter = ui.NewTaskReporter(totalTasks)
	go func() {
		err := r.reporter.Run()
		if err != nil {
			ui.PrintError(err)
		}
	}()
}

func (r *ProgressSyncReporter) SyncCompleted() {
	r.reporter.Done()
}

func (r *ProgressSyncReporter) TaskStarted(taskId string, description string) {
	r.reporter.Start(taskId, description)
}

func (r *ProgressSyncReporter) TaskCompleted(taskId string, description string, err error) {
	r.reporter.Complete(taskId, description, err)
}
