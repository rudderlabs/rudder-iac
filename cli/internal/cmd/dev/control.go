package dev

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

const stopPollInterval = 200 * time.Millisecond

func newCmdInfo(deps Deps) *cobra.Command {
	var o clientFlags
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show the running server (GET /_dev/v1/info)",
		Long: "Show the server identity, cursor and store counts. Compare url with the URL the app sends to.\n" +
			"Read cursor here before an action, then pass it as --since.",
		Example: "  rudder-cli dev info --json\n" +
			"  rudder-cli dev info --json --jq .cursor",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := output{stdout: cmd.OutOrStdout(), stderr: cmd.ErrOrStderr(), flags: o}
			if err := o.check(cmd, nil); err != nil {
				return out.fail(err)
			}
			client, err := resolve(cmd.Context(), deps, o, 0)
			if err != nil {
				return out.fail(err)
			}
			info, err := client.Info(cmd.Context())
			if err != nil {
				return out.fail(err)
			}
			return out.result(cmd.Context(), info.Raw, "", func(w io.Writer) {
				fmt.Fprintf(w, "url:       %s\nserverId:  %s\ncursor:    %d\nrequests:  %d\nevents:    %d\ncontrol:   %d\n",
					info.URL, info.ServerID, info.Cursor, info.Store.Requests, info.Store.Events, info.Store.Control)
			})
		},
	}
	o.register(cmd.Flags(), false)
	return cmd
}

func newCmdReset(deps Deps) *cobra.Command {
	var o clientFlags
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Clear the captured requests (POST /_dev/v1/reset)",
		Long: "Remove every captured request. seq keeps counting, so an old cursor stays valid.\n" +
			"Prefer a cursor from dev info over a reset when other callers share the server.",
		Example: "  rudder-cli dev reset --json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := output{stdout: cmd.OutOrStdout(), stderr: cmd.ErrOrStderr(), flags: o}
			if err := o.check(cmd, nil); err != nil {
				return out.fail(err)
			}
			client, err := resolve(cmd.Context(), deps, o, 0)
			if err != nil {
				return out.fail(err)
			}
			res, err := client.Reset(cmd.Context())
			if err != nil {
				return out.fail(err)
			}
			return out.result(cmd.Context(), res.Raw, "", func(w io.Writer) {
				fmt.Fprintf(w, "removed %d requests, %d control; cursor %d\n", res.Removed.Requests, res.Removed.Control,
					res.Cursor)
			})
		},
	}
	o.register(cmd.Flags(), false)
	return cmd
}

