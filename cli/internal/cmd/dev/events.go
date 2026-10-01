package dev

import (
	"cmp"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten"
)

const (
	// urlEnv lets a script set the listener URL once per shell.
	urlEnv = "RUDDERSTACK_DEV_URL"
	// placeholderURL stands in a next command when no URL is known yet.
	placeholderURL = "http://127.0.0.1:4321"
	// clientMargin is the time a read gets on top of its wait.
	clientMargin = 5 * time.Second
)

// readOptions are the flags both read commands share. Each filter is the
// query parameter of the same name.
type readOptions struct {
	url         string
	serverID    string
	timeout     time.Duration
	json        bool
	since       string
	events      []string
	types       []string
	userID      string
	anonymousID string
	writeKeys   []string
}

func addReadFlags(f *pflag.FlagSet, o *readOptions) {
	f.StringVar(&o.url, "url", "", "URL of the listener, from its ready line (default $"+urlEnv+")")
	f.StringVar(&o.serverID, "server-id", "", "Answer 409 server_changed unless the listener has this serverId")
	f.DurationVar(&o.timeout, "timeout", 0, "Give up on the listener after this long (default --wait plus 5s)")
	f.BoolVarP(&o.json, "json", "j", false, "Print the listener's JSON instead of a table")
	f.StringVar(&o.since, "since", "0", "Read after this cursor, or since a duration such as 5m or an RFC 3339 time")
	f.StringArrayVar(&o.events, "event", nil, "Only events with this name, or a prefix ending in *; repeat for more")
	f.StringArrayVar(&o.types, "type", nil, "Only events of this type, such as track or page; repeat for more")
	f.StringVar(&o.userID, "user-id", "", "Only events with this userId")
	f.StringVar(&o.anonymousID, "anonymous-id", "", "Only events with this anonymousId")
	f.StringArrayVar(&o.writeKeys, "write-key", nil,
		"Only requests sent with this write key; '' selects requests without one; repeat for more")
}

// query maps each flag the caller set to its parameter, and leaves the
// others to the server's defaults.
func (o *readOptions) query(f *pflag.FlagSet) url.Values {
	q := url.Values{}
	set := func(flag, param, value string) {
		if f.Changed(flag) {
			q.Set(param, value)
		}
	}
	set("since", "since", o.since)
	set("server-id", "serverId", o.serverID)
	set("user-id", "userId", o.userID)
	set("anonymous-id", "anonymousId", o.anonymousID)
	q["event"], q["type"], q["writeKey"] = o.events, o.types, o.writeKeys
	for k, v := range q {
		if v == nil {
			delete(q, k)
		}
	}
	return q
}

// listenerURL returns the URL from --url, else from the environment.
func (o *readOptions) listenerURL(cmd *cobra.Command) (string, *usageError) {
	raw := o.url
	if !cmd.Flags().Changed("url") {
		raw = os.Getenv(urlEnv)
	}
	if raw == "" {
		return "", &usageError{
			message: "no listener URL: pass --url or set " + urlEnv + " to the url of the ready line",
			next:    cmd.CommandPath() + " --url " + placeholderURL,
		}
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", &usageError{
			message: fmt.Sprintf("--url must be an http URL such as %s, got %q", placeholderURL, raw),
			next:    cmd.CommandPath() + " --url " + placeholderURL,
		}
	}
	return strings.TrimSuffix(raw, "/"), nil
}

// validate checks the shared flags with the server's own rules, so a bad
// value fails before the call.
func (o *readOptions) validate(cmd *cobra.Command) *usageError {
	if err := devlisten.ParseSince(o.since, time.Now()); err != nil {
		return &usageError{message: "--since: " + err.Error(), next: cmd.CommandPath() + " --help"}
	}
	return nil
}

func newCmdEvents() *cobra.Command {
	var (
		opts    readOptions
		wait    string
		atLeast int
	)
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Count and diagnose captured events",
		Long: heredoc.Doc(`
			Print the summary of what arrived: accepted events by name (byEvent), events in
			rejected requests (rejected), requests, write keys, control requests, a
			diagnosis, and the cursor. Filters narrow every count; the diagnosis covers the
			whole window. Each --event NAME and --write-key KEY you pass is listed with 0
			when nothing arrived, so a check is one jq -e call.

			--wait holds the call until --min matching accepted events exist, at most 110s.
			Every successful read exits 0, also when --wait runs out (timedOut is true in
			the JSON); errors exit 1.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli dev events --url http://127.0.0.1:4321

			# Assert a count in CI
			$ rudder-cli dev events --url "$url" --since "$cur" --event 'Order Completed' --json | jq -e '.summary.byEvent["Order Completed"] == 1'

			# Wait up to 30s for an event the app sends later
			$ rudder-cli dev events --url "$url" --since "$cur" --event 'Order Completed' --wait 30s --json
		`),
		Args: groupArgs,
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			defer func() { track("dev events", err) }()
			base, e := opts.listenerURL(cmd)
			if e == nil {
				e = opts.validate(cmd)
			}
			q := opts.query(cmd.Flags())
			q.Set("view", "counts")
			timeout := clientMargin
			if e == nil && cmd.Flags().Changed("wait") {
				d, err := devlisten.ParseWait(wait)
				if err != nil {
					e = &usageError{message: "--wait: " + err.Error(), next: cmd.CommandPath() + " --help"}
				}
				q.Set("wait", wait)
				timeout += d
			}
			if e == nil && cmd.Flags().Changed("min") {
				if atLeast < 1 {
					e = &usageError{message: fmt.Sprintf("--min must be 1 or more, got %d", atLeast), next: cmd.CommandPath() + " --help"}
				}
				q.Set("min", strconv.Itoa(atLeast))
			}
			if e != nil {
				e.next = withURL(e.next, base)
				return fail(cmd, e)
			}
			if opts.timeout > 0 {
				timeout = opts.timeout
			}
			return readSummary(cmd, &opts, base, q, timeout)
		},
	}
	f := cmd.Flags()
	addReadFlags(f, &opts)
	f.StringVar(&wait, "wait", "0s", "Hold the call until --min matching accepted events exist, at most 110s")
	f.IntVar(&atLeast, "min", 1, "The number of matching accepted events --wait waits for")

	cmd.AddCommand(newCmdEventsList())
	return cmd
}

