package experimental

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/spf13/cobra"
)

func NewCmdReset() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset all experimental flags to their defaults",
		Long:  "Reset all experimental flags by removing the experimental section from the configuration",
		Example: heredoc.Doc(`
			RUDDERSTACK_CLI_EXPERIMENTAL=true rudder-cli experimental reset
		`),
		Run: func(cmd *cobra.Command, args []string) {
			config.ResetExperimentalFlags()
		},
	}

	return cmd
}
