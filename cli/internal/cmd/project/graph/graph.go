package graph

import (
	"encoding/json"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/projectgraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/renderer"
	"github.com/spf13/cobra"
)

var graphLog = logger.New("root", logger.Attr{
	Key:   "cmd",
	Value: "graph",
})

func NewCmdGraph() *cobra.Command {
	var (
		location string
		varFiles []string
	)

	cmd := &cobra.Command{
		Use:   "graph",
		Short: "Emit the project's resource graph and diagnostics as JSON",
		Long: heredoc.Doc(`
			Loads the project, builds its resource graph and prints the resources,
			their dependencies and any validation diagnostics as a single JSON
			document on stdout.

			Intended for tools built on top of the CLI — editor integrations in
			particular — so they can render and navigate a project without
			reimplementing spec parsing or reference resolution.

			Unlike 'validate', this command needs no credentials and makes no
			network calls: it reads only local spec files. A project that fails
			validation still produces output, with the reasons in "diagnostics",
			so the exit code is zero whenever the JSON was written. Because no
			workspace is resolved, workspace-scoped rules may report differently
			than they do under 'validate'.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli graph --location ./specs
			$ rudder-cli graph | jq '.nodes[] | select(.type == "event")'
		`),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			graphLog.Debug("graph", "location", location)

			projectOpts, err := app.NewProjectOptions(varFiles)
			if err != nil {
				return err
			}

			capture := renderer.NewCaptureRenderer()
			projectOpts = append(projectOpts, project.WithRenderer(capture))

			p, err := app.NewOfflineProject(projectOpts...)
			if err != nil {
				return fmt.Errorf("initialising project: %w", err)
			}

			// A load failure is the normal case for an editor: the user is
			// mid-edit. The reasons are already in the captured diagnostics, so
			// keep going and report them in the payload rather than aborting.
			if err := p.Load(location); err != nil {
				graphLog.Debug("project load reported failures", "error", err)
			}

			// Nil on a project that failed syntax validation, since no spec was
			// ever loaded into the providers. Build handles that.
			graph, err := p.ResourceGraph()
			if err != nil {
				graphLog.Debug("building resource graph", "error", err)
				graph = nil
			}

			payload := projectgraph.Build(
				graph,
				capture.Diagnostics(),
				sourceLocations(p),
				app.GetVersion(),
				location,
			)

			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(payload); err != nil {
				return fmt.Errorf("encoding graph: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the directory containing the project files or a specific file")
	cmd.Flags().StringArrayVar(&varFiles, "var-file", nil, "Path to a variable file ending in .vars.yaml or .vars.yml (repeatable; later files take priority)")
	return cmd
}

// sourceLocations pulls per-URN positions off the project when it tracks them,
// and returns nothing when it does not, so the payload degrades to a graph
// without file positions rather than failing.
func sourceLocations(p project.Project) map[string]projectgraph.Location {
	locator, ok := p.(project.SourceLocator)
	if !ok {
		return nil
	}

	locations := make(map[string]projectgraph.Location)
	for urn, loc := range locator.SourceLocations() {
		locations[urn] = projectgraph.Location{
			File:   loc.File,
			Line:   loc.Line,
			Column: loc.Column,
		}
	}

	return locations
}
