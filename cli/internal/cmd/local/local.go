// Package local holds the experimental `local` command group: commands that run a
// piece of RudderStack on this machine, with nothing sent to RudderStack. Today
// that is `local event-stream`, a mock of event-stream ingestion that captures
// the events an app sends, and the commands that read them.
package local

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten"
)

const disabledMessage = "local is experimental: set RUDDERSTACK_CLI_EXPERIMENTAL=true and RUDDERSTACK_X_LOCAL_EVENT_STREAM=true, " +
	"or run rudder-cli experimental enable localEventStream with RUDDERSTACK_CLI_EXPERIMENTAL=true"

func NewCmdLocal() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "local",
		Short:  "Run parts of RudderStack on this machine (experimental)",
		Long:   "Run parts of RudderStack on this machine. Nothing is sent to RudderStack.",
		Hidden: true,
		Args:   groupArgs,
		// A child with its own PersistentPreRunE would replace this check.
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if config.GetConfig().ExperimentalFlags.LocalEventStream {
				return nil
			}
			return fail(cmd, &usageError{
				code: "experimental_disabled", message: disabledMessage, next: "rudder-cli experimental enable localEventStream",
			})
		},
		RunE: showHelp,
	}

	cmd.AddCommand(newCmdEventStream())

	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		// events list reads what is there; only the summary waits.
		if c.Name() == "list" {
			if flag, ok := strings.CutPrefix(err.Error(), "unknown flag: --"); ok && (flag == "wait" || flag == "min") {
				return fail(c, &usageError{
					message: "--wait and --min belong to events summary: events list reads what is there",
					next:    withURL("rudder-cli local event-stream events summary --wait 30s --json", knownURL()),
				})
			}
		}
		return fail(c, &usageError{message: err.Error(), next: c.CommandPath() + " --help"})
	})

	return cmd
}

func newCmdEventStream() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "event-stream",
		Short: "A local event-stream data plane: capture the events an SDK sends and read them back",
		Long:  devlisten.Guide,
		Args:  groupArgs,
		RunE:  showHelp,
	}

	cmd.AddCommand(newCmdServe())
	cmd.AddCommand(newCmdEvents())

	return cmd
}

// showHelp is the RunE of a group. Without it Cobra prints help and exits 0
// for any argument, a mistyped subcommand included.
func showHelp(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
