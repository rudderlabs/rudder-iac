package dev

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

// eventFilters are the flags dev events and dev events list share.
type eventFilters struct {
	clientFlags
	since      string
	event      []string
	typ        []string
	statusCode []string
	writeKey   []string
	userID     string
	anonID     string
	wait       time.Duration
	min        int
}

func (o *eventFilters) register(f *pflag.FlagSet) {
	o.clientFlags.register(f)
	f.StringVar(&o.since, "since", "0", "Filter by cursor (requests after seq N), a duration back from now (5m) or an RFC 3339 time")
	f.StringArrayVar(&o.event, "event", nil, "Filter by event name; repeat to match any of several")
	f.StringArrayVar(&o.typ, "type", nil, "Filter by event type, such as track; repeatable")
	f.StringArrayVar(&o.statusCode, "status-code", nil, "Filter by HTTP status code; repeatable")
	f.StringArrayVar(&o.writeKey, "write-key", nil, "Filter by the write key a request was sent with; repeatable")
	f.StringVar(&o.userID, "user-id", "", "Filter by userId")
	f.StringVar(&o.anonID, "anonymous-id", "", "Filter by anonymousId")
	f.DurationVar(&o.wait, "wait", 0, "Wait up to this long for --min matching events, at most 110s")
	f.IntVar(&o.min, "min", 1, "Return once this many matching events exist")
}

type eventsOptions struct {
	eventFilters
	limit    int
	view     string
	fields   []string
	maxBytes int
}

func newCmdEvents(deps Deps) *cobra.Command {
	var o eventFilters
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Count and diagnose captured events",
		Long: "Print the summary of the captured events: counts by event name, type and write key,\n" +
			"control requests, SDK families and a diagnosis with the next command to run.",
		Example: "  rudder-cli dev events --url http://127.0.0.1:4321\n" +
			"  rudder-cli dev events --url http://127.0.0.1:4321 --event 'Order Completed' --json",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			defer func() { deps.Track("dev events", err, telemetry.KV{K: "json", V: o.json}) }()
			opts := eventsOptions{eventFilters: o, view: string(devlisten.ViewCounts)}
			return runEvents(cmd, deps, opts, true)
		},
	}
	o.register(cmd.Flags())
	cmd.AddCommand(newCmdEventsList(deps))
	return cmd
}

func newCmdEventsList(deps Deps) *cobra.Command {
	var o eventsOptions
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List captured events with their summary",
		Long:  "List the captured events, one line each by default, with the summary block.",
		Example: "  rudder-cli dev events list --url http://127.0.0.1:4321\n" +
			"  rudder-cli dev events list --url http://127.0.0.1:4321 --event 'Order Completed' --fields properties --json",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			defer func() {
				deps.Track("dev events list", err, telemetry.KV{K: "view", V: o.view}, telemetry.KV{K: "json", V: o.json})
			}()
			return runEvents(cmd, deps, o, cmd.Flags().Changed("view"))
		},
	}
	f := cmd.Flags()
	o.eventFilters.register(f)
	f.IntVar(&o.limit, "limit", 100, "Output at most this many events, at a request boundary")
	f.StringVar(&o.view, "view", "list", "Output view: list, compact or full")
	f.StringArrayVar(&o.fields, "fields", nil, "Output only this dotted path of each event, such as properties; repeatable")
	f.IntVar(&o.maxBytes, "max-bytes", 24000, "Output at most this many bytes per page; 0 turns the cap off")
	return cmd
}

