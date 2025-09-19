package importcmd

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/spf13/cobra"
)

func NewWorkspaceImport() *cobra.Command {
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
			// Namer is initialized here to ensure it's ready for use
			// by the composite provider
			idNamer := namer.NewExternalIdNamer(namer.NewKebabCase())

			graph, err := p.GetResourceGraph()
			if err != nil {
				return fmt.Errorf("getting resource graph: %w", err)
			}

			resourcesMap := graph.Resources()
			projectIDs := make([]string, 0, len(resourcesMap))
			for _, r := range resourcesMap {
				projectIDs = append(projectIDs, r.ID())
			}

			if err := idNamer.Load(projectIDs); err != nil {
				return fmt.Errorf("preloading namer with project IDs: %w", err)
			}

			if _, err := deps.CompositeProvider().WorkspaceImport(cmd.Context(), idNamer); err != nil {
				return fmt.Errorf("workspace import (no-op): %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the directory containing the project files")
	return cmd
}
