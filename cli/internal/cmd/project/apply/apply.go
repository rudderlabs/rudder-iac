package apply

import (
	"context"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	applyLog = logger.New("root", logger.Attr{
		Key:   "cmd",
		Value: "apply",
	})
)

// validateApplyFlags returns an error when --file/-f and --location are both
// explicitly provided, since they represent mutually exclusive loading modes.
func validateApplyFlags(files []string, locationChanged bool) error {
	if len(files) > 0 && locationChanged {
		return fmt.Errorf("--file/-f and --location are mutually exclusive")
	}
	return nil
}

// buildSyncOptions assembles the syncer option list from the given parameters.
// When scoped is true, WithScopeToTarget is appended so that no out-of-scope
// resources are ever deleted — used by the -f mode to limit blast radius.
func buildSyncOptions(scoped, dryRun, confirm bool, reporter syncer.SyncReporter, concurrency int, useConcurrency bool) []syncer.Option {
	options := []syncer.Option{
		syncer.WithDryRun(dryRun),
		syncer.WithAskConfirmation(confirm),
		syncer.WithReporter(reporter),
	}

	if useConcurrency {
		options = append(options, syncer.WithConcurrency(concurrency))
	}

	if scoped {
		options = append(options, syncer.WithScopeToTarget())
	}

	return options
}

func NewCmdApply() *cobra.Command {
	var (
		deps      app.Deps
		p         project.Project
		workspace *client.Workspace
		err       error
		location  string
		dryRun    bool
		confirm   bool
		varFiles  []string
		files     []string
	)

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply project configuration changes",
		Long: heredoc.Doc(`
			Applies the project configuration changes to the RudderStack workspace associated with your access token.

			--location (default): Reconciles the ENTIRE project directory. Creates, updates, and
			DELETES resources to match the local state exactly — resources absent from the
			directory will be removed from the workspace (full reconcile).

			--file / -f: Scoped, delete-free mode. Applies ONLY the resources in the given
			files or directories. Resources outside those paths are NEVER deleted — this mode
			only creates and updates. Use this when you want to apply a subset of your project
			without risking unintended deletions of other resources.

			--file and --location are mutually exclusive.
		`),
		Example: heredoc.Doc(`
			# Full project reconcile (can delete out-of-scope resources)
			$ rudder-cli apply --location </path/to/dir or file>
			$ rudder-cli apply --location </path/to/dir or file> --dry-run
			$ rudder-cli apply --location </path/to/dir or file> --confirm=false

			# Scoped apply — only the listed files/dirs, never deletes anything else
			$ rudder-cli apply -f sources.yaml
			$ rudder-cli apply -f sources.yaml -f destinations.yaml
			$ rudder-cli apply -f ./tracking-plans/ --dry-run
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := validateApplyFlags(files, cmd.Flags().Changed("location")); err != nil {
				return err
			}

			deps, err = app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			workspace, err = deps.Client().Workspaces.GetByAuthToken(context.Background())
			if err != nil {
				return fmt.Errorf("fetching workspace information: %w", err)
			}

			projectOpts, err := app.NewProjectOptions(config.GetConfig(), varFiles)
			if err != nil {
				return err
			}
			projectOpts = append(projectOpts, project.WithWorkspaceID(workspace.ID))

			p = deps.NewProject(projectOpts...)

			if len(files) > 0 {
				// Load each path individually; handlers accumulate resources across
				// calls so multi-file -f correctly builds a combined resource graph.
				for _, f := range files {
					if err := p.Load(f); err != nil {
						return fmt.Errorf("loading and validating project from %s: %w", f, err)
					}
				}
			} else {
				if err := p.Load(location); err != nil {
					return fmt.Errorf("loading and validating project: %w", err)
				}
			}

			if project.HasLegacySpecs(p.Specs()) {
				ui.PrintDeprecationWarning(project.LegacySpecDeprecationWarning)
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			scoped := len(files) > 0

			applyLog.Debug("apply", "location", location, "files", files, "dryRun", dryRun, "confirm", confirm, "scoped", scoped)
			applyLog.Debug("identifying changes for the upstream catalog")

			defer func() {
				telemetry.TrackCommand("apply", err, []telemetry.KV{
					{K: "location", V: location},
					{K: "dryRun", V: dryRun},
					{K: "confirm", V: confirm},
					{K: "scoped", V: scoped},
				}...)
			}()

			// Get resource graph to understand dependencies
			graph, err := p.ResourceGraph()
			if err != nil {
				return fmt.Errorf("getting resource graph: %w", err)
			}

			cfg := config.GetConfig()
			options := buildSyncOptions(
				scoped,
				dryRun,
				confirm,
				app.SyncReporter(),
				cfg.Concurrency.Syncer,
				cfg.ExperimentalFlags.ConcurrentSyncs,
			)

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
	cmd.Flags().StringArrayVar(&varFiles, "var-file", nil, "Path to a variable file ending in .vars.yaml or .vars.yml (repeatable; later files take priority)")
	cmd.Flags().StringArrayVarP(&files, "file", "f", nil, "Apply ONLY the resources in these files/dirs (scoped: creates/updates only, never deletes). Mutually exclusive with --location.")

	return cmd
}
