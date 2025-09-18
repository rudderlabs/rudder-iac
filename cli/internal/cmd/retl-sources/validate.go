package retlsource

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/spf13/cobra"
)

func newCmdValidate() *cobra.Command {
	var location string

	cmd := &cobra.Command{
		Use:   "validate <external-id>",
		Short: "Validate a RETL source SQL model",
		Long:  "Validate a RETL source SQL model by executing the query without returning data",
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

			p := project.New(location, d.CompositeProvider())
			if err := p.Load(); err != nil {
				return fmt.Errorf("loading project: %w", err)
			}

			graph, err := p.GetResourceGraph()
			if err != nil {
				return fmt.Errorf("getting resource graph: %w", err)
			}
			resource, ok := graph.GetResource(sqlmodel.ResourceType + ":" + externalID)
			if !ok {
				return fmt.Errorf("resource with external id '%s' not found in project", externalID)
			}
			resourceData := resource.Data()

			// Get the RETL provider
			retlProvider, ok := d.Providers().RETL.(*retl.Provider)
			if !ok {
				return fmt.Errorf("failed to cast RETL provider")
			}

			// Validate by attempting to preview with limit=0
			_, err = retlProvider.Preview(cmd.Context(), externalID, sqlmodel.ResourceType, resourceData, 0)
			if err != nil {
				fmt.Printf("❌ SQL query failed to execute: %s\n", err.Error())
				return err
			}

			fmt.Println("✅ SQL query executed successfully")
			return nil
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the project directory")

	return cmd
}
