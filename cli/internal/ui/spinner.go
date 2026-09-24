package ui

import (
	"os"
	"time"

	"github.com/briandowns/spinner"
)

// One spinner per process, like gh's progress indicator. Stderr keeps stdout
// clean for --json and piping; the library no-ops when stderr is not a TTY.
// ponytail: not goroutine-safe, every Start/Stop runs on the command goroutine.
var current *spinner.Spinner

// StartSpinner replaces any running spinner with a new message.
func StartSpinner(message string) {
	StopSpinner()
	current = spinner.New(spinner.CharSets[14], 100*time.Millisecond,
		spinner.WithSuffix(" "+message), spinner.WithWriterFile(os.Stderr))
	current.Start()
}

func StopSpinner() {
	if current != nil {
		current.Stop()
		current = nil
	}
}
