package retlsource

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/previewer"
	"github.com/spf13/cobra"
)

func newCmdPreview() *cobra.Command {
	var location string
	var limit int
	var jsonOutput bool
	var interactive bool

	cmd := &cobra.Command{
		Use:   "preview <external-id>",
		Short: "Preview a RETL source (SQL model or table)",
		Long:  "Preview a RETL source (SQL model or warehouse table) to see the data structure and sample rows. s3 table sources have no query to preview.",
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

			// Rejected here rather than clamped downstream: previewSQL bounds the
			// query with max(limit, 1) while the limit also travels in the request
			// unchanged, so a negative value would ask the server for -1 rows and
			// the warehouse for 1. validate's limit of 0 is the one deliberate
			// mismatch (no rows returned, one row read to prove the table reads).
			//
			// Below the defer on purpose: every other preview failure records a
			// TrackCommand event, and this one used to be the exception.
			if limit < 0 {
				err = fmt.Errorf("--limit cannot be negative, got %d", limit)
				return err
			}

			d, err := app.NewDeps()
			if err != nil {
				return err
			}

			p := d.NewProject()
			if err := p.Load(location); err != nil {
				return fmt.Errorf("loading project: %w", err)
			}

			graph, err := p.ResourceGraph()
			if err != nil {
				return fmt.Errorf("getting resource graph: %w", err)
			}
			resource, err := findSource(graph, externalID)
			if err != nil {
				return err
			}
			resourceData, err := resolveAccountRef(cmd.Context(), d.Client().Accounts, resource.Data())
			if err != nil {
				return err
			}
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