// query builds the request. sendView is true when the view must reach the
// wire: the bare dev events always sends counts.
func (o eventsOptions) query(f *pflag.FlagSet, sendView bool) (devlisten.Query, error) {
	codes, err := parseCodes(o.statusCode)
	if err != nil {
		return devlisten.Query{}, err
	}
	q := devlisten.Query{
		Event: o.event, Type: o.typ, StatusCode: codes, WriteKey: o.writeKey, UserID: o.userID,
		AnonymousID: o.anonID, Fields: o.fields, Wait: o.wait,
	}
	// A flag reaches the wire only when set, so the server default applies
	// otherwise and --fields does not collide with a default --view.
	whenSet(f, map[string]func(){
		"since":     func() { q.Since, q.SinceWindow = splitSince(o.since) },
		"limit":     func() { q.Limit = o.limit },
		"max-bytes": func() { q.MaxBytes = maxBytes(o.maxBytes) },
		"min":       func() { q.Min = o.min },
	})
	if sendView {
		q.View = devlisten.View(o.view)
	}
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

var statusCodeFlag = regexp.MustCompile(`^([1-5][0-9][0-9]|[1-5]xx)$`)

// parseCodes checks --status-code values: a code such as 401 or a class
// such as 4xx.
func parseCodes(raw []string) ([]string, error) {
	for _, r := range raw {
		if !statusCodeFlag.MatchString(r) {
			return nil, usageError("rudder-cli dev events list --help",
				"--status-code %q is not a status code such as 401 or a class such as 4xx", r)
		}
	}
	return raw, nil
}

func runEvents(cmd *cobra.Command, deps Deps, o eventsOptions, sendView bool) error {
	out := newOutput(cmd.OutOrStdout(), cmd.ErrOrStderr(), o.json)
	q, err := o.query(cmd.Flags(), sendView)
	if err != nil {
		return out.fail(err)
	}
	client, err := resolve(cmd, nil, deps, o.clientFlags, o.wait)
	if err != nil {
		return out.fail(err)
	}
	page, err := client.Events(cmd.Context(), q)
	if err != nil {
		return out.fail(err)
	}
	human := func(w io.Writer) { printEventList(w, page, o.fields) }
	if o.view == string(devlisten.ViewCounts) {
		human = func(w io.Writer) { printSummary(w, page) }
	}
	if err := out.page(page.Raw, len(page.Events), page.Truncated, human); err != nil {
		return err
	}
	if page.HasMore {
		out.hint("More events. Next: rudder-cli dev events list --since %d", page.Cursor)
	}
	return nil
}

// printSummary is the table form of dev events: counts and diagnosis.
func printSummary(w io.Writer, page devlisten.Page) {
	s := page.Summary
	fmt.Fprintf(w, "since %d, cursor %d\n", page.Since, page.Cursor)
	fmt.Fprintf(w, "requests: %d (%d failed)   events: %d   control: %d\n", s.Requests.Total, s.Requests.Failed,
		s.Events.Total, s.Control.Total)
	for _, name := range slices.Sorted(maps.Keys(s.ByEvent)) {
		fmt.Fprintf(w, "  %-40s %d\n", clean(name, 40), s.ByEvent[name])
	}
	for _, key := range slices.Sorted(maps.Keys(s.ByWriteKey)) {
		c := s.ByWriteKey[key]
		fmt.Fprintf(w, "  key %-20s %d requests, %d events\n", clean(keyName(key), 20), c.Requests, c.Events)
	}
	for _, d := range s.Diagnosis {
		fmt.Fprintf(w, "%s (%d): %s\n  Next: %s\n", d.Code, d.Count, d.Message, d.Next)
	}
	if page.TimedOut {
		fmt.Fprintln(w, "timedOut: the wait ended before --min events arrived")
	}
}

// printEventList is the table form of dev events list: one header line,
// then one row per event. With --fields the chosen paths are the columns.
func printEventList(w io.Writer, page devlisten.Page, fields []string) {
	fmt.Fprintf(w, "%d of %d matching events, cursor %d", page.Returned, page.Total, page.Cursor)
	if page.HasMore {
		fmt.Fprint(w, ", more after it")
	}
	fmt.Fprintln(w)
	if len(page.Events) == 0 {
		return
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	defer func() { _ = tw.Flush() }()
	perSeq := map[uint64]int{}
	for _, ev := range page.Events {
		perSeq[ev.Seq]++
	}
	if len(fields) > 0 {
		fmt.Fprintln(tw, "SEQ\tTYPE\tEVENT\t"+strings.Join(fields, "\t"))
		for _, ev := range page.Events {
			fmt.Fprintf(tw, "%s\t%s\t%s", seqLabel(ev, perSeq), dash(clean(ev.Type, 16)), dash(clean(eventLabel(ev), 48)))
			for _, path := range fields {
				fmt.Fprint(tw, "\t"+clean(pathValue(ev.Raw, path), 60))
			}
			fmt.Fprintln(tw)
		}
		return
	}
	fmt.Fprintln(tw, "SEQ\tTIME\tTYPE\tEVENT\tWRITE KEY\tSTATUS")
	for _, ev := range page.Events {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%d\n", seqLabel(ev, perSeq), clock(ev.ReceivedAt),
			dash(clean(ev.Type, 16)), dash(clean(eventLabel(ev), 64)), clean(keyName(ev.WriteKey), 24), ev.StatusCode)
	}
}

// seqLabel is the request seq, with /idx when the request carried more than
// one of the listed events.
func seqLabel(ev devlisten.Event, perSeq map[uint64]int) string {
	if perSeq[ev.Seq] > 1 {
		return fmt.Sprintf("%d/%d", ev.Seq, ev.Idx)
	}
	return strconv.FormatUint(ev.Seq, 10)
}

func eventLabel(ev devlisten.Event) string {
	if ev.Name != "" {
		return ev.Name
	}
	return ev.Label
}

func keyName(key string) string {
	if key == "" {
		return "(none)"
	}
	return key
}

// pathValue is the JSON of one dotted path of an event item, "-" when absent.
func pathValue(raw json.RawMessage, path string) string {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return "-"
	}
	path = strings.TrimPrefix(path, "message.")
	if m, ok := v.(map[string]any); ok && strings.HasPrefix(path, "context") {
		v = m["message"]
	}
	for _, part := range strings.Split(path, ".") {
		m, ok := v.(map[string]any)
		if !ok {
			return "-"
		}
		v = m[part]
	}
	b, _ := json.Marshal(v)
	return string(b)
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

// splitSince sends a cursor as a number and anything else as a window for
// the server to parse, so the CLI and HTTP accept the same forms.
func splitSince(raw string) (uint64, string) {
	if n, err := strconv.ParseUint(raw, 10, 64); err == nil {
		return n, ""
	}
	return 0, raw
}
