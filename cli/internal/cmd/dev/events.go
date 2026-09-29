package dev

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

// clientSlack is added to --wait for the HTTP deadline, so the server's
// timedOut answer arrives before the client gives up.
const clientSlack = 5 * time.Second

type eventsOptions struct {
	url      string
	serverID string
	query    devlisten.Query
	json     bool
}

func newCmdEvents() *cobra.Command {
	var opts eventsOptions
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Print the captured events as the /_dev/v1/events JSON envelope",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEvents(cmd, opts)
		},
	}
	f := cmd.Flags()
	f.StringVar(&opts.url, "url", "", "server URL from the ready line (required)")
	f.StringVar(&opts.serverID, "server-id", "", "serverId from the ready line; a restarted server then answers server_changed")
	f.Uint64Var(&opts.query.Since, "since", 0, "return requests with seq greater than this cursor")
	f.StringArrayVar(&opts.query.Event, "event", nil, "event name to match; repeat to match any of several")
	f.StringArrayVar(&opts.query.Type, "type", nil, "event type to match; repeat to match any of several")
	f.IntVar(&opts.query.Limit, "limit", 0, "maximum events per page (server default 100)")
	f.IntVar(&opts.query.Min, "min", 0, "return once this many matches exist (server default 1)")
	f.DurationVar(&opts.query.Wait, "wait", 0, "long-poll up to this long for --min matches (at most 110s)")
	f.BoolVar(&opts.json, "json", false, "print the JSON envelope (the only output mode in this build)")
	_ = cmd.MarkFlagRequired("url")
	return cmd
}

func runEvents(cmd *cobra.Command, opts eventsOptions) error {
	clientOpts := []devlisten.ClientOption{
		devlisten.WithHTTPClient(&http.Client{
			Timeout:   opts.query.Wait + clientSlack,
			Transport: http.DefaultTransport.(*http.Transport).Clone(),
		}),
	}
	if opts.serverID != "" {
		clientOpts = append(clientOpts, devlisten.WithServerID(opts.serverID))
	}

	page, err := devlisten.NewClient(opts.url, clientOpts...).Events(cmd.Context(), opts.query)
	if err != nil {
		return printError(cmd.OutOrStdout(), err)
	}
	return printJSON(cmd.OutOrStdout(), page)
}

// printError prints the section 4.8 error object on stdout and nothing else,
// so a JSON reader always gets JSON.
func printError(w io.Writer, err error) error {
	apiErr := &devlisten.APIError{Code: "server_unreachable", Message: err.Error()}
	errors.As(err, &apiErr)
	if printErr := printJSON(w, map[string]any{"error": apiErr}); printErr != nil {
		return printErr
	}
	return &cmderrors.SilentError{Err: err}
}

func printJSON(w io.Writer, v any) error {
	out, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("encoding output: %w", err)
	}
	_, err = fmt.Fprintln(w, string(out))
	return err
}
