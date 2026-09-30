package dev

import (
	"fmt"
	"io"
	"maps"
	"slices"

	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

type summaryOptions struct {
	clientFlags
	since  uint64
	expect []string
}

func newCmdSummary(deps Deps) *cobra.Command {
	var o summaryOptions
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Count captured requests and diagnose them (GET /_dev/v1/summary)",
		Long: "Count the requests and events captured after a cursor and diagnose what went wrong.\n" +
			"It is the cheapest call. Each diagnosis carries next, the command to run next.\n\n" +
			"all_accepted means every received request was accepted; it says nothing about events that\n" +
			"never arrived. To prove presence and count in one call, map each user action to its event\n" +
			"and pass --expect NAME=COUNT (=0 asserts absence). expected then lists want, got and status\n" +
			"(present, missing, count_mismatch). A plain --expect NAME only checks presence: expected\n" +
			"still reports got, and note names the NAME=COUNT to pass. expected_missing names the\n" +
			"missing events.\n" +
			"bySource splits traffic by channel and by SDK family (browser, node, go, mobile, other), and\n" +
			"no_browser_traffic flags server events with no browser request.",
		Example: "  rudder-cli dev summary --since 0 --json\n" +
			"  rudder-cli dev summary --since 41 --expect 'Order Completed=1' --expect 'Suggestion Sent=1' --json\n" +
			"  rudder-cli dev summary --since 0 --json --jq '.diagnosis[].next'",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSummary(cmd, deps, o)
		},
	}
	o.register(cmd.Flags(), true)
	cmd.Flags().Uint64Var(&o.since, "since", 0, "Filter by cursor: requests with seq above this")
	cmd.Flags().StringArrayVar(&o.expect, "expect", nil,
		"Check that event `NAME=COUNT` arrived COUNT times (NAME alone: at least once); repeat for more")
	return cmd
}

func runSummary(cmd *cobra.Command, deps Deps, o summaryOptions) error {
	out := output{stdout: cmd.OutOrStdout(), stderr: cmd.ErrOrStderr(), flags: o.clientFlags}
	if err := o.check(cmd, nil); err != nil {
		return out.fail(err)
	}
	client, err := resolve(cmd.Context(), deps, o.clientFlags, 0)
	if err != nil {
		return out.fail(err)
	}
	s, err := client.Summary(cmd.Context(), devlisten.SummaryQuery{Since: o.since, Expect: o.expect})
	if err != nil {
		return out.fail(err)
	}
	return out.result(cmd.Context(), s.Raw, "", func(w io.Writer) { printSummary(w, s) })
}

func printSummary(w io.Writer, s devlisten.Summary) {
	fmt.Fprintf(w, "since %d, cursor %d\n", s.Since, s.Cursor)
	fmt.Fprintf(w, "requests: %d (%d failed, %d probes)\n", s.Requests.Total, s.Requests.Failed, s.Requests.Probes)
	fmt.Fprintf(w, "events:   %d\n", s.Events.Total)
	for _, name := range slices.Sorted(maps.Keys(s.Events.ByEvent)) {
		fmt.Fprintf(w, "  %-40s %d\n", clean(name, 40), s.Events.ByEvent[name])
	}
	fmt.Fprintf(w, "control:  %d (%d sourceConfig, %d preflight)\n", s.Control.Total, s.Control.SourceConfig,
		s.Control.Preflight)
	for _, family := range slices.Sorted(maps.Keys(s.BySource.BySdk)) {
		c := s.BySource.BySdk[family]
		fmt.Fprintf(w, "  sdk %-8s %d requests, %d events, %d control\n", family, c.Requests, c.Events, c.Control)
	}
	for _, e := range s.Expected {
		fmt.Fprintf(w, "expect %-40s got %d: %s\n", clean(e.Event, 40), e.Got, e.Status)
		if e.Note != "" {
			fmt.Fprintf(w, "  note: %s\n", clean(e.Note, 0))
		}
	}
	for _, d := range s.Diagnosis {
		fmt.Fprintf(w, "\n%s (%d): %s\nNext: %s\n", d.Code, d.Count, d.Message, d.Next)
	}
}
