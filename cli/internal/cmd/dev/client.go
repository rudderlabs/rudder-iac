package dev

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten"
)

// get sends the one request of a read command and returns the answer of a
// listener. Any other outcome is printed as an error.
func get(cmd *cobra.Command, base string, q url.Values, timeout time.Duration) (*http.Response, []byte, error) {
	target := base + "/_dev/v1/events"
	if len(q) > 0 {
		target += "?" + q.Encode()
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, nil, fail(cmd, &usageError{message: err.Error(), next: cmd.CommandPath() + " --help"})
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		message := "no listener answers at " + base
		if errors.Is(err, context.DeadlineExceeded) {
			message = fmt.Sprintf("the listener at %s did not answer within %s", base, timeout)
		}
		return nil, nil, fail(cmd, &usageError{code: "server_unreachable", message: message, next: "rudder-cli dev listen --help"})
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fail(cmd, &usageError{
			code: "server_unreachable", message: "reading the answer of " + base + ": " + err.Error(),
			next: "rudder-cli dev listen --help",
		})
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, serverError(cmd, base, resp.StatusCode, body)
	}
	// A dev server with an HTML fallback answers 200 on any path; only the
	// listener names itself.
	if resp.Header.Get("X-Dev-Server-Id") == "" {
		return nil, nil, fail(cmd, &usageError{
			code:    "server_unreachable",
			message: base + " answered HTTP 200, not as dev listen answers; check the url of the ready line",
			next:    "rudder-cli dev listen --help",
		})
	}
	return resp, body, nil
}

// serverError prints the listener's error object with the URL added to its
// next. An answer that is not one means the URL is not a listener.
func serverError(cmd *cobra.Command, base string, status int, body []byte) error {
	e := gjson.GetBytes(body, "error")
	if !e.IsObject() || !e.Get("code").Exists() {
		return fail(cmd, &usageError{
			code:    "server_unreachable",
			message: fmt.Sprintf("%s answered HTTP %d, not as dev listen answers; check the url of the ready line", base, status),
			next:    "rudder-cli dev listen --help",
		})
	}
	next := withURL(e.Get("next").String(), base)
	err := errors.New(e.Get("message").String())
	if !wantsJSON(cmd) {
		return fail(cmd, &usageError{message: err.Error(), next: next})
	}
	out, setErr := sjson.SetBytes(body, "error.next", next)
	if setErr != nil {
		out = body
	}
	_, _ = cmd.ErrOrStderr().Write(trimNewline(out))
	_, _ = cmd.ErrOrStderr().Write([]byte("\n"))
	return &cmderrors.SilentError{Err: err}
}

func trimNewline(b []byte) []byte {
	for len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	return b
}

// readSummary prints the summary, then fails a read that cannot back a
// check: a window that lost requests, or a wait that ran out.
func readSummary(cmd *cobra.Command, opts *readOptions, base string, q url.Values, timeout time.Duration) error {
	_, body, err := get(cmd, base, q, timeout)
	if err != nil {
		return err
	}
	if err := writeSummary(cmd, opts, base, body); err != nil {
		return err
	}
	return verdict(cmd, opts, base, readResult{
		lost:           gjson.GetBytes(body, `summary.diagnosis.#(code=="`+devlisten.CodeEvicted+`")`).Exists(),
		evictedThrough: gjson.GetBytes(body, "evictedThrough").String(),
		waitTimeout:    waitTimeout(cmd, body, base, q, opts.since),
	})
}

// waitTimeout is the error of a wait that ran out, or nil.
func waitTimeout(cmd *cobra.Command, body []byte, base string, q url.Values, since string) *usageError {
	if !gjson.GetBytes(body, "timedOut").Bool() {
		return nil
	}
	return &usageError{
		code: "wait_timeout",
		message: fmt.Sprintf("--wait ran out after %.1fs: %d of --min %s matching accepted events arrived",
			gjson.GetBytes(body, "waitedMs").Float()/1000, gjson.GetBytes(body, "summary.events.total").Int(), cmp.Or(q.Get("min"), "1")),
		next: again(cmd, base, since),
	}
}

// again is the command line the caller ran, with --url filled in and --since
// replaced, so a next command asks the same question of the same listener.
func again(cmd *cobra.Command, base, since string) string {
	words := []string{cmd.CommandPath(), "--url", shellWord(base), "--since", shellWord(since)}
	cmd.Flags().Visit(func(f *pflag.Flag) {
		slice, isSlice := f.Value.(pflag.SliceValue)
		switch {
		case f.Name == "url" || f.Name == "since":
		case isSlice:
			for _, v := range slice.GetSlice() {
				words = append(words, "--"+f.Name, shellWord(v))
			}
		case f.Value.Type() == "bool":
			// A bool takes no separate word: --json, or --json=false.
			words = append(words, strings.TrimSuffix("--"+f.Name+"="+f.Value.String(), "=true"))
		default:
			words = append(words, "--"+f.Name, shellWord(f.Value.String()))
		}
	})
	return strings.Join(words, " ")
}

