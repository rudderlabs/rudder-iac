package ui

import (
	"time"

	"github.com/briandowns/spinner"
)

func NewSpinner(message string) *spinner.Spinner {
	return spinner.New(spinner.CharSets[14], 100*time.Millisecond, spinner.WithSuffix(" "+message))
}
