// Package dev holds the experimental `dev` command group: a local listener
// that captures the events an app sends, and the commands that read them.
package dev

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
)

const disabledMessage = "dev is experimental: set RUDDERSTACK_CLI_EXPERIMENTAL=true and RUDDERSTACK_X_DEV_LISTEN=true, " +
	"or run rudder-cli experimental enable devListen with RUDDERSTACK_CLI_EXPERIMENTAL=true"

func NewCmdDev() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Capture and inspect the events an app sends (experimental)",
		Long: heredoc.Doc(`
			Capture the requests a RudderStack SDK sends to a local listener, and read
			them back as tables or JSON.
		`),
		Hidden: true,
		Args:   groupArgs,
		// A child with its own PersistentPreRunE would replace this check.
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if config.GetConfig().ExperimentalFlags.DevListen {
				return nil
			}
			return fail(cmd, &usageError{message: disabledMessage, next: "rudder-cli experimental enable devListen"})
		},
		RunE: showHelp,
	}

	cmd.AddCommand(newCmdListen())

	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		return fail(c, &usageError{message: err.Error(), next: c.CommandPath() + " --help"})
	})

	return cmd
}

// showHelp is the RunE of a group. Without it Cobra prints help and exits 0
// for any argument, a mistyped subcommand included.
func showHelp(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
