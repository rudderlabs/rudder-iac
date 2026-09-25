package ui

import (
	"os"
	"sync"
	"time"

	"github.com/briandowns/spinner"
)

// One spinner per process, like gh's progress indicator. Stderr keeps stdout
// clean for --json and piping; the library no-ops when stderr is not a TTY.
// The mutex is for parallel tests that load projects concurrently.
var (
	spinnerMu sync.Mutex
	current   *spinner.Spinner
)

// StartSpinner replaces any running spinner with a new message.
func StartSpinner(message string) {
	spinnerMu.Lock()
	defer spinnerMu.Unlock()
	stopLocked()
	current = spinner.New(spinner.CharSets[14], 100*time.Millisecond,
		spinner.WithSuffix(" "+message), spinner.WithWriterFile(os.Stderr),
		spinner.WithHiddenCursor(false))
	current.Start()
}

// StopSpinner clears the running spinner, if any.
func StopSpinner() {
	spinnerMu.Lock()
	defer spinnerMu.Unlock()
	stopLocked()
}

func stopLocked() {
	if current != nil {
		current.Stop()
		current = nil
	}
}
