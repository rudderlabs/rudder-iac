package hello

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

const supportedName = "claude"

// NewCmdHello returns a command that greets Claude. It deliberately does not
// report telemetry, so it stays out of the root trackedCommands list.
func NewCmdHello() *cobra.Command {
	return &cobra.Command{
		Use:       "hello <name>",
		Short:     "Say hello",
		Long:      "Print a greeting for the given name. Only \"claude\" is supported.",
		Example:   "  rudder-cli hello claude",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{supportedName},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !strings.EqualFold(args[0], supportedName) {
				return fmt.Errorf("unsupported name %q: only %q is supported", args[0], supportedName)
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Hello, Claude!")
			return err
		},
	}
}
