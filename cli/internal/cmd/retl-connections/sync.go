package retlconnection

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/spf13/cobra"
)

// pollInterval is how often a waiting sync asks for its own status. Runs are
// measured in minutes, so anything tighter only adds load.
const pollInterval = 5 * time.Second

func newCmdSync() *cobra.Command {
	var (
		syncType   string
		wait       bool
		timeout    time.Duration
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "sync <external-id>",
		Short: "Run a sync on a RETL connection",
		Long: heredoc.Doc(`
			Starts a sync and, unless --wait=false, follows it to a verdict.

			The exit code reports whether every row was delivered, not merely
			whether the run finished: a run that completes having dropped rows
			is reported as failed and exits non-zero.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli retl-connections sync customers-to-crm
			$ rudder-cli retl-connections sync customers-to-crm --type incremental
			$ rudder-cli retl-connections sync customers-to-crm --wait=false
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			defer func() {
				telemetry.TrackCommand("retl-connections sync", err, []telemetry.KV{
					{K: "syncType", V: syncType},
					{K: "wait", V: wait},
					{K: "json", V: jsonOutput},
				}...)
			}()

			d, err := app.NewDeps()
			if err != nil {
				return err
			}

			connectionID, err := resolveConnection(cmd.Context(), d, args[0])
			if err != nil {
				return err
			}

			store := retlClient.NewRudderRETLStore(d.Client())
			started, err := store.StartSync(cmd.Context(), connectionID, retlClient.SyncType(syncType))
			if err != nil {
				return err
			}

			if !wait {
				if jsonOutput {
					return printJSON(cmd, map[string]string{"syncId": started.SyncID})
				}
				fmt.Fprintf(cmd.OutOrStdout(), "sync %s started\n", started.SyncID)
				return nil
			}

			sync, err := awaitSync(cmd.Context(), store, connectionID, started.SyncID, timeout)
			if err != nil {
				return err
			}

			if jsonOutput {
				if err = printJSON(cmd, sync); err != nil {
					return err
				}
			} else {
				fmt.Fprint(cmd.OutOrStdout(), renderSync(*sync, time.Now()))
			}

			// The run finishing is not the same as the data arriving, so the
			// exit code follows the outcome rather than the API's status.
			if !sync.Delivered() {
				err = fmt.Errorf("sync %s %s", sync.ID, sync.Outcome())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&syncType, "type", string(retlClient.SyncTypeFull), "Sync type: full or incremental")
	cmd.Flags().BoolVar(&wait, "wait", true, "Wait for the sync to reach a verdict")
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Minute, "How long to wait before giving up on a running sync")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func newCmdStop() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop <external-id>",
		Short: "Stop the sync running on a RETL connection",
		Long:  "Cancels the run in progress. Without it, a sync started from the CLI could not be cancelled from the CLI.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			defer func() {
				telemetry.TrackCommand("retl-connections stop", err)
			}()

			d, err := app.NewDeps()
			if err != nil {
				return err
			}

			connectionID, err := resolveConnection(cmd.Context(), d, args[0])
			if err != nil {
				return err
			}

			store := retlClient.NewRudderRETLStore(d.Client())
			if err = store.StopSync(cmd.Context(), connectionID); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "stop requested for %s\n", args[0])
			return nil
		},
	}

	return cmd
}

// awaitSync polls until the run reaches a verdict. A sync the CLI just started
// can be missing from the history for a moment, which is not an error.
func awaitSync(ctx context.Context, store retlClient.RETLStore, connectionID, syncID string, timeout time.Duration) (*retlClient.Sync, error) {
	deadline := time.After(timeout)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		sync, err := store.GetSync(ctx, connectionID, syncID)
		if err == nil && sync.Status != retlClient.SyncRunning {
			return sync, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline:
			return nil, fmt.Errorf("sync %s did not finish within %s; it is still running", syncID, timeout)
		case <-ticker.C:
		}
	}
}

func printJSON(cmd *cobra.Command, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(b))
	return nil
}
