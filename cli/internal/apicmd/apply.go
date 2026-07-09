package apicmd

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

var applyLog = logger.New("rudder-api", logger.Attr{Key: "cmd", Value: "apply"})

// newCmdApply is the scoped, delete-free apply — it lives ONLY in rudder-api.
// rudder-cli's `apply` stays the whole-project reconcile (--location). This mode
// applies only the resources in the given -f files/dirs and never deletes
// anything outside them (syncer.WithScopeToTarget), so blast radius is bounded.
func newCmdApply() *cobra.Command {
	var (
		deps      app.Deps
		p         project.Project
		workspace *client.Workspace
		err       error
		files     []string
		dryRun    bool
		confirm   bool
		varFiles  []string
	)

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply only the resources in the given files (scoped, never deletes)",
		Long: heredoc.Doc(`
			Applies ONLY the resources in the given files or directories. Resources
			outside those paths are NEVER deleted — this mode only creates and updates.
			Each -f path is loaded and validated independently; cross-file consistency
			checks are not performed across distinct -f paths, so pass non-overlapping
			sets to avoid ambiguous results.
		`),
		Example: heredoc.Doc(`
			$ rudder-api apply -f sources.yaml
			$ rudder-api apply -f sources.yaml -f destinations.yaml
			$ rudder-api apply -f ./tracking-plans/ --dry-run
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(files) == 0 {
				return fmt.Errorf("at least one --file/-f is required")
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

			// Load each path individually; handlers accumulate resources across
			// calls so multi-file -f builds a combined resource graph.
			for _, f := range files {
				if err := p.Load(f); err != nil {
					return fmt.Errorf("loading and validating project from %s: %w", f, err)
				}
			}

			if project.HasLegacySpecs(p.Specs()) {
				ui.PrintDeprecationWarning(project.LegacySpecDeprecationWarning)
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			applyLog.Debug("apply", "files", files, "dryRun", dryRun, "confirm", confirm, "scoped", true)

			defer func() {
				telemetry.TrackCommand("apply", err, []telemetry.KV{
					{K: "files", V: len(files)},
					{K: "dryRun", V: dryRun},
					{K: "confirm", V: confirm},
					{K: "scoped", V: true},
				}...)
			}()

			graph, err := p.ResourceGraph()
			if err != nil {
				return fmt.Errorf("getting resource graph: %w", err)
			}

			cfg := config.GetConfig()
			options := []syncer.Option{
				syncer.WithDryRun(dryRun),
				syncer.WithAskConfirmation(confirm),
				syncer.WithReporter(app.SyncReporter()),
				syncer.WithScopeToTarget(),
			}
			if cfg.ExperimentalFlags.ConcurrentSyncs {
				options = append(options, syncer.WithConcurrency(cfg.Concurrency.Syncer))
			}

			s, err := syncer.New(deps.CompositeProvider(), workspace, options...)
			if err != nil {
				return err
			}

			if err = s.Sync(context.Background(), graph); err != nil {
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

	cmd.Flags().StringArrayVarP(&files, "file", "f", nil, "Apply ONLY the resources in these files or directories (scoped: creates/updates only, never deletes). Required, repeatable.")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Only show the changes without applying them")
	cmd.Flags().BoolVar(&confirm, "confirm", true, "Confirm changes before applying them")
	cmd.Flags().StringArrayVar(&varFiles, "var-file", nil, "Path to a variable file ending in .vars.yaml or .vars.yml (repeatable; later files take priority)")

	return cmd
}
