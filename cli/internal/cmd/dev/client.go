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
	"time"

	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
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
	if gjson.GetBytes(body, `summary.diagnosis.#(code=="`+codeEvicted+`")`).Exists() {
		if err := partial(cmd, opts, base, gjson.GetBytes(body, "evictedThrough").String()); err != nil {
			return err
		}
	}
	if !gjson.GetBytes(body, "timedOut").Bool() {
		return nil
	}
	return fail(cmd, &usageError{
		code: "wait_timeout",
		message: fmt.Sprintf("--wait ran out after %.1fs: %d of --min %s matching accepted events arrived",
			gjson.GetBytes(body, "waitedMs").Float()/1000, gjson.GetBytes(body, "summary.events.total").Int(), cmp.Or(q.Get("min"), "1")),
		next: "rudder-cli dev events --url " + shellWord(base) + " --since " + gjson.GetBytes(body, "since").String() + " --json",
	})
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

// readStream prints the stream as the server wrote it, or as a table. The
// cursor to continue from goes to stderr, so stdout stays one event per line.
func readStream(cmd *cobra.Command, opts *readOptions, base string, q url.Values, timeout time.Duration, view string, fields []string) error {
	resp, body, err := get(cmd, base, q, timeout)
	if err != nil {
		return err
	}
	hasMore := resp.Header.Get("X-Dev-Has-More") == "true"
	cursor := resp.Header.Get("X-Dev-Cursor")
	if opts.json {
		if hasMore {
			fmt.Fprintf(cmd.ErrOrStderr(), "more events: continue with --since %s\n", cursor)
		}
		_, err = cmd.OutOrStdout().Write(body)
	} else {
		more := ""
		if hasMore {
			more = cursor
		}
		err = renderStream(cmd.OutOrStdout(), body, streamTable{since: opts.since, base: base, more: more, view: view, fields: fields})
	}
	if err != nil || resp.Header.Get("X-Dev-Diagnosis") != codeEvicted {
		return err
	}
	return partial(cmd, opts, base, resp.Header.Get("X-Dev-Evicted-Through"))
}

// codeEvicted is the listener's code for a window that lost requests to the
// store budget. The listener decides which windows did; the CLI only reads it.
const codeEvicted = "events_evicted"

// partial runs after the output is written. It fails the read, because a
// count that came out short must not pass a check, unless the caller
// accepted the loss.
func partial(cmd *cobra.Command, opts *readOptions, base, evictedThrough string) error {
	message := "requests up to seq " + evictedThrough + " were evicted, so this window is incomplete"
	if opts.allowPartial {
		fmt.Fprintln(cmd.ErrOrStderr(), "note: "+message)
		return nil
	}
	return fail(cmd, &usageError{
		code:    codeEvicted,
		message: message + "; pass --allow-partial to accept it",
		// The window after the newest evicted request is complete.
		next: cmd.CommandPath() + " --url " + shellWord(base) + " --since " + evictedThrough,
	})
}