// writeSummary prints the summary object as the server wrote it, with the
// URL added to every rudder-cli next, or as text.
func writeSummary(cmd *cobra.Command, opts *readOptions, base string, body []byte) error {
	if !opts.json {
		var s summaryText
		if err := json.Unmarshal(body, &s); err != nil {
			return fail(cmd, &usageError{
				code: "server_unreachable", message: "the summary does not parse: " + err.Error(),
				next: "rudder-cli dev listen --help",
			})
		}
		s.Next = withURL(s.Next, base)
		for i := range s.Summary.Diagnosis {
			s.Summary.Diagnosis[i].Next = withURL(s.Summary.Diagnosis[i].Next, base)
		}
		return s.render(cmd.OutOrStdout())
	}

	paths := []string{"next"}
	for i := range gjson.GetBytes(body, "summary.diagnosis.#").Int() {
		paths = append(paths, "summary.diagnosis."+strconv.FormatInt(i, 10)+".next")
	}
	for _, path := range paths {
		next := gjson.GetBytes(body, path)
		if !next.Exists() {
			continue
		}
		// SetBytes keeps the other keys and their order as they are.
		if out, err := sjson.SetBytes(body, path, withURL(next.String(), base)); err == nil {
			body = out
		}
	}
	_, err := cmd.OutOrStdout().Write(body)
	return err
}

// readStream prints the stream, then fails a window that lost requests.
func readStream(cmd *cobra.Command, opts *readOptions, base string, q url.Values, timeout time.Duration, view string, fields []string) error {
	resp, body, err := get(cmd, base, q, timeout)
	if err != nil {
		return err
	}
	if err := writeStream(cmd, opts, base, resp.Header, body, view, fields); err != nil {
		return err
	}
	return verdict(cmd, opts, base, readResult{
		lost:           resp.Header.Get("X-Dev-Diagnosis") == devlisten.CodeEvicted,
		evictedThrough: resp.Header.Get("X-Dev-Evicted-Through"),
	})
}

// writeStream prints the stream as the server wrote it, or as a table. The
// cursor to continue from goes to stderr, so stdout stays one event per line.
func writeStream(cmd *cobra.Command, opts *readOptions, base string, header http.Header, body []byte, view string, fields []string) error {
	hasMore := header.Get("X-Dev-Has-More") == "true"
	cursor := header.Get("X-Dev-Cursor")
	if opts.json {
		// A read that fails on loss writes one JSON error to stderr and nothing else.
		lost := header.Get("X-Dev-Diagnosis") == devlisten.CodeEvicted
		if hasMore && (!lost || opts.allowPartial) {
			fmt.Fprintf(cmd.ErrOrStderr(), "more events: continue with --since %s\n", cursor)
		}
		_, err := cmd.OutOrStdout().Write(body)
		return err
	}
	more := ""
	if hasMore {
		more = cursor
	}
	return renderStream(cmd.OutOrStdout(), body, streamTable{since: opts.since, base: base, more: more, view: view, fields: fields})
}

// readResult is what a read says about its own output. The listener
// decides which windows lost requests; the CLI only reads its answer.
type readResult struct {
	lost           bool
	evictedThrough string
	waitTimeout    *usageError
}

// verdict runs after the output is written. It fails a read whose output
// cannot back a check: a count that came out short, unless the caller
// accepted the loss, or a wait that ran out. The loss comes first, because
// it can explain a short count.
func verdict(cmd *cobra.Command, opts *readOptions, base string, r readResult) error {
	if r.lost {
		message := "requests up to seq " + r.evictedThrough + " were evicted, so this window is incomplete"
		if !opts.allowPartial {
			return fail(cmd, &usageError{
				code:    devlisten.CodeEvicted,
				message: message + "; pass --allow-partial to accept it",
				// The window after the newest evicted request is complete.
				next: again(cmd, base, r.evictedThrough),
			})
		}
		// With --json, stderr holds one JSON error, so the note gives way to it.
		if !opts.json || r.waitTimeout == nil {
			fmt.Fprintln(cmd.ErrOrStderr(), "note: "+message)
		}
	}
	if r.waitTimeout != nil {
		return fail(cmd, r.waitTimeout)
	}
	return nil
}
