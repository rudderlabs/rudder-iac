package dev

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

type requestsOptions struct {
	clientFlags
	since      uint64
	limit      int
	kind       string
	statusCode []string
	writeKey   []string
	failed     bool
	view       string
	fields     []string
	maxBytes   int
}

func newCmdRequests(deps Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "requests",
		Short: "Inspect whole captured requests",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newCmdRequestsList(deps))
	cmd.AddCommand(newCmdRequestsShow(deps))
	return cmd
}

func newCmdRequestsList(deps Deps) *cobra.Command {
	var o requestsOptions
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List captured requests (GET /_dev/v1/requests)",
		Long: "List whole captured requests after a cursor: the unit of failure, and the only view of rejected\n" +
			"and control requests. Each flag is the query parameter of the same name.\n\n" +
			"Views: list (one short line), compact (the default: seq, receivedAt, method, route,\n" +
			"statusCode, outcome, kind, rejection and event names) and full (the whole record).\n" +
			"--fields replaces the view and always keeps seq and request.method.\n" +
			"--kind control shows /sourceConfig, preflights (method OPTIONS) and unknown paths.\n" +
			"--write-key filters by the key a request was sent with; on dev listen it is the allowlist.\n" +
			"omitted.next names the dev requests show call for one whole request.",
		Example: "  rudder-cli dev requests --url \"$url\" --failed --json\n" +
			"  rudder-cli dev requests --url \"$url\" --kind control --view list --json\n" +
			"  rudder-cli dev requests show 42 --url \"$url\" --fields request.body --json",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			defer func() {
				deps.Track("dev requests list", err, telemetry.KV{K: "view", V: o.view}, telemetry.KV{K: "json", V: o.json})
			}()
			return runRequests(cmd, deps, o)
		},
	}
	f := cmd.Flags()
	o.register(f)
	f.Uint64Var(&o.since, "since", 0, "Filter by cursor: requests with seq above this")
	f.IntVar(&o.limit, "limit", 100, "Output at most this many requests")
	f.StringVar(&o.kind, "kind", "ingestion", "Filter by kind: ingestion, control or all")
	f.StringArrayVar(&o.statusCode, "status-code", nil, "Filter by HTTP status `CODE`; repeatable")
	f.StringArrayVar(&o.writeKey, "write-key", nil, "Filter by the write `KEY` a request was sent with; repeatable")
	f.BoolVar(&o.failed, "failed", false, "Filter by failure; --failed=false selects accepted requests")
	f.StringVar(&o.view, "view", "list", "Output view: list, compact or full")
	f.StringArrayVar(&o.fields, "fields", nil, "Output only this dotted `PATH`, such as request.headers; repeatable")
	f.IntVar(&o.maxBytes, "max-bytes", 24000, "Output at most this many bytes per page; 0 turns the cap off")
	return cmd
}

func runRequests(cmd *cobra.Command, deps Deps, o requestsOptions) error {
	out := newOutput(cmd.OutOrStdout(), cmd.ErrOrStderr(), o.json)
	q, err := o.query(cmd.Flags())
	if err != nil {
		return out.fail(err)
	}
	client, err := resolve(cmd, nil, deps, o.clientFlags, 0)
	if err != nil {
		return out.fail(err)
	}
	page, err := client.Requests(cmd.Context(), q)
	if err != nil {
		return out.fail(err)
	}
	err = out.page(page.Raw, len(page.Requests), page.Truncated, func(w io.Writer) { printRequestsTable(w, page) })
	if err == nil && len(page.Requests) == 0 {
		out.hint("No matching requests. Next: rudder-cli dev events list --since %d", page.Since)
	}
	return err
}

func (o requestsOptions) query(f *pflag.FlagSet) (devlisten.RequestQuery, error) {
	codes, err := parseCodes(o.statusCode)
	if err != nil {
		return devlisten.RequestQuery{}, err
	}
	q := devlisten.RequestQuery{StatusCode: codes, WriteKey: o.writeKey, Fields: o.fields}
	whenSet(f, map[string]func(){
		"since":     func() { q.Since = o.since },
		"limit":     func() { q.Limit = o.limit },
		"kind":      func() { q.Kind = o.kind },
		"view":      func() { q.View = devlisten.View(o.view) },
		"failed":    func() { q.Failed = &o.failed },
		"max-bytes": func() { q.MaxBytes = maxBytes(o.maxBytes) },
	})
	return q, nil
}

