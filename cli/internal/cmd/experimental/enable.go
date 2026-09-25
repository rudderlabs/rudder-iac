package experimental

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/spf13/cobra"
)

func NewCmdEnable() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "enable <flag-name>",
		Short:   "Enable an experimental flag",
		Long:    "Enable one experimental feature by its configuration flag name. Use experimental list to find valid names and inspect their current state.",
		Example: `  rudder-cli experimental enable importMerge`,
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			flagName := args[0]
			config.SetExperimentalFlag(flagName, true)
		},
	}

	return cmd
}