func newCmdEventsList() *cobra.Command {
	var (
		opts   readOptions
		limit  int
		view   string
		fields []string
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Print captured events as the SDK sent them",
		Long: heredoc.Doc(`
			Print the accepted events after a cursor. With --json the output is NDJSON: one
			event per line, as the SDK sent it, with nothing added. Without --json it is a
			table. --view compact drops the SDK's auto-collected context; --fields PATH keeps
			only that dotted path (repeat it; it cannot be combined with --view).

			The stream carries no cursor. Take it from the ready line or from dev events
			--json. At most --limit events per page; when more are left, stderr names the
			--since to continue from. Events of rejected requests are not in the stream:
			dev events counts them.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli dev events list --url http://127.0.0.1:4321 --since 5m

			# The properties of one event
			$ rudder-cli dev events list --url "$url" --since "$cur" --event 'Order Completed' --fields properties --json

			# How many events
			$ rudder-cli dev events list --url "$url" --since "$cur" --json | wc -l
		`),
		Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			defer func() { track("dev events list", err) }()
			base, e := opts.listenerURL(cmd)
			if e == nil {
				e = opts.validate(cmd)
			}
			if e == nil {
				e = checkListFlags(cmd, limit, view, fields)
			}
			if e != nil {
				e.next = withURL(e.next, base)
				return fail(cmd, e)
			}
			q := opts.query(cmd.Flags())
			if cmd.Flags().Changed("limit") {
				q.Set("limit", strconv.Itoa(limit))
			}
			// list is a table of the whole events; full is the server default.
			if view == "compact" {
				q.Set("view", "compact")
			}
			if len(fields) > 0 {
				q["fields"] = fields
			}
			timeout := clientMargin
			if opts.timeout > 0 {
				timeout = opts.timeout
			}
			return readStream(cmd, &opts, base, q, timeout, view, fields)
		},
	}
	f := cmd.Flags()
	addReadFlags(f, &opts)
	f.IntVar(&limit, "limit", 100, "Print at most this many events, 1 to 1000; a request is never split")
	f.StringVar(&view, "view", "list", "list (a table of the whole events), compact (without the auto-collected context) or full")
	f.StringArrayVar(&fields, "fields", nil, "Keep only this dotted path of each event, such as properties; repeat for more")
	return cmd
}

func checkListFlags(cmd *cobra.Command, limit int, view string, fields []string) *usageError {
	help := cmd.CommandPath() + " --help"
	switch {
	case view == "counts":
		return &usageError{message: "--view counts is the summary, which dev events prints", next: "rudder-cli dev events --json"}
	case !slices.Contains([]string{"list", "compact", "full"}, view):
		return &usageError{message: fmt.Sprintf("--view must be list, compact or full, got %q", view), next: help}
	case len(fields) > 0 && cmd.Flags().Changed("view"):
		return &usageError{
			message: "--fields replaces --view; give one of them",
			next:    cmd.CommandPath() + " --fields " + strings.Join(shellWords(fields), " --fields "),
		}
	case limit < 1 || limit > devlisten.MaxLimit:
		return &usageError{message: fmt.Sprintf("--limit must be 1 to %d, got %d", devlisten.MaxLimit, limit), next: help}
	}
	for _, path := range fields {
		if strings.Contains(path, ",") {
			parts := strings.Split(path, ",")
			return &usageError{
				message: "--fields: repeat the flag for each path, as --fields " + strings.Join(parts, " --fields "),
				next:    cmd.CommandPath() + " --fields " + strings.Join(shellWords(parts), " --fields "),
			}
		}
		if err := devlisten.CheckFieldPath(path); err != nil {
			return &usageError{message: "--fields: " + err.Error(), next: help}
		}
	}
	return nil
}

func shellWords(values []string) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = shellWord(v)
	}
	return out
}

// withURL adds the listener URL to a dev events command, so a next runs
// against the listener the caller read. The server writes next without it.
func withURL(next, base string) string {
	if base == "" {
		return next
	}
	for _, path := range []string{"rudder-cli dev events list", "rudder-cli dev events"} {
		if rest, ok := strings.CutPrefix(next, path); ok && (rest == "" || rest[0] == ' ') {
			return path + " --url " + shellWord(base) + rest
		}
	}
	return next
}

// knownURL is the URL a next command names before the flags parsed.
func knownURL() string {
	return cmp.Or(os.Getenv(urlEnv), placeholderURL)
}
