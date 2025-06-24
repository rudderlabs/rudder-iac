package trackingplan

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	tpApplyCmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan/apply"
	tpDestroyCmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan/destroy"
	tpImportFromSourceCmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan/importfromsource"
	tpValidateCmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan/validate"
)

func NewCmdTrackingPlan() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "tp <command>",
		Short: "Manage datacatalog resources",
		Long:  "Manage the lifecycle of datacatalog resources using user defined state",
		Example: heredoc.Doc(`
			$ rudder-cli tp validate
			$ rudder-cli tp apply
			$ rudder-cli tp importFromSource output/
		`),
	}

	cmd.AddCommand(tpValidateCmd.NewCmdTPValidate())
	cmd.AddCommand(tpApplyCmd.NewCmdTPApply())
	cmd.AddCommand(tpDestroyCmd.NewCmdTPDestroy())
	cmd.AddCommand(tpImportFromSourceCmd.NewCmdTPImportFromSource())

	return cmd
}
