package dev

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	"golang.org/x/term"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

// output prints one command result. With --json it prints the server bytes
// on stdout and errors as one JSON object on stderr; otherwise a table or
// text, like every other rudder-cli command.
type output struct {
	stdout  io.Writer
	stderr  io.Writer
	machine bool
}

func newOutput(stdout, stderr io.Writer, jsonFlag bool) output {
	return output{stdout: stdout, stderr: stderr, machine: jsonFlag}
}

// result prints raw in machine mode, or calls human otherwise.
func (o output) result(raw []byte, human func(io.Writer)) error {
	if !o.machine {
		human(o.stdout)
		return nil
	}
	_, err := fmt.Fprintln(o.stdout, string(bytes.TrimSpace(raw)))
	return err
}

// fail prints err and returns a SilentError, so the root prints nothing
// more. Machine mode: one JSON object on stderr. Human mode: Error and Next.
func (o output) fail(err error) error {
	apiErr := asAPIError(err)
	if o.machine {
		line, _ := json.Marshal(map[string]any{"error": apiErr})
		fmt.Fprintln(o.stderr, string(line))
		return &cmderrors.SilentError{Err: err}
	}
	fmt.Fprintln(o.stderr, ui.Error(fmt.Errorf("%s: %s", apiErr.Code, apiErr.Message)))
	if apiErr.Next != "" {
		fmt.Fprintln(o.stderr, "Next: "+apiErr.Next)
	}
	return &cmderrors.SilentError{Err: err}
}

// hint prints a human next step on stderr, on a terminal only.
func (o output) hint(format string, args ...any) {
	if !o.machine && isTerminal(o.stderr) {
		fmt.Fprintf(o.stderr, format+"\n", args...)
	}
}

// page prints an /events or /requests page. A page whose first request
// alone exceeds maxBytes is an output_limit failure.
func (o output) page(raw []byte, items int, t *devlisten.Truncated, human func(io.Writer)) error {
	if t != nil && t.RequestBytes > 0 && items == 0 {
		return o.outputLimit(raw, t, human)
	}
	return o.result(raw, human)
}

// outputLimit handles a page whose first request alone is larger than
// maxBytes: stdout keeps the page, stderr gets the error, exit 1.
func (o output) outputLimit(raw []byte, t *devlisten.Truncated, human func(io.Writer)) error {
	if err := o.result(raw, human); err != nil {
		return err
	}
	return o.fail(&cliError{Code: "output_limit",
		Message: "one request is " + strconv.Itoa(t.RequestBytes) + " bytes, above --max-bytes",
		Next:    t.Next})
}

// isTerminal checks the writer itself: ui.IsTerminal checks stdout only.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

// clean escapes control characters in a captured value before it reaches a
// terminal.
func clean(s string, maxLen int) string {
	quoted := strconv.Quote(s)
	quoted = quoted[1 : len(quoted)-1]
	if maxLen > 0 && len(quoted) > maxLen {
		quoted = quoted[:maxLen] + "..."
	}
	return quoted
}
