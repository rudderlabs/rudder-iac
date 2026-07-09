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
		deps     app.Deps
		p        project.Project
		err      error
		location string
		merge    bool
	)

	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Import workspace resources",
		Long:  "Import upstream workspace resources using available providers into configuration files",
		Example: heredoc.Doc(`
			$ rudder-cli import workspace --location </path/to/project_dir>
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if merge && !config.GetConfig().ExperimentalFlags.ImportMerge {
				return fmt.Errorf("--merge requires the %q experimental flag to be enabled", "importMerge")
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

			err = importer.WorkspaceImport(cmd.Context(), p, deps.CompositeProvider(), merge)

			spinner.Stop()
			if err == nil {
				ui.PrintSuccess("Done")
			}

			return err
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the directory containing the project files")
	cmd.Flags().BoolVar(&merge, "merge", false, "Allow import on a diverged project, linking remote resources that match existing local resources (experimental)")
	return cmd
}
