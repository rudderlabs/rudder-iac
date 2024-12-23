package ui

import (
	"time"

	"github.com/briandowns/spinner"
)

func NewSpinner(message string) *spinner.Spinner {
	return spinner.New(spinner.CharSets[14], 100*time.Millisecond, spinner.WithSuffix(" "+message))
}

func WithSpinner(message string, f func()) {
	s := NewSpinner(message)
	s.Start()
	defer s.Stop()
	f()
}
