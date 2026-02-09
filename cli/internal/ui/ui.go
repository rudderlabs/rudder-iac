package ui

import (
	"fmt"
	"io"
	"os"
)

var uiWriter io.Writer = os.Stdout
var uiErrWriter io.Writer = os.Stderr
var originalWriter io.Writer = os.Stdout

func Print(a ...interface{}) {
	fmt.Fprint(uiWriter, a...)
}

func Println(a ...interface{}) {
	fmt.Fprintln(uiWriter, a...)
}

func Printf(format string, a ...interface{}) {
	fmt.Fprintf(uiWriter, format, a...)
}

// SetWriter sets the writer for UI output (for testing)
func SetWriter(w io.Writer) {
	originalWriter = uiWriter
	uiWriter = w
}

// RestoreWriter restores the original writer (for testing)
func RestoreWriter() {
	uiWriter = originalWriter
}
