package retlsource

import (
	"errors"
	"fmt"
	"io"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/spf13/cobra"
)

func newCmdValidate() *cobra.Command {
	var location string

	cmd := &cobra.Command{
		Use:   "validate <external-id>",
		Short: "Validate a RETL source (SQL model or table)",
		Long:  "Validate a RETL source (SQL model or warehouse table) by executing its query without returning data. s3 table sources have no query to validate.",
		Example: heredoc.Doc(`
			$ rudder-cli retl-sources validate my-model
			$ rudder-cli retl-sources validate my-model --location ./project
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("retl-source external id is required")
			}
			externalID := args[0]
			var err error
			defer func() {
				telemetry.TrackCommand("retl-sources validate", err)
			}()

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

			// Get the RETL provider
			retlProvider := d.Providers().RETL

			// Validate by attempting to preview with limit=0
			ui.StartSpinner("Validating SQL query ...")
			_, err = retlProvider.Preview(cmd.Context(), externalID, resource.Type(), resourceData, 0)
			ui.StopSpinner()
			return reportValidation(cmd.OutOrStdout(), err)
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the project directory")

	return cmd
}

// reportValidation treats ErrPreviewUnsupported (an s3 table source) as a pass,
// so validating every source in a project in CI does not fail on one.
func reportValidation(w io.Writer, err error) error {
	switch {
	case err == nil:
		fmt.Fprintln(w, "✅ SQL query executed successfully")
		return nil
	case errors.Is(err, table.ErrPreviewUnsupported):
		fmt.Fprintf(w, "✅ Nothing to validate: %s\n", err)
		return nil
	default:
		fmt.Fprintf(w, "❌ SQL query failed to execute: %s\n", err)
		return err
	}
}
