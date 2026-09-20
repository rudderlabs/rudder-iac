package retlconnection

import (
	"fmt"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/spf13/cobra"
)

func newCmdSyncs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "syncs",
		Short: "Inspect the sync history of a RETL connection",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newCmdSyncsList())
	cmd.AddCommand(newCmdSyncsView())

	return cmd
}

func newCmdSyncsList() *cobra.Command {
	var (
		limit      int
		page       int
		status     string
		since      time.Duration
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "list <external-id>",
		Short: "List the runs of a RETL connection, newest first",
		Long: heredoc.Doc(`
			Shows the run history: what happened, how long it took, and how many
			rows moved.

			--status filters on the API's own run status (running, succeeded,
			failed), which is not always what the run is shown as. A run listed
			here as "failed" is an API "succeeded" that dropped rows, so
			--status succeeded will include it.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli retl-connections syncs list customers-to-crm
			$ rudder-cli retl-connections syncs list customers-to-crm --limit 5
			$ rudder-cli retl-connections syncs list customers-to-crm --since 24h
			$ rudder-cli retl-connections syncs list customers-to-crm --json
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			defer func() {
				telemetry.TrackCommand("retl-connections syncs list", err, []telemetry.KV{
					{K: "json", V: jsonOutput},
					{K: "limit", V: limit},
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

			req := retlClient.ListSyncsRequest{Page: page, PerPage: limit, Status: retlClient.SyncStatus(status)}
			if since > 0 {
				req.StartedAfter = time.Now().Add(-since)
			}

			store := retlClient.NewRudderRETLStore(d.Client())
			syncs, err := store.ListSyncs(cmd.Context(), connectionID, req)
			if err != nil {
				return err
			}

			if jsonOutput {
				return printJSON(cmd, syncs)
			}

			// A connection that has never run is an empty history, not an
			// error, and an empty table says nothing — so say it.
			if len(syncs.Syncs) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "%s has no runs yet\n", args[0])
				return nil
			}

			fmt.Fprint(cmd.OutOrStdout(), renderSyncTable(syncs.Syncs, time.Now()))
			if syncs.Paging.Total > len(syncs.Syncs) {
				fmt.Fprintf(cmd.OutOrStdout(), "\nshowing %d of %d runs\n", len(syncs.Syncs), syncs.Paging.Total)
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "How many runs to show")
	cmd.Flags().IntVar(&page, "page", 1, "Page of the history to show")
	cmd.Flags().StringVar(&status, "status", "", "Filter on the API run status: running, succeeded or failed")
	cmd.Flags().DurationVar(&since, "since", 0, "Only runs started within this window, e.g. 24h")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func newCmdSyncsView() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "view <external-id> <run-id>",
		Short: "Show one run of a RETL connection in full",
		Long:  "Shows a single run, including the reason it failed. That reason is the point of this command, so it is never abbreviated.",
		Example: heredoc.Doc(`
			$ rudder-cli retl-connections syncs view customers-to-crm dannc59bjios73e2imjg
			$ rudder-cli retl-connections syncs view customers-to-crm dannc59bjios73e2imjg --json
		`),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			defer func() {
				telemetry.TrackCommand("retl-connections syncs view", err, []telemetry.KV{
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
			sync, err := store.GetSync(cmd.Context(), connectionID, args[1])
			if err != nil {
				// The API answers 500 for every run id it cannot resolve —
				// "invalid run id" for a malformed one and a bare "Internal
				// Server Error" for a well-formed one that does not exist — so
				// a typo is indistinguishable from an outage. Checking the
				// history turns the common case into something actionable and
				// leaves a genuine server error alone. DEX-923.
				if absent, listErr := runAbsent(cmd, store, connectionID, args[1]); listErr == nil && absent {
					err = fmt.Errorf("no run %q on rETL connection %q", args[1], args[0])
				}
				return err
			}

			if jsonOutput {
				return printJSON(cmd, sync)
			}
			fmt.Fprint(cmd.OutOrStdout(), renderSync(*sync, time.Now()))
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

// runAbsent reports whether the connection's recent history really lacks the
// run, used only to tell a typo apart from a failing server.
func runAbsent(cmd *cobra.Command, store retlClient.RETLStore, connectionID, syncID string) (bool, error) {
	page, err := store.ListSyncs(cmd.Context(), connectionID, retlClient.ListSyncsRequest{PerPage: 100})
	if err != nil {
		return false, err
	}
	for _, sync := range page.Syncs {
		if sync.ID == syncID {
			return false, nil
		}
	}
	return true, nil
}
