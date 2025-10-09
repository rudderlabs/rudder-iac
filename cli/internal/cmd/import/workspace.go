package importcmd

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importer"
	"github.com/spf13/cobra"
)

func NewCmdWorkspaceImport() *cobra.Command {
	var (
		deps     app.Deps
		p        project.Project
		err      error
		location string
	)

	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Import workspace resources",
		Long:  "Import upstream workspace resources using available providers into configuration files",
		Example: heredoc.Doc(`
			$ rudder-cli import workspace --location </path/to/project_dir>
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			deps, err = app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}
			p = project.New(location, deps.CompositeProvider())

			if err := p.Load(); err != nil {
				return fmt.Errorf("loading and validating project: %w", err)
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			defer func() {
				telemetry.TrackCommand("import workspace", err)
			}()
			err = importer.WorkspaceImport(cmd.Context(), location, p, deps.CompositeProvider())
			return err
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the directory containing the project files")
	return cmd
}
