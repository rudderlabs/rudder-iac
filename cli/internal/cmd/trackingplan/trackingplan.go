package trackingplan

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/iac"
	tpApplyCmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan/apply"
	tpDestroyCmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan/destroy"
	tpPushCmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan/push"
	tpValidateCmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan/validate"
)

func NewCmdTrackingPlan(store *iac.Store) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "tp <command>",
		Short: "Manage datacatalog resources",
		Long:  "Manage the lifecycle of datacatalog resources using user defined state",

		Example: heredoc.Doc(`
			$ rudder-cli tp validate
			$ rudder-cli tp push
		`),
	}

	cmd.AddCommand(tpValidateCmd.NewCmdTPValidate())
	cmd.AddCommand(tpPushCmd.NewCmdTPPush(store))
	cmd.AddCommand(tpDestroyCmd.NewCmdTPDestroy(store))
	cmd.AddCommand(tpApplyCmd.NewCmdTPApply())

	return cmd
}
