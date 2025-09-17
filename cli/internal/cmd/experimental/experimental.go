package experimental

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdExperimental() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "experimental",
		Short:  "Manage experimental features",
		Long:   "Enable, disable, and manage experimental features in rudder-cli",
		Hidden: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !viper.GetBool("experimental") {
				return fmt.Errorf("experimental commands are disabled")
			}
			return nil
		},
	}

	cmd.AddCommand(NewCmdList())
	cmd.AddCommand(NewCmdEnable())
	cmd.AddCommand(NewCmdDisable())
	cmd.AddCommand(NewCmdReset())

	return cmd
}
