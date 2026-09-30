package dev

import (
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

func newCmdEvents(deps Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Inspect captured events",
		Long: "Inspect the events inside captured requests.\n\n" +
			"Start with rudder-cli dev summary, then events list --view summary, then the default compact\n" +
			"view, then --fields or --view full, then rudder-cli dev requests show SEQ.",
		Example: "  rudder-cli dev events list --since 0 --view summary --json\n" +
			"  rudder-cli dev events list --since 41 --event 'Order Completed' --wait 30s --json",
	}
	cmd.AddCommand(newCmdEventsList(deps))
	return cmd
}

type eventsListOptions struct {
	clientFlags
	since      uint64
	limit      int
	order      string
	view       string
	fields     []string
	include    []string
	maxBytes   int
	event      []string
	typ        []string
	route      []string
	statusCode []string
	userID     string
	anonID     string
	wait       time.Duration
	min        int
}

func newCmdEventsList(deps Deps) *cobra.Command {
	var o eventsListOptions
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List captured events (GET /_dev/v1/events)",
		Long: "List the events captured after a cursor. Each flag is the query parameter of the same name.\n\n" +
			"--json prints the compact view capped at 24000 bytes; omitted.next and truncated.next name\n" +
			"the call that shows more. --since is exclusive: pass the cursor of the previous page.\n" +
			"--wait long-polls for --min matches, at most 110s; a timeout exits 0 with timedOut true.\n" +
			"Server lookup: --url, RUDDERSTACK_DEV_URL, then the state file of rudder-cli dev listen.",
		Example: "  rudder-cli dev events list --since 0 --view summary --json\n" +
			"  rudder-cli dev events list --since 41 --event 'Suggestion Sent' --include context --json\n" +
			"  rudder-cli dev events list --since 41 --fields properties --json --jq '.events[].properties'",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEventsList(cmd, deps, o)
		},
	}
	f := cmd.Flags()
	o.register(f, true)
	f.Uint64Var(&o.since, "since", 0, "Filter by cursor: events of requests with seq above this")
	f.IntVar(&o.limit, "limit", 100, "Output at most this many events, at a request boundary")
	f.StringVar(&o.order, "order", "asc", "Output order; asc only in this phase")
	f.StringVar(&o.view, "view", "compact", "Output view: summary, compact or full (requests list has the same three)")
	f.StringArrayVar(&o.fields, "fields", nil, "Output only this dotted `PATH`, such as properties; repeat for more")
	f.StringArrayVar(&o.include, "include", nil, "Output more in compact: context or enrichment; repeatable")
	f.IntVar(&o.maxBytes, "max-bytes", 24000, "Output at most this many bytes per page; 0 turns the cap off")
	f.StringArrayVar(&o.event, "event", nil, "Filter by event `NAME`; repeat to match any of several")
	f.StringArrayVar(&o.typ, "type", nil, "Filter by event `TYPE`; repeatable")
	f.StringArrayVar(&o.route, "route", nil, "Filter by route, such as /v1/track; repeatable")
	f.StringArrayVar(&o.statusCode, "status-code", nil, "Filter by HTTP status code; repeatable")
	f.StringVar(&o.userID, "user-id", "", "Filter by userId `ID`")
	f.StringVar(&o.anonID, "anonymous-id", "", "Filter by anonymousId `ID`")
	f.DurationVar(&o.wait, "wait", 0, "Wait up to `DURATION` for --min matches (at most 110s)")
	f.IntVar(&o.min, "min", 1, "Return once this many matches exist")
	return cmd
}

func (o eventsListOptions) query(f *pflag.FlagSet) (devlisten.Query, error) {
	codes, err := parseCodes(o.statusCode)
	if err != nil {
		return devlisten.Query{}, err
	}
	q := devlisten.Query{
		Event: o.event, Type: o.typ, Route: o.route, StatusCode: codes, UserID: o.userID, AnonymousID: o.anonID,
		Include: o.include, Fields: o.fields, Wait: o.wait,
	}
	// A flag reaches the wire only when set, so the server default applies
	// otherwise and --fields does not collide with a default --view.
	whenSet(f, map[string]func(){
		"since":     func() { q.Since = o.since },
		"limit":     func() { q.Limit = o.limit },
		"order":     func() { q.Order = devlisten.Order(o.order) },
		"view":      func() { q.View = devlisten.View(o.view) },
		"max-bytes": func() { q.MaxBytes = maxBytes(o.maxBytes) },
		"min":       func() { q.Min = o.min },
	})
	return q, nil
}

// whenSet runs each setter whose flag the caller set.
func whenSet(f *pflag.FlagSet, setters map[string]func()) {
	for name, set := range setters {
		if f.Changed(name) {
			set()
		}
	}
}

func maxBytes(n int) int {
	if n == 0 {
		return devlisten.MaxBytesOff
	}
	return n
}

func parseCodes(raw []string) ([]int, error) {
	codes := make([]int, 0, len(raw))
	for _, r := range raw {
		n, err := strconv.Atoi(r)
		if err != nil {
			return nil, usageError("rudder-cli dev events list --help", "--status-code %q is not an integer", r)
		}
		codes = append(codes, n)
	}
	return codes, nil
}

func runEventsList(cmd *cobra.Command, deps Deps, o eventsListOptions) error {
	out := output{stdout: cmd.OutOrStdout(), stderr: cmd.ErrOrStderr(), flags: o.clientFlags}
	if err := o.check(cmd, nil); err != nil {
		return out.fail(err)
	}
	q, err := o.query(cmd.Flags())
	if err != nil {
		return out.fail(err)
	}
	client, err := resolve(cmd.Context(), deps, o.clientFlags, o.wait)
	if err != nil {
		return out.fail(err)
	}
	page, err := client.Events(cmd.Context(), q)
	if err != nil {
		return out.fail(err)
	}
	err = out.page(cmd.Context(), page.Raw, len(page.Events), page.TimedOut, page.Truncated,
		func(w io.Writer) { printEventsTable(w, page) })
	if err == nil {
		eventsHints(out, page)
	}
	return err
}

func eventsHints(out output, page devlisten.Page) {
	if len(page.Events) == 0 {
		out.hint("No matching events. Next: rudder-cli dev summary --since %d", page.Since)
	}
	if page.HasMore {
		out.hint("More events. Next: rudder-cli dev events list --since %d", page.Cursor)
	}
}

func printEventsTable(w io.Writer, page devlisten.Page) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "SEQ\tTIME\tST\tTYPE\tEVENT\tUSER\tROUTE")
	for _, ev := range page.Events {
		fmt.Fprintf(tw, "%d.%d\t%s\t%d\t%s\t%s\t%s\t%s\n", ev.Seq, ev.Idx, clock(ev.ReceivedAt), ev.StatusCode,
			dash(clean(ev.Type, 16)), dash(clean(ev.Name, 64)), dash(clean(ev.UserID, 32)), ev.Route)
	}
	_ = tw.Flush()
}

func clock(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Local().Format("15:04:05")
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
