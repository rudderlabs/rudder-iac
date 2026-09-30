package dev

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/itchyny/gojq"
	"golang.org/x/term"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

// defaultJQTimeout bounds --jq when no --timeout is given.
const defaultJQTimeout = 10 * time.Second

// output prints one command result. Machine mode (--json) prints the
// server bytes, or the --jq results, on stdout and errors as one JSON
// object on stderr. Human mode prints a table or text.
type output struct {
	stdout io.Writer
	stderr io.Writer
	flags  clientFlags
}

// result prints raw in machine mode, or calls human otherwise. note, when
// set, goes to stderr once under --jq, because jq may hide the flag that
// explains a short result.
func (o output) result(ctx context.Context, raw []byte, note string, human func(io.Writer)) error {
	if !o.flags.json {
		human(o.stdout)
		return nil
	}
	if o.flags.jq == "" {
		_, err := fmt.Fprintln(o.stdout, string(bytes.TrimSpace(raw)))
		return err
	}
	if err := o.runJQ(ctx, raw); err != nil {
		return o.fail(err)
	}
	if note != "" {
		fmt.Fprintln(o.stderr, "note: "+note)
	}
	return nil
}

func (o output) runJQ(ctx context.Context, raw []byte) error {
	code, err := compileJQ(o.flags.jq)
	if err != nil {
		return err
	}
	var input any
	if err := json.Unmarshal(raw, &input); err != nil {
		return fmt.Errorf("decoding response for --jq: %w", err)
	}
	timeout := o.flags.timeout
	if timeout == 0 {
		timeout = defaultJQTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	iter := code.RunWithContext(ctx, input)
	for v, ok := iter.Next(); ok; v, ok = iter.Next() {
		if err, isErr := v.(error); isErr {
			return &cliError{Code: "jq_error", Message: err.Error(), Next: "rudder-cli dev --help"}
		}
		if err := printJQValue(o.stdout, v); err != nil {
			return err
		}
	}
	return nil
}

// compileJQ compiles without an environ or module loader: $ENV is empty
// and import fails.
func compileJQ(expr string) (*gojq.Code, error) {
	query, err := gojq.Parse(expr)
	if err != nil {
		return nil, usageError("rudder-cli dev --help", "--jq: %v", err)
	}
	code, err := gojq.Compile(query)
	if err != nil {
		return nil, usageError("rudder-cli dev --help", "--jq: %v", err)
	}
	return code, nil
}

// printJQValue prints strings raw and other values as one JSON line, as
// gh --jq does.
func printJQValue(w io.Writer, v any) error {
	if s, ok := v.(string); ok {
		_, err := fmt.Fprintln(w, s)
		return err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encoding --jq result: %w", err)
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// fail prints err and returns a SilentError, so the root prints nothing
// more. Machine mode: one JSON object on stderr. Human mode: Error and Next.
func (o output) fail(err error) error {
	apiErr := asAPIError(err)
	if o.flags.json {
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
	if !o.flags.json && isTerminal(o.stderr) {
		fmt.Fprintf(o.stderr, format+"\n", args...)
	}
}

// page prints an /events or /requests page. A page whose first request
// alone exceeds maxBytes is an output_limit failure.
func (o output) page(ctx context.Context, raw []byte, items int, timedOut bool, t *devlisten.Truncated,
	human func(io.Writer),
) error {
	if t != nil && t.RequestBytes > 0 && items == 0 {
		return o.outputLimit(ctx, raw, t, human)
	}
	return o.result(ctx, raw, pageNote(timedOut, t), human)
}

// outputLimit handles a page whose first request alone is larger than
// maxBytes: stdout keeps the page, stderr gets the error, exit 1.
func (o output) outputLimit(ctx context.Context, raw []byte, t *devlisten.Truncated, human func(io.Writer)) error {
	if err := o.result(ctx, raw, "", human); err != nil {
		return err
	}
	return o.fail(&cliError{Code: "output_limit",
		Message: "one request is " + strconv.Itoa(t.RequestBytes) + " bytes, above --max-bytes",
		Next:    t.Next})
}

func pageNote(timedOut bool, t *devlisten.Truncated) string {
	var notes []string
	if timedOut {
		notes = append(notes, "timedOut: the wait ended before --min matches arrived")
	}
	if t != nil {
		notes = append(notes, "truncated: the page hit --max-bytes; next: "+t.Next)
	}
	return strings.Join(notes, "; ")
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
