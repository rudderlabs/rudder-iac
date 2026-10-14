// Package sources holds the experimental `sources` command group: a local listener
// that captures the events an app sends, and the commands that read them.
package sources

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten"
)

const disabledMessage = "sources is experimental: set RUDDERSTACK_CLI_EXPERIMENTAL=true and RUDDERSTACK_X_SOURCES_LISTEN=true, " +
	"or run rudder-cli experimental enable sourcesListen with RUDDERSTACK_CLI_EXPERIMENTAL=true"

func NewCmdSources() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "sources",
		Short:  "Capture and inspect the events an app sends (experimental)",
		Long:   devlisten.Guide,
		Hidden: true,
		Args:   groupArgs,
		// A child with its own PersistentPreRunE would replace this check.
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if config.GetConfig().ExperimentalFlags.SourcesListen {
				return nil
			}
			return fail(cmd, &usageError{
				code: "experimental_disabled", message: disabledMessage, next: "rudder-cli experimental enable sourcesListen",
			})
		},
		RunE: showHelp,
	}

	cmd.AddCommand(newCmdListen())
	cmd.AddCommand(newCmdEvents())

	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		// sources events list reads what is there; only the summary waits.
		if c.Name() == "list" {
			if flag, ok := strings.CutPrefix(err.Error(), "unknown flag: --"); ok && (flag == "wait" || flag == "min") {
				return fail(c, &usageError{
					message: "--wait and --min belong to sources events: sources events list reads what is there",
					next:    withURL("rudder-cli sources events --wait 30s --json", knownURL()),
				})
			}
		}
		return fail(c, &usageError{message: err.Error(), next: c.CommandPath() + " --help"})
	})

	return cmd
}

// showHelp is the RunE of a group. Without it Cobra prints help and exits 0
// for any argument, a mistyped subcommand included.
func showHelp(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
