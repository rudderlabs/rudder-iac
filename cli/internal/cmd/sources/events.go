package sources

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
	url          string
	allowPartial bool
	serverID     string
	timeout      time.Duration
	json         bool
	since        string
	events       []string
	types        []string
	userID       string
	anonymousID  string
	writeKeys    []string
}

func addReadFlags(f *pflag.FlagSet, o *readOptions) {
	f.StringVar(&o.url, "url", "", "URL of the listener, from its ready line (default $"+urlEnv+")")
	f.StringVar(&o.serverID, "server-id", "", "Answer 409 server_changed unless the listener has this serverId")
	f.BoolVarP(&o.json, "json", "j", false, "Print the listener's JSON instead of a table")
	f.BoolVar(&o.allowPartial, "allow-partial", false,
		"Exit 0 when the listener evicted requests of this window; the output is then incomplete")
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
		opts   readOptions
		limit  int
		view   string
		fields []string
	)
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Print stored events as the SDK sent them",
		Long: heredoc.Doc(`
			Print the accepted events a listener started with --store holds, after a cursor. With --json
			the output is NDJSON: one event per line, exactly as the SDK sent it, with nothing added.
			Without --json it is a table.
			--view compact drops the SDK's auto-collected context; --fields PATH keeps only that dotted
			path (repeat it; it cannot be combined with --view).

			The stream carries no cursor. Take it from the ready line. At most --limit events per page;
			when more are left, stderr names the --since to continue from. Events of rejected requests
			are not in the stream: read them on URL/_dev/v1/requests or the review page.

			The listener keeps the last 10,000 requests or 64 MiB. When the window lost requests to
			that limit, the command prints the events that are left and exits 1 with events_evicted on
			stderr, or exits 0 with --allow-partial.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli sources events --url http://127.0.0.1:4321 --since 5m

			# The properties of one event
			$ rudder-cli sources events --url "$url" --since "$cur" --event 'Order Completed' --fields properties --json

			# Check a property type (guard the empty stream)
			$ rudder-cli sources events --url "$url" --since "$cur" --event 'Order Completed' --fields properties --json | jq -e -s 'length > 0 and all(.[]; .properties.total | type == "number")'

			# How many events
			$ rudder-cli sources events --url "$url" --since "$cur" --json | wc -l
		`),
		Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			defer func() { track("sources events", err) }()
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
	f.DurationVar(&opts.timeout, "timeout", 0, "Give up on the listener after this long (default 5s)")
	f.IntVar(&limit, "limit", 100, "Print at most this many events, 1 to 1000; a request is never split")
	f.StringVar(&view, "view", "list", "list (a table of the whole events), compact (without the auto-collected context) or full")
	f.StringArrayVar(&fields, "fields", nil, "Keep only this dotted path of each event, such as properties; repeat for more")
	return cmd
}

func checkListFlags(cmd *cobra.Command, limit int, view string, fields []string) *usageError {
	help := cmd.CommandPath() + " --help"
	switch {
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

// withURL adds the listener URL to a sources events command, so a next runs
// against the listener the caller read. The server writes next without it.
func withURL(next, base string) string {
	if base == "" {
		return next
	}
	for _, path := range []string{"rudder-cli sources events", "rudder-cli sources events"} {
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
