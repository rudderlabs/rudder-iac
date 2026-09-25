package experimental

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdExperimental() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "experimental",
		Short:   "Manage experimental features",
		Long:    "Inspect and change opt-in rudder-cli feature flags stored in the selected configuration file. The experimental command group itself must be enabled before these commands can run.",
		Example: "  rudder-cli experimental list\n  rudder-cli experimental enable importMerge",
		Hidden:  true,
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
