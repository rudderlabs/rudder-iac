package typer

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/spf13/cobra"
)

func newCmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all tracking plans",
		Long:  "List all tracking plans available in the workspace",
		Args:  cobra.NoArgs,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !config.GetConfig().ExperimentalFlags.RudderTyper {
				return fmt.Errorf("typer commands are disabled")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			defer func() {
				telemetry.TrackCommand("typer list", err)
			}()

			deps, err := app.NewDeps()
			if err != nil {
				return fmt.Errorf("failed to initialize dependencies: %w", err)
			}

			client := deps.Client()
			dataCatalogClient := catalog.NewRudderDataCatalog(client)

			ctx := context.Background()

			spinner := ui.NewSpinner("Fetching tracking plans...")
			spinner.Start()
			trackingPlans, err := dataCatalogClient.GetTrackingPlans(ctx)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to fetch tracking plans: %w", err)
			}

			if len(trackingPlans) == 0 {
				fmt.Println("No tracking plans found")
				return nil
			}

			return printTrackingPlansTable(os.Stdout, trackingPlans)
		},
	}

	return cmd
}

func printTrackingPlansTable(w io.Writer, trackingPlans []*catalog.TrackingPlanWithIdentifiers) error {
	fmt.Fprintf(w, "%-30s %-34s %s\n", "NAME", "ID", "VERSION")
	fmt.Fprintf(w, "%s %s %s\n", "------------------------------", "----------------------------------", "-------")

	for _, tp := range trackingPlans {
		name := tp.Name
		if name == "" {
			name = "- not set -"
		}

		fmt.Fprintf(w, "%-30s %-34s %d\n", name, tp.ID, tp.Version)
	}

	return nil
}
