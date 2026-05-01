package importcmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importer"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/spf13/cobra"
)

func NewCmdWorkspaceImport() *cobra.Command {
	var (
		deps       app.Deps
		p          project.Project
		err        error
		location   string
		filter     string
		onConflict string
	)

	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Import workspace resources",
		Long: `Import upstream workspace resources using available providers into configuration files.

The --filter flag controls which resources are imported:
  - unmanaged (default): Import only resources WITHOUT external IDs (not yet managed by IaC)
  - managed: Import only resources WITH external IDs (already managed by IaC, useful for backup/sync)
  - all: Import both managed and unmanaged resources

The --on-conflict flag controls how to handle resources that already exist locally (only applies when --filter is 'managed' or 'all'):
  - keep-local (default): Skip importing resources that already exist locally
  - accept-incoming: Overwrite local files with the incoming/remote version
  - keep-both: Keep both versions by creating a new file with '-imported' suffix`,
		Example: heredoc.Doc(`
			$ rudder-cli import workspace --location </path/to/project_dir>
			$ rudder-cli import workspace --location </path/to/project_dir> --filter unmanaged
			$ rudder-cli import workspace --location </path/to/project_dir> --filter managed
			$ rudder-cli import workspace --location </path/to/project_dir> --filter managed --on-conflict accept-incoming
			$ rudder-cli import workspace --location </path/to/project_dir> --filter all --on-conflict keep-both
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.GetConfig()

			// Check if experimental flag is enabled for non-default options
			if !cfg.ExperimentalFlags.ImportFilter {
				if filter != "unmanaged" {
					return fmt.Errorf("--filter flag requires experimental feature 'importFilter' to be enabled. Run: rudder-cli experimental enable importFilter")
				}
				if onConflict != "keep-local" {
					return fmt.Errorf("--on-conflict flag requires experimental feature 'importFilter' to be enabled. Run: rudder-cli experimental enable importFilter")
				}
			}

			// Validate filter option
			switch importer.FilterOption(filter) {
			case importer.FilterUnmanaged, importer.FilterManaged, importer.FilterAll:
				// valid
			default:
				return fmt.Errorf("invalid filter option: %q (must be 'unmanaged', 'managed', or 'all')", filter)
			}

			// Validate on-conflict option
			switch importer.ConflictResolution(onConflict) {
			case importer.ConflictKeepLocal, importer.ConflictAcceptIncoming, importer.ConflictKeepBoth:
				// valid
			default:
				return fmt.Errorf("invalid on-conflict option: %q (must be 'keep-local', 'accept-incoming', or 'keep-both')", onConflict)
			}

			deps, err = app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}
			p = deps.NewProject()

			if err := p.Load(location); err != nil {
				return fmt.Errorf("loading and validating project: %w", err)
			}

			if project.HasLegacySpecs(p.Specs()) {
				ui.PrintDeprecationWarning(project.ImportLegacySpecWarning)
			}

			_, err := os.Stat(filepath.Join(location, importer.ImportedDir))
			if err == nil {
				return fmt.Errorf("directory for import: %s already exists", filepath.Join(location, importer.ImportedDir))
			}

			if errors.Is(err, os.ErrNotExist) {
				return nil
			}

			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			defer func() {
				telemetry.TrackCommand("import workspace", err)
			}()

			spinner := ui.NewSpinner("Importing ...")
			spinner.Start()

			opts := importer.ImportOptions{
				Filter:     importer.FilterOption(filter),
				OnConflict: importer.ConflictResolution(onConflict),
			}
			err = importer.WorkspaceImport(cmd.Context(), p, deps.CompositeProvider(), opts)

			spinner.Stop()
			if err == nil {
				ui.PrintSuccess("Done")
			}

			return err
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the directory containing the project files")
	cmd.Flags().StringVarP(&filter, "filter", "f", "unmanaged", "Filter resources to import: 'unmanaged' (default), 'managed', or 'all'")
	cmd.Flags().StringVar(&onConflict, "on-conflict", "keep-local", "Conflict resolution when importing managed resources: 'keep-local' (default), 'accept-incoming', or 'keep-both'")
	return cmd
}
