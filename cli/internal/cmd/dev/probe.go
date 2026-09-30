package dev

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

type probeOptions struct {
	clientFlags
	browser bool
}

type probeResult struct {
	URL      string `json:"url"`
	ServerID string `json:"serverId"`
	Cursor   uint64 `json:"cursor"`
	Message  string `json:"message"`
	Next     string `json:"next"`
}

const probeMessage = "Open url in the app's browser, or with Playwright. The page loads @rudderstack/analytics-js " +
	devlisten.ProbeSDK + " from the listener and sends one track named '" + devlisten.ProbeEvent +
	"', captured with probe: true. A captured probe proves the browser-to-listener path; " +
	"if it arrives and the app's events do not, the app wiring is at fault (app checklist in rudder-cli dev listen --help)."

func newCmdProbe(deps Deps) *cobra.Command {
	var o probeOptions
	cmd := &cobra.Command{
		Use:   "probe --browser",
		Short: "Print a page that sends one event from a real browser SDK to the listener",
		Long: "Print the URL of a page that the listener serves. " + probeMessage + "\n\n" +
			"The page needs no network: the SDK bundle is embedded in rudder-cli. For the server path,\n" +
			"use rudder-cli dev send.",
		Example: "  rudder-cli dev probe --browser --json\n" +
			"  rudder-cli dev summary --since 0 --expect 'dev probe=1' --json",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProbe(cmd, deps, o)
		},
	}
	o.register(cmd.Flags(), false)
	cmd.Flags().BoolVar(&o.browser, "browser", false, "Print the browser probe page (the only probe so far)")
	return cmd
}

func runProbe(cmd *cobra.Command, deps Deps, o probeOptions) error {
	out := output{stdout: cmd.OutOrStdout(), stderr: cmd.ErrOrStderr(), flags: o.clientFlags}
	if err := o.check(cmd, nil); err != nil {
		return out.fail(err)
	}
	if !o.browser {
		return out.fail(usageError("rudder-cli dev probe --browser --json",
			"pass --browser; for a server-side probe use rudder-cli dev send"))
	}
	client, err := resolve(cmd.Context(), deps, o.clientFlags, 0)
	if err != nil {
		return out.fail(err)
	}
	info, err := client.Info(cmd.Context())
	if err != nil {
		return out.fail(err)
	}
	res := probeResult{URL: client.URL() + devlisten.ProbePagePath, ServerID: info.ServerID, Cursor: info.Cursor,
		Message: probeMessage,
		Next: fmt.Sprintf("rudder-cli dev summary --since %d --expect '%s=1' --json", info.Cursor,
			devlisten.ProbeEvent)}
	raw, _ := json.Marshal(res)
	return out.result(cmd.Context(), raw, "", func(w io.Writer) {
		fmt.Fprintf(w, "%s\n\n%s\nNext: %s\n", res.URL, res.Message, res.Next)
	})
}