type stopResult struct {
	Stopped  bool   `json:"stopped"`
	ServerID string `json:"serverId,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

func newCmdStop(deps Deps) *cobra.Command {
	var o clientFlags
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the running server (POST /_dev/v1/shutdown)",
		Long: "Stop the server named by --url, RUDDERSTACK_DEV_URL or the state file, and wait until it is gone.\n" +
			"No running server is not an error. A server bound to a non-loopback address refuses it.",
		Example: "  rudder-cli dev stop\n" +
			"  rudder-cli dev stop --url http://127.0.0.1:25171 --json",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStop(cmd, deps, o)
		},
	}
	o.register(cmd.Flags(), false)
	cmd.Flags().Lookup("timeout").Usage = "Wait up to `DURATION` for the server to stop; default 10s"
	return cmd
}

func runStop(cmd *cobra.Command, deps Deps, o clientFlags) error {
	out := output{stdout: cmd.OutOrStdout(), stderr: cmd.ErrOrStderr(), flags: o}
	if err := o.check(cmd, nil); err != nil {
		return out.fail(err)
	}
	timeout := o.timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()

	res, err := stopServer(ctx, deps, o)
	if err != nil {
		return out.fail(err)
	}
	raw, _ := json.Marshal(res)
	return out.result(cmd.Context(), raw, "", func(w io.Writer) {
		if res.Stopped {
			fmt.Fprintf(w, "stopped %s\n", res.ServerID)
			return
		}
		fmt.Fprintln(w, "no server is running")
	})
}

func stopServer(ctx context.Context, deps Deps, o clientFlags) (stopResult, error) {
	notRunning := stopResult{Reason: "not_running"}
	o.timeout = verifyTimeout
	client, err := resolve(ctx, deps, o, 0)
	var cliErr *cliError
	if errors.As(err, &cliErr) {
		if cliErr.Code == "stale_state" {
			_ = os.Remove(stateFilePath(deps.ConfigDir()))
		}
		return notRunning, nil
	}
	if err != nil {
		return stopResult{}, err
	}
	res, err := client.Shutdown(ctx)
	var apiErr *devlisten.APIError
	switch {
	case errors.As(err, &apiErr):
		return stopResult{}, err
	case err != nil:
		return notRunning, nil
	}
	return waitStopped(ctx, client.URL(), res.ServerID)
}

// waitStopped polls /info until the connection is refused or another
// server answers on the port.
func waitStopped(ctx context.Context, url, serverID string) (stopResult, error) {
	probe := devlisten.NewClient(url, devlisten.WithHTTPClient(&http.Client{Timeout: verifyTimeout}))
	ticker := time.NewTicker(stopPollInterval)
	defer ticker.Stop()
	for {
		if gone(ctx, probe, serverID) {
			return stopResult{Stopped: true, ServerID: serverID}, nil
		}
		select {
		case <-ctx.Done():
			return stopResult{}, &cliError{Code: "stop_timeout",
				Message: "server " + serverID + " at " + url + " is still running", Next: "rudder-cli dev listen --help"}
		case <-ticker.C:
		}
	}
}

// gone is true when the port refuses the connection or another server
// answers. A 503 means the server is still draining.
func gone(ctx context.Context, probe *devlisten.Client, serverID string) bool {
	info, err := probe.Info(ctx)
	if ctx.Err() != nil {
		return false
	}
	var apiErr *devlisten.APIError
	if err != nil {
		return !errors.As(err, &apiErr)
	}
	return info.ServerID != serverID
}

type sendOptions struct {
	clientFlags
	event    string
	typ      string
	userID   string
	writeKey string
}

func newCmdSend(deps Deps) *cobra.Command {
	var o sendOptions
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send one probe event through the ingestion path",
		Long: "Post one probe event like an SDK would, then report the status and the seq it was captured under.\n" +
			"If the probe arrives but the app's events do not, the app configuration is the problem.",
		Example: "  rudder-cli dev send --json\n" +
			"  rudder-cli dev send --write-key MY_KEY --json",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSend(cmd, deps, o)
		},
	}
	f := cmd.Flags()
	o.register(f, false)
	f.StringVar(&o.event, "event", "probe", "Event `NAME` of the probe")
	f.StringVar(&o.typ, "type", "track", "Event `TYPE`; it picks the route /v1/TYPE")
	f.StringVar(&o.userID, "user-id", "dev-send", "userId `ID` of the probe")
	f.StringVar(&o.writeKey, "write-key", devlisten.DefaultWriteKey, "Write `KEY` for Basic auth")
	return cmd
}

func runSend(cmd *cobra.Command, deps Deps, o sendOptions) error {
	out := output{stdout: cmd.OutOrStdout(), stderr: cmd.ErrOrStderr(), flags: o.clientFlags}
	if err := o.check(cmd, nil); err != nil {
		return out.fail(err)
	}
	client, err := resolve(cmd.Context(), deps, o.clientFlags, 0)
	if err != nil {
		return out.fail(err)
	}
	probe := devlisten.Probe{Event: o.event, Type: o.typ, UserID: o.userID, WriteKey: o.writeKey,
		UserAgent: "rudder-cli dev send/" + cmd.Root().Version}
	res, err := client.Send(cmd.Context(), probe)
	if err != nil {
		return out.fail(err)
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return out.fail(&cliError{Code: "send_failed", Message: fmt.Sprintf("%d %s", res.StatusCode, res.Body),
			Next: fmt.Sprintf("rudder-cli dev requests show %d --json", res.Seq)})
	}
	raw, _ := json.Marshal(res)
	return out.result(cmd.Context(), raw, "", func(w io.Writer) {
		fmt.Fprintf(w, "statusCode=%d body=%s seq=%d route=%s\n", res.StatusCode, res.Body, res.Seq, res.Route)
	})
}
