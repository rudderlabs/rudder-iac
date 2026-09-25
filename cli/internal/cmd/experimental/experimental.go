package experimental

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdExperimental() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "experimental",
		Short: "Manage experimental features",
		Long:  "List and manage opt-in experimental feature flags. Experimental commands require experimental mode during normal CLI execution.",
		Example: heredoc.Doc(`
			RUDDERSTACK_CLI_EXPERIMENTAL=true rudder-cli experimental list
		`),
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
