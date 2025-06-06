package schema

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdSchema() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema <command>",
		Short: "Manage event schemas and data catalog resources",
		Long:  "Manage the lifecycle of event schemas and data catalog resources using RudderStack Event Audit API",
		Example: heredoc.Doc(`
			$ rudder-cli schema fetch schemas.json
			$ rudder-cli schema unflatten input.json output.json
			$ rudder-cli schema convert schemas.json output/
		`),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !viper.GetBool("experimental") {
				return fmt.Errorf("schema commands require experimental mode. Set RUDDERSTACK_CLI_EXPERIMENTAL=true or add \"experimental\": true to your config file")
			}
			return nil
		},
	}

	// Add subcommands
	cmd.AddCommand(NewCmdFetch())
	cmd.AddCommand(NewCmdUnflatten())
	cmd.AddCommand(NewCmdConvert())

	return cmd
}