func printRequestsTable(w io.Writer, page devlisten.RequestPage) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "SEQ\tTIME\tST\tKIND\tROUTE\tOUTCOME\tSTAGE\tEVENTS")
	for _, rec := range page.Requests {
		stage := ""
		if rec.Rejection != nil {
			stage = rec.Rejection.Stage
		}
		fmt.Fprintf(tw, "%d\t%s\t%d\t%s\t%s\t%s\t%s\t%d\n", rec.Seq, clock(rec.ReceivedAt), rec.StatusCode, rec.Kind,
			clean(rec.Route, 64), rec.Outcome, dash(stage), len(rec.Events))
	}
	_ = tw.Flush()
}

type requestsShowOptions struct {
	clientFlags
	view     string
	fields   []string
	maxBytes int
}

func newCmdRequestsShow(deps Deps) *cobra.Command {
	var o requestsShowOptions
	cmd := &cobra.Command{
		Use:   "show SEQ",
		Short: "Show one whole captured request (GET /_dev/v1/requests/{seq})",
		Long: "Show one captured request: headers, body, response, rejection and the scalars of each event.\n\n" +
			"Views: compact (the default) leaves out each event's message and enrichedMessage, because\n" +
			"request.body already holds them as sent; full restores both. --fields picks paths.\n" +
			"Credential headers are redacted and listed in request.redactedHeaders.",
		Example: "  rudder-cli dev requests show 42 --fields request.body --json\n" +
			"  rudder-cli dev requests show 42 --url \"$url\" --json\n" +
			"  rudder-cli dev requests show 42 --url \"$url\" --view full --json",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			defer func() {
				deps.Track("dev requests show", err, telemetry.KV{K: "view", V: o.view}, telemetry.KV{K: "json", V: o.json})
			}()
			return runRequestsShow(cmd, deps, o, args)
		},
	}
	f := cmd.Flags()
	o.register(f)
	f.StringVar(&o.view, "view", "compact", "Output view: compact or full")
	f.StringArrayVar(&o.fields, "fields", nil, "Output only this dotted `PATH`, such as request.body; repeatable")
	f.IntVar(&o.maxBytes, "max-bytes", 24000, "Output at most this many bytes; 0 turns the cap off")
	return cmd
}

func runRequestsShow(cmd *cobra.Command, deps Deps, o requestsShowOptions, args []string) error {
	out := newOutput(cmd.OutOrStdout(), cmd.ErrOrStderr(), o.json)
	seq, err := parseSeq(args)
	if err != nil {
		return out.fail(err)
	}
	q := o.query(cmd.Flags())
	client, err := resolve(cmd, args, deps, o.clientFlags, 0)
	if err != nil {
		return out.fail(err)
	}
	rec, err := client.Request(cmd.Context(), seq, q)
	if err != nil {
		return out.fail(err)
	}
	human := func(w io.Writer) { printIndented(w, rec.Raw) }
	if rec.Truncated != nil {
		return out.outputLimit(rec.Raw, rec.Truncated, human)
	}
	return out.result(rec.Raw, human)
}

func (o requestsShowOptions) query(f *pflag.FlagSet) devlisten.RecordQuery {
	q := devlisten.RecordQuery{Fields: o.fields}
	if f.Changed("view") {
		q.View = devlisten.View(o.view)
	}
	if f.Changed("max-bytes") {
		q.MaxBytes = maxBytes(o.maxBytes)
	}
	return q
}

func parseSeq(args []string) (uint64, error) {
	if len(args) != 1 {
		return 0, usageError("rudder-cli dev requests list --view list --json",
			"requests show takes one SEQ; list the requests to find it")
	}
	seq, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return 0, usageError("rudder-cli dev requests list --view list --json", "SEQ %q is not a request seq", args[0])
	}
	return seq, nil
}

func printIndented(w io.Writer, raw []byte) {
	var buf bytes.Buffer
	if json.Indent(&buf, raw, "", "  ") != nil {
		buf.Write(raw)
	}
	fmt.Fprintln(w, buf.String())
}
