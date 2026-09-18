package apply

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/reporters"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/renderer"
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
		deps       app.Deps
		p          project.Project
		workspace  *client.Workspace
		err        error
		location   string
		dryRun     bool
		confirm    bool
		varFiles   []string
		jsonOutput bool
		// Diagnostics are buffered rather than written straight out: on a clean
		// load apply's document is the plan, and an empty diagnostics document
		// ahead of it would mean two JSON values on stdout and nothing that
		// parses. Only a failed load promotes the buffer to the output.
		diagnostics bytes.Buffer
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
			$ rudder-cli apply --dry-run --json
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			deps, err = app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			workspace, err = deps.Client().Workspaces.GetByAuthToken(context.Background())
			if err != nil {
				return fmt.Errorf("fetching workspace information: %w", err)
			}

			projectOpts, err := app.NewProjectOptions(varFiles)
			if err != nil {
				return err
			}
			projectOpts = append(projectOpts, project.WithWorkspaceID(workspace.ID))

			// A validation failure during load would otherwise render text
			// diagnostics into the middle of the JSON document.
			if jsonOutput {
				projectOpts = append(projectOpts, project.WithRenderer(
					renderer.NewJSONRenderer(&diagnostics),
				))
			}

			p = deps.NewProject(projectOpts...)

			// Load and validate the project configuration
			if err := p.Load(location); err != nil {
				if jsonOutput {
					if _, copyErr := io.Copy(cmd.OutOrStdout(), &diagnostics); copyErr != nil {
						applyLog.Error("writing json diagnostics", "error", copyErr)
					}
				}
				return fmt.Errorf("loading and validating project: %w", err)
			}

			if project.HasLegacySpecs(p.Specs()) && !jsonOutput {
				ui.PrintDeprecationWarning(project.LegacySpecDeprecationWarning)
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

			// Get resource graph to understand dependencies
			graph, err := p.ResourceGraph()
			if err != nil {
				return fmt.Errorf("getting resource graph: %w", err)
			}

			reporter := app.SyncReporter()
			if jsonOutput {
				jsonReporter := reporters.NewJSONSyncReporter(cmd.OutOrStdout(), dryRun)
				// Deferred rather than flushed at the end of RunE: a dry run, an
				// empty plan, a declined confirmation and a failed operation all
				// leave by different paths, and every one of them still owes the
				// caller a document.
				defer func() {
					if flushErr := jsonReporter.Flush(); flushErr != nil {
						applyLog.Error("flushing json report", "error", flushErr)
					}
				}()
				reporter = jsonReporter
			}

			options := []syncer.Option{
				syncer.WithDryRun(dryRun),
				syncer.WithAskConfirmation(confirm),
				syncer.WithReporter(reporter),
				syncer.WithConcurrency(config.GetConfig().Concurrency.Syncer),
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
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output the plan and results as JSON")
	cmd.Flags().StringArrayVar(&varFiles, "var-file", nil, "Path to a variable file ending in .vars.yaml or .vars.yml (repeatable; later files take priority)")

	return cmd
}
