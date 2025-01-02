package trackingplan

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	tpApplyCmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan/apply"
	tpDestroyCmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan/destroy"
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
		`),
	}

	cmd.AddCommand(tpValidateCmd.NewCmdTPValidate())
	cmd.AddCommand(tpApplyCmd.NewCmdTPApply())
	cmd.AddCommand(tpDestroyCmd.NewCmdTPDestroy())

	return cmd
}
