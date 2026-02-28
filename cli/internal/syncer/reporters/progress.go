package reporters

import (
	"fmt"
	"io"
	"os"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

// ProgressSyncReporter implements SyncReporter interface and provides progress reporting using a UI task reporter.
// It displays spinners for running tasks as well as a progress bar for overall sync progress.
type ProgressSyncReporter struct {
	planReporter
	reporter  *ui.TaskReporter
	errWriter io.Writer
}

func (r *ProgressSyncReporter) SetErrWriter(errWriter io.Writer) {
	r.errWriter = errWriter
}

func (r *ProgressSyncReporter) getErrWriter() io.Writer {
	if r.errWriter == nil {
		return os.Stderr
	}
	return r.errWriter
}

func (r *ProgressSyncReporter) AskConfirmation() (bool, error) {
	return ui.Confirm("Do you want to apply these changes?")
}

// ConfirmNameMatches prompts the user to confirm linking name-matched resources.
// Returns the matches that the user confirmed for linking.
func (r *ProgressSyncReporter) ConfirmNameMatches(matches []differ.NameMatchCandidate) []differ.NameMatchCandidate {
	if len(matches) == 0 {
		return nil
	}

	fmt.Fprintln(r.getWriter())
	fmt.Fprintln(r.getWriter(), ui.Bold("The following local resources match existing remote resources by name:"))
	for _, match := range matches {
		fmt.Fprintf(r.getWriter(), "  - %s → %s (remote: %s)\n",
			ui.Color(match.LocalURN, ui.ColorWhite),
			ui.Color(match.RemoteName, ui.ColorGreen),
			ui.Color(match.RemoteID, ui.ColorBlue),
		)
	}
	fmt.Fprintln(r.getWriter())

	confirmed, err := ui.Confirm("Link these resources? (This will import them instead of creating new ones)")
	if err != nil || !confirmed {
		return nil
	}

	return matches
}

func (r *ProgressSyncReporter) SyncStarted(totalTasks int) {
	r.reporter = ui.NewTaskReporter(totalTasks)
	go func() {
		err := r.reporter.Run()
		if err != nil {
			fmt.Fprintln(r.getErrWriter(), ui.Error(err))
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
