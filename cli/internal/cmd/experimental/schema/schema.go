package schema

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

func NewCmdSchema() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema <command>",
		Short: "Manage event schemas and data catalog resources",
		Long:  "Manage the lifecycle of event schemas and data catalog resources using RudderStack Event Audit API",
		Example: heredoc.Doc(`
			$ rudder-cli experimental schema fetch schemas.json
			$ rudder-cli experimental schema unflatten input.json output.json
			$ rudder-cli experimental schema convert schemas.json output/
		`),
	}

	// Add subcommands
	cmd.AddCommand(NewCmdFetch())
	cmd.AddCommand(NewCmdUnflatten())
	cmd.AddCommand(NewCmdConvert())

	return cmd
}
