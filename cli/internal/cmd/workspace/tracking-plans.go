package workspace

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	"github.com/spf13/cobra"
)

func NewCmdTrackingPlans() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tracking-plans",
		Short: "Manage tracking plans in the workspace",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newCmdListTrackingPlans())

	return cmd
}

func newCmdListTrackingPlans() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tracking plans in the workspace",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOutput, _ := cmd.Flags().GetBool("json")

			var err error
			defer func() {
				telemetry.TrackCommand("workspace tracking-plans list", err, []telemetry.KV{
					{K: "json", V: jsonOutput},
				}...)
			}()

			d, err := app.NewDeps()
			if err != nil {
				return err
			}

			// Cast the DataCatalog provider to access the List method
			dcProvider, ok := d.Providers().DataCatalog.(*datacatalog.Provider)
			if !ok {
				return fmt.Errorf("failed to cast DataCatalog provider")
			}

			// Get tracking plans
			trackingPlans, err := dcProvider.List(cmd.Context(), "tracking-plan", nil)
			if err != nil {
				return err
			}

			if jsonOutput {
				// Output as JSON
				for _, tp := range trackingPlans {
					fmt.Printf("{\"name\":\"%v\",\"id\":\"%v\",\"version\":%v}\n",
						tp["name"], tp["id"], tp["version"])
				}
			} else {
				// Output as simple table
				fmt.Printf("%-30s %-34s %s\n", "NAME", "ID", "VERSION")
				fmt.Printf("%s %s %s\n",
					"------------------------------",
					"----------------------------------",
					"-------")

				for _, tp := range trackingPlans {
					name := fmt.Sprintf("%v", tp["name"])
					if name == "" {
						name = "- not set -"
					}
					fmt.Printf("%-30s %-34s %v\n", name, tp["id"], tp["version"])
				}
			}

			return nil
		},
	}

	cmd.Flags().Bool("json", false, "Output as JSON")

	return cmd
}
