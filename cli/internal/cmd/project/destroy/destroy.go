package destroy

import (
	"context"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/telemetry"
	"github.com/spf13/cobra"
)

var (
	destroyLog = logger.New("root", logger.Attr{
		Key:   "cmd",
		Value: "destroy",
	})
)

func NewCmdDestroy() *cobra.Command {
	var (
		deps    app.Deps
		err     error
		dryRun  bool
		confirm bool
	)

	cmd := &cobra.Command{
		Use:   "destroy",
		Short: "Delete all resources from both the upstream system and state",
		Long: heredoc.Doc(`
			Deletes all resources from both the upstream system and local state.
			This operation is destructive and will remove ALL resources managed
			by the CLI, regardless of any configuration files.
			Use with extreme caution.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli destroy
			$ rudder-cli destroy --dry-run
			$ rudder-cli destroy --confirm=false
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			deps, err = app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			destroyLog.Debug("destroy", "dryRun", dryRun, "confirm", confirm)
			destroyLog.Debug("identifying all resources to destroy")

			defer func() {
				telemetry.TrackCommand("destroy", err, []telemetry.KV{
					{K: "dryRun", V: dryRun},
					{K: "confirm", V: confirm},
				}...)
			}()

			options := []syncer.Option{
				syncer.WithDryRun(dryRun),
				syncer.WithAskConfirmation(confirm),
				syncer.WithReporter(app.SyncReporter()),
			}

			if config.GetConfig().ExperimentalFlags.ConcurrentSyncs {
				options = append(options, syncer.WithConcurrency(config.GetConfig().Concurrency.Syncer))
			}

			s, err := syncer.New(deps.CompositeProvider(), &client.Workspace{}, options...)
			if err != nil {
				return err
			}

			// Destroy all resources
			errors := s.Destroy(context.Background())
			if len(errors) > 0 {
				return fmt.Errorf("destroying resources: %w", errors[0])
			}

			if dryRun {
				destroyLog.Info("Dry run completed. No resources were destroyed.")
			} else {
				destroyLog.Info("Successfully destroyed all resources")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Only show the resources that would be destroyed without actually destroying them")
	cmd.Flags().BoolVar(&confirm, "confirm", true, "Confirm before destroying resources")

	return cmd
}
