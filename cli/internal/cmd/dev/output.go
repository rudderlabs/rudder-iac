package dev

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/itchyny/gojq"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

const (
	// defaultJQTimeout bounds --jq when no --timeout is given.
	defaultJQTimeout = 10 * time.Second
	// jqCeiling is the page size a --jq command fetches: --max-bytes then
	// applies to the jq output, so a projection sees the whole page.
	jqCeiling = 4 << 20
)

// output prints one command result. Machine mode (--json) prints the
// server bytes, or the --jq results, on stdout and errors as one JSON
// object on stderr. Human mode prints a table or text.
type output struct {
	stdout io.Writer
	stderr io.Writer
	flags  clientFlags
	// jqCap is --max-bytes for the --jq output, 0 for no cap; jqRetry is
	// the command that lifts it.
	jqCap   int
	jqRetry string
}

// withJQCap applies --max-bytes to the --jq output of a page command.
func (o output) withJQCap(cmd *cobra.Command, args []string, maxBytes int) output {
	o.jqCap = maxBytes
	o.jqRetry = commandLine(cmd, args, "max-bytes") + " --max-bytes 0"
	return o
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
	return o.jqResult(ctx, raw, note)
}

// jqResult runs --jq, then writes the trailer and the note. An output_limit
// still writes them: the printed lines are a valid partial result.
func (o output) jqResult(ctx context.Context, raw []byte, note string) error {
	err := o.runJQ(ctx, raw)
	var cliErr *cliError
	if err != nil && !(errors.As(err, &cliErr) && cliErr.Code == "output_limit") {
		return o.fail(err)
	}
	if line := trailer(raw); line != "" {
		fmt.Fprintln(o.stderr, line)
	}
	if note != "" {
		fmt.Fprintln(o.stderr, "note: "+note)
	}
	if err != nil {
		return o.fail(err)
	}
	return nil
}

// trailer is the one stderr line --jq adds, so a projection never loses the
// cursor: cursor=N serverId=X hasMore=B timedOut=B, each key when the
// response has it.
func trailer(raw []byte) string {
	var env struct {
		Cursor   *uint64 `json:"cursor"`
		ServerID *string `json:"serverId"`
		HasMore  *bool   `json:"hasMore"`
		TimedOut *bool   `json:"timedOut"`
	}
	if json.Unmarshal(raw, &env) != nil || env.Cursor == nil {
		return ""
	}
	parts := []string{"cursor=" + strconv.FormatUint(*env.Cursor, 10)}
	if env.ServerID != nil {
		parts = append(parts, "serverId="+*env.ServerID)
	}
	if env.HasMore != nil {
		parts = append(parts, "hasMore="+strconv.FormatBool(*env.HasMore))
	}
	if env.TimedOut != nil {
		parts = append(parts, "timedOut="+strconv.FormatBool(*env.TimedOut))
	}
	return strings.Join(parts, " ")
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

	var buf bytes.Buffer
	iter := code.RunWithContext(ctx, input)
	for v, ok := iter.Next(); ok; v, ok = iter.Next() {
		if err, isErr := v.(error); isErr {
			return &cliError{Code: "jq_error", Message: err.Error(), Next: "rudder-cli dev --help"}
		}
		if err := printJQValue(&buf, v); err != nil {
			return err
		}
	}
	return o.writeCapped(buf.Bytes())
}

// writeCapped writes the whole lines that fit in jqCap, then fails with
// output_limit when some did not.
func (o output) writeCapped(b []byte) error {
	if o.jqCap <= 0 || len(b) <= o.jqCap {
		_, err := o.stdout.Write(b)
		return err
	}
	cut := bytes.LastIndexByte(b[:o.jqCap], '\n') + 1
	if _, err := o.stdout.Write(b[:cut]); err != nil {
		return err
	}
	return &cliError{Code: "output_limit",
		Message: fmt.Sprintf("the --jq output is %d bytes, above --max-bytes %d; %d bytes were printed", len(b), o.jqCap, cut),
		Next:    o.jqRetry}
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
	return o.result(ctx, raw, pageNote(timedOut, t, o.flags.jq != ""), human)
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

func pageNote(timedOut bool, t *devlisten.Truncated, jq bool) string {
	var notes []string
	if timedOut {
		notes = append(notes, "timedOut: the wait ended before --min matches arrived")
	}
	switch {
	case t != nil && jq:
		notes = append(notes, fmt.Sprintf("truncated: --jq read the page up to its %d-byte ceiling; next: %s",
			jqCeiling, t.Next))
	case t != nil:
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
