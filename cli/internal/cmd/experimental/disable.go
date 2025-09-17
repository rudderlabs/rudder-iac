package experimental

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/spf13/cobra"
)

func NewCmdDisable() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable <flag-name>",
		Short: "Disable an experimental flag",
		Long:  "Disable a specific experimental flag by name",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			flagName := args[0]
			config.SetExperimentalFlag(flagName, false)
		},
	}

	return cmd
}
