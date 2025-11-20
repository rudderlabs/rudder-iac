package ui

import (
	"fmt"
	"io"
	"os"
)

var uiWriter io.Writer = os.Stdout
var uiErrWriter io.Writer = os.Stderr

// Writer returns the current writer used by the UI package.
func Writer() io.Writer {
	return uiWriter
}

func ErrWriter() io.Writer {
	return uiErrWriter
}

// SetWriter sets the writer used by the UI package. Useful for redirecting output in tests.
func SetWriter(w io.Writer) {
	uiWriter = w
}

// SetErrWriter sets the error writer used by the UI package. Useful for redirecting error output in tests.
func SetErrWriter(w io.Writer) {
	uiErrWriter = w
}

// ResetWriter resets the UI package writer to the default os.Stdout.
func ResetWriter() {
	uiWriter = os.Stdout
}

// ResetErrWriter resets the UI package error writer to the default os.Stderr.
func ResetErrWriter() {
	uiErrWriter = os.Stderr
}

func Print(a ...interface{}) {
	fmt.Fprint(uiWriter, a...)
}

func Println(a ...interface{}) {
	fmt.Fprintln(uiWriter, a...)
}

func Printf(format string, a ...interface{}) {
	fmt.Fprintf(uiWriter, format, a...)
}
