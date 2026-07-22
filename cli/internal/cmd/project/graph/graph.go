package graph

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	graphrender "github.com/rudderlabs/rudder-iac/cli/internal/graph"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/spf13/cobra"
)

var graphLog = logger.New("root", logger.Attr{
	Key:   "cmd",
	Value: "graph",
})

func NewCmdGraph() *cobra.Command {
	var (
		deps       app.Deps
		p          project.Project
		err        error
		location   string
		format     string
		typeFilter string
		varFiles   []string
	)

	cmd := &cobra.Command{
		Use:   "graph [path]",
		Short: "Export the project's resource dependency graph",
		Long: heredoc.Doc(`
			Exports the resource dependency graph of a project.

			The 'dot' and 'mermaid' formats are meant for humans (assessing
			blast-radius during review, onboarding). The 'json' format is a
			stable machine contract consumed by the VS Code graph view and by
			agents.

			This is a pure, read-only export: it loads the local project and
			renders its graph. It never mutates any resource.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli graph
			$ rudder-cli graph ./project --format dot | dot -Tsvg > graph.svg
			$ rudder-cli graph --format mermaid
			$ rudder-cli graph --format json --type event
		`),
		Args: cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// A positional path argument overrides --location when provided.
			if len(args) == 1 {
				location = args[0]
			}

			deps, err = app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			projectOpts, err := app.NewProjectOptions(config.GetConfig(), varFiles)
			if err != nil {
				return err
			}

			p = deps.NewProject(projectOpts...)
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			graphLog.Debug("graph", "location", location, "format", format, "type", typeFilter)

			defer func() {
				telemetry.TrackCommand("graph", err, []telemetry.KV{
					{K: "format", V: format},
				}...)
			}()

			if err = p.Load(location); err != nil {
				return fmt.Errorf("loading project: %w", err)
			}

			g, err := p.ResourceGraph()
			if err != nil {
				return fmt.Errorf("building resource graph: %w", err)
			}

			if err = graphrender.Render(cmd.OutOrStdout(), g, graphrender.Options{
				Format:     graphrender.Format(format),
				TypeFilter: typeFilter,
			}); err != nil {
				return fmt.Errorf("rendering graph: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the directory containing the project files or a specific file")
	cmd.Flags().StringVarP(&format, "format", "f", string(graphrender.FormatDOT), "Output format: dot, mermaid, or json")
	cmd.Flags().StringVarP(&typeFilter, "type", "t", "", "Restrict the graph to resources of this type (default: whole graph)")
	cmd.Flags().StringArrayVar(&varFiles, "var-file", nil, "Path to a variable file ending in .vars.yaml or .vars.yml (repeatable; later files take priority)")
	return cmd
}
