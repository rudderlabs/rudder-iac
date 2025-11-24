package ui

import (
	"fmt"
	"io"
	"os"
)

var uiWriter io.Writer = os.Stdout
var uiErrWriter io.Writer = os.Stderr

func Print(a ...interface{}) {
	fmt.Fprint(uiWriter, a...)
}

func Println(a ...interface{}) {
	fmt.Fprintln(uiWriter, a...)
}

func Printf(format string, a ...interface{}) {
	fmt.Fprintf(uiWriter, format, a...)
}
