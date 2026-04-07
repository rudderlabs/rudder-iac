package trackingplan

import (
	"github.com/spf13/cobra"

	tpApplyCmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan/apply"
	tpDestroyCmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan/destroy"
	tpValidateCmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan/validate"
)

func NewCmdTrackingPlan() *cobra.Command {

	cmd := &cobra.Command{
		Use:        "tp <command>",
		Short:      "[Deprecated] Manage datacatalog resources",
		Long:       "[Deprecated] Manage the lifecycle of datacatalog resources using user defined state",
		Deprecated: "all tp subcommands have been replaced by top-level commands (apply, validate, destroy)",
	}

	cmd.AddCommand(tpValidateCmd.NewCmdTPValidate())
	cmd.AddCommand(tpApplyCmd.NewCmdTPApply())
	cmd.AddCommand(tpDestroyCmd.NewCmdTPDestroy())

	return cmd
}
