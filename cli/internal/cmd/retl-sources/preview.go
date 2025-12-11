package retlsource

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/previewer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/telemetry"
	"github.com/spf13/cobra"
)

func newCmdPreview() *cobra.Command {
	var location string
	var limit int
	var jsonOutput bool
	var interactive bool

	cmd := &cobra.Command{
		Use:   "preview <external-id>",
		Short: "Preview a RETL source SQL model",
		Long:  "Preview a RETL source SQL model to see the data structure and sample rows",
		Example: heredoc.Doc(`
			$ rudder-cli retl-sources preview my-model
			$ rudder-cli retl-sources preview my-model --location ./project --limit 5
			$ rudder-cli retl-sources preview my-model --interactive=false
			$ rudder-cli retl-sources preview my-model --json
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("retl-source external id is required")
			}
			externalID := args[0]

			var err error
			defer func() {
				telemetry.TrackCommand("retl-sources preview", err, []telemetry.KV{
					{K: "json", V: jsonOutput},
					{K: "interactive", V: interactive},
					{K: "limit", V: limit},
				}...)
			}()

			d, err := app.NewDeps()
			if err != nil {
				return err
			}

			p := project.New(location, d.CompositeProvider())
			if err := p.Load(); err != nil {
				return fmt.Errorf("loading project: %w", err)
			}

			graph, err := p.ResourceGraph()
			if err != nil {
				return fmt.Errorf("getting resource graph: %w", err)
			}
			resource, ok := graph.GetResource(sqlmodel.ResourceType + ":" + externalID)
			if !ok {
				return fmt.Errorf("resource with external id '%s' not found in project", externalID)
			}
			resourceData := resource.Data()
			resourceType := resource.Type()

			// Get the RETL provider
			retlProvider := d.Providers().RETL
			opts := []previewer.PreviewerOpts{}
			opts = append(opts, previewer.WithJson(jsonOutput))
			opts = append(opts, previewer.WithLimit(limit))
			opts = append(opts, previewer.WithInteractive(interactive))
			previewer := previewer.New(retlProvider, opts...)

			return previewer.Preview(cmd.Context(), externalID, resourceType, resourceData)
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the project directory")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output preview rows as JSON")
	cmd.Flags().IntVar(&limit, "limit", 10, "Number of rows to preview")
	cmd.Flags().BoolVar(&interactive, "interactive", true, "Enable interactive table display")

	return cmd
}
