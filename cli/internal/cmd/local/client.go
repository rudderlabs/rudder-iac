package local

import (
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
	target := base + "/_local/v1/events"
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
		return nil, nil, fail(cmd, &usageError{code: "server_unreachable", message: message, next: "rudder-cli local event-stream serve --help"})
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fail(cmd, &usageError{
			code: "server_unreachable", message: "reading the answer of " + base + ": " + err.Error(),
			next: "rudder-cli local event-stream serve --help",
		})
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, serverError(cmd, base, resp.StatusCode, body)
	}
	// A dev server with an HTML fallback answers 200 on any path; only the
	// listener names itself.
	if resp.Header.Get("X-Local-Server-Id") == "" {
		return nil, nil, fail(cmd, &usageError{
			code:    "server_unreachable",
			message: base + " answered HTTP 200, not as local event-stream serve answers; check the url of the ready line",
			next:    "rudder-cli local event-stream serve --help",
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
			message: fmt.Sprintf("%s answered HTTP %d, not as local event-stream serve answers; check the url of the ready line", base, status),
			next:    "rudder-cli local event-stream serve --help",
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

// readSummary prints the summary object as the server wrote it, with the
// URL added to every rudder-cli next, or as text.
func readSummary(cmd *cobra.Command, opts *readOptions, base string, q url.Values, timeout time.Duration) error {
	_, body, err := get(cmd, base, q, timeout)
	if err != nil {
		return err
	}
	noteEviction(cmd, opts.since, gjson.GetBytes(body, "evictedThrough").Uint())

	if !opts.json {
		var s summaryText
		if err := json.Unmarshal(body, &s); err != nil {
			return fail(cmd, &usageError{
				code: "server_unreachable", message: "the summary does not parse: " + err.Error(),
				next: "rudder-cli local event-stream serve --help",
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
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

// readStream prints the stream as the server wrote it, or as a table. The
// cursor to continue from goes to stderr, so stdout stays one event per line.
func readStream(cmd *cobra.Command, opts *readOptions, base string, q url.Values, timeout time.Duration, view string, fields []string) error {
	resp, body, err := get(cmd, base, q, timeout)
	if err != nil {
		return err
	}
	hasMore := resp.Header.Get("X-Local-Has-More") == "true"
	cursor := resp.Header.Get("X-Local-Cursor")
	if opts.json {
		if hasMore {
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

// noteEviction warns that the window lost requests to the store budget, so
// a count may be short.
func noteEviction(cmd *cobra.Command, since string, evictedThrough uint64) {
	cursor, err := strconv.ParseUint(since, 10, 64)
	if err != nil || evictedThrough <= cursor {
		return
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "note: requests up to seq %d were evicted; the window starts after them\n", evictedThrough)
}
