package dev

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

func newCmdRequests(deps Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "requests",
		Short: "Inspect captured requests",
		Long: "Inspect whole captured requests: headers, body, response and rejection.\n\n" +
			"A request is the unit of failure; use requests list --failed to find rejected ones.",
		Example: "  rudder-cli dev requests list --failed --json\n" +
			"  rudder-cli dev requests show 42 --json",
	}
	cmd.AddCommand(newCmdRequestsList(deps))
	cmd.AddCommand(newCmdRequestsShow(deps))
	return cmd
}

type requestsListOptions struct {
	clientFlags
	since      uint64
	limit      int
	order      string
	kind       string
	route      []string
	statusCode []string
	failed     bool
	stage      string
	fields     []string
	maxBytes   int
	wait       time.Duration
	min        int
}

func newCmdRequestsList(deps Deps) *cobra.Command {
	var o requestsListOptions
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List captured requests (GET /_dev/v1/requests)",
		Long: "List captured requests after a cursor, without bodies. Each flag is the query parameter of\n" +
			"the same name. --kind control shows /sourceConfig, preflights and unknown paths.\n" +
			"omitted.next names the rudder-cli dev requests show call for one whole request.",
		Example: "  rudder-cli dev requests list --failed --stage auth --json\n" +
			"  rudder-cli dev requests list --kind control --json",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRequestsList(cmd, deps, o)
		},
	}
	f := cmd.Flags()
	o.register(f, true)
	f.Uint64Var(&o.since, "since", 0, "Filter by cursor: requests with seq above this")
	f.IntVar(&o.limit, "limit", 100, "Output at most this many requests")
	f.StringVar(&o.order, "order", "asc", "Output order; asc only in this phase")
	f.StringVar(&o.kind, "kind", "ingestion", "Filter by kind: ingestion, control or all")
	f.StringArrayVar(&o.route, "route", nil, "Filter by route, such as /v1/track; repeatable")
	f.StringArrayVar(&o.statusCode, "status-code", nil, "Filter by HTTP status code; repeatable")
	f.BoolVar(&o.failed, "failed", false, "Filter by failure; --failed=false selects accepted requests")
	f.StringVar(&o.stage, "stage", "", "Filter by rejection stage, such as auth or body")
	f.StringArrayVar(&o.fields, "fields", nil, "Output only this dotted `PATH`, such as request.headers; repeatable")
	f.IntVar(&o.maxBytes, "max-bytes", 24000, "Output at most this many bytes per page; 0 turns the cap off")
	f.DurationVar(&o.wait, "wait", 0, "Wait up to `DURATION` for --min matches (at most 110s)")
	f.IntVar(&o.min, "min", 1, "Return once this many matches exist")
	return cmd
}

func runRequestsList(cmd *cobra.Command, deps Deps, o requestsListOptions) error {
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
	page, err := client.Requests(cmd.Context(), q)
	if err != nil {
		return out.fail(err)
	}
	err = out.page(cmd.Context(), page.Raw, len(page.Requests), page.TimedOut, page.Truncated,
		func(w io.Writer) { printRequestsTable(w, page) })
	if err == nil && len(page.Requests) == 0 {
		out.hint("No matching requests. Next: rudder-cli dev summary --since %d", page.Since)
	}
	return err
}

func (o requestsListOptions) query(f *pflag.FlagSet) (devlisten.RequestQuery, error) {
	codes, err := parseCodes(o.statusCode)
	if err != nil {
		return devlisten.RequestQuery{}, err
	}
	q := devlisten.RequestQuery{Route: o.route, StatusCode: codes, Stage: o.stage, Fields: o.fields, Wait: o.wait}
	whenSet(f, map[string]func(){
		"since":     func() { q.Since = o.since },
		"limit":     func() { q.Limit = o.limit },
		"order":     func() { q.Order = devlisten.Order(o.order) },
		"kind":      func() { q.Kind = o.kind },
		"failed":    func() { q.Failed = &o.failed },
		"max-bytes": func() { q.MaxBytes = maxBytes(o.maxBytes) },
		"min":       func() { q.Min = o.min },
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
	fields   []string
	maxBytes int
}

func newCmdRequestsShow(deps Deps) *cobra.Command {
	var o requestsShowOptions
	cmd := &cobra.Command{
		Use:   "show SEQ",
		Short: "Show one whole captured request (GET /_dev/v1/requests/{seq})",
		Long: "Show one captured request with headers, body, response and every event, enriched.\n" +
			"Credential headers are redacted and listed in request.redactedHeaders.",
		Example: "  rudder-cli dev requests show 42 --json\n" +
			"  rudder-cli dev requests show 42 --fields request.headers --fields response --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRequestsShow(cmd, deps, o, args)
		},
	}
	f := cmd.Flags()
	o.register(f, true)
	f.StringArrayVar(&o.fields, "fields", nil, "Output only this dotted `PATH`, such as request.headers; repeatable")
	f.IntVar(&o.maxBytes, "max-bytes", 24000, "Output at most this many bytes; 0 turns the cap off")
	return cmd
}

func runRequestsShow(cmd *cobra.Command, deps Deps, o requestsShowOptions, args []string) error {
	out := output{stdout: cmd.OutOrStdout(), stderr: cmd.ErrOrStderr(), flags: o.clientFlags}
	if err := o.check(cmd, args); err != nil {
		return out.fail(err)
	}
	seq, err := parseSeq(args)
	if err != nil {
		return out.fail(err)
	}
	q := devlisten.RecordQuery{Fields: o.fields}
	if cmd.Flags().Changed("max-bytes") {
		q.MaxBytes = maxBytes(o.maxBytes)
	}
	client, err := resolve(cmd.Context(), deps, o.clientFlags, 0)
	if err != nil {
		return out.fail(err)
	}
	rec, err := client.Request(cmd.Context(), seq, q)
	if err != nil {
		return out.fail(err)
	}
	human := func(w io.Writer) { printIndented(w, rec.Raw) }
	if rec.Truncated != nil {
		return out.outputLimit(cmd.Context(), rec.Raw, rec.Truncated, human)
	}
	return out.result(cmd.Context(), rec.Raw, "", human)
}

func parseSeq(args []string) (uint64, error) {
	if len(args) != 1 {
		return 0, usageError("rudder-cli dev requests list --json",
			"requests show takes one SEQ; list the requests to find it")
	}
	seq, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return 0, usageError("rudder-cli dev requests list --json", "SEQ %q is not a request seq", args[0])
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
