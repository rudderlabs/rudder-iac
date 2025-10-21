package workspace

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/lister"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
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

			format := lister.TableFormat
			if jsonOutput {
				format = lister.JSONFormat
			}
			l := lister.New(dcProvider, format)

			err = l.List(cmd.Context(), state.TrackingPlanResourceType, nil)
			return err
		},
	}

	cmd.Flags().Bool("json", false, "Output as JSON")

	return cmd
}
