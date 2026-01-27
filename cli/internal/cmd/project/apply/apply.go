package apply

import (
	"context"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/spf13/cobra"
)

var (
	applyLog = logger.New("root", logger.Attr{
		Key:   "cmd",
		Value: "apply",
	})
)

func NewCmdApply() *cobra.Command {
	var (
		deps     app.Deps
		p        project.Project
		err      error
		location string
		dryRun   bool
		confirm  bool
	)

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply project configuration changes",
		Long: heredoc.Doc(`
			Applies the project configuration changes to the RudderStack workspace associated with your access token.
			This includes creating, updating, or deleting resources based on
			the differences between local configuration and the workspace resources.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli apply --location </path/to/dir or file>
			$ rudder-cli apply --location </path/to/dir or file> --dry-run
			$ rudder-cli apply --location </path/to/dir or file> --confirm=false
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			deps, err = app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			p = deps.NewProject()

			// Load and validate the project configuration
			if err := p.Load(location); err != nil {
				return fmt.Errorf("loading and validating project: %w", err)
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			applyLog.Debug("apply", "location", location, "dryRun", dryRun, "confirm", confirm)
			applyLog.Debug("identifying changes for the upstream catalog")

			defer func() {
				telemetry.TrackCommand("apply", err, []telemetry.KV{
					{K: "location", V: location},
					{K: "dryRun", V: dryRun},
					{K: "confirm", V: confirm},
				}...)
			}()

			workspace, err := deps.Client().Workspaces.GetByAuthToken(context.Background())
			if err != nil {
				return fmt.Errorf("fetching workspace information: %w", err)
			}

			// Get resource graph to understand dependencies
			graph, err := p.ResourceGraph()
			if err != nil {
				return fmt.Errorf("getting resource graph: %w", err)
			}

			options := []syncer.Option{
				syncer.WithDryRun(dryRun),
				syncer.WithAskConfirmation(confirm),
				syncer.WithReporter(app.SyncReporter()),
			}

			if config.GetConfig().ExperimentalFlags.ConcurrentSyncs {
				options = append(options, syncer.WithConcurrency(config.GetConfig().Concurrency.Syncer))
			}

			// Create syncer to handle the changes
			s, err := syncer.New(deps.CompositeProvider(), workspace, options...)
			if err != nil {
				return err
			}

			// Apply the changes
			err = s.Sync(context.Background(), graph)
			if err != nil {
				return fmt.Errorf("syncing resources: %w", err)
			}

			if dryRun {
				applyLog.Info("Dry run completed. No changes were applied.")
			} else {
				applyLog.Info("Successfully applied all changes")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the directory containing the project files or a specific file")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Only show the changes without applying them")
	cmd.Flags().BoolVar(&confirm, "confirm", true, "Confirm changes before applying them")

	return cmd
}
