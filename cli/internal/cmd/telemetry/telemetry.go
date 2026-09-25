package telemetry

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/telemetry"
	"github.com/spf13/cobra"
)

var log = logger.New("telemetry")

func NewCmdTelemetry() *cobra.Command {
	telemetryCmd := &cobra.Command{
		Use:   "telemetry",
		Short: "Manage telemetry settings",
		Long: heredoc.Doc(`
		Manage telemetry settings for the CLI.
		
		Telemetry helps us understand how the CLI is being used and helps us improve it.
		No sensitive information is collected. The data collected includes:
		- Command usage statistics
		- Error occurrences (without sensitive details)
		- Basic system information
		
		Use 'status' to check current telemetry settings.
		Use 'enable' or 'disable' to modify telemetry collection.
	`),
		Example: "  rudder-cli telemetry status\n  rudder-cli telemetry disable",
	}

	telemetryCmd.AddCommand(
		&cobra.Command{
			Use:     "enable",
			Short:   "Enable telemetry",
			Long:    "Enable anonymous rudder-cli usage and error telemetry in the selected configuration file.",
			Example: `  rudder-cli telemetry enable`,
			RunE: func(cmd *cobra.Command, args []string) error {
				telemetry.EnableTelemetry()
				log.Info("telemetry has been enabled")
				return nil
			},
		},
		&cobra.Command{
			Use:     "disable",
			Short:   "Disable telemetry",
			Long:    "Disable rudder-cli usage and error telemetry in the selected configuration file.",
			Example: `  rudder-cli telemetry disable`,
			RunE: func(cmd *cobra.Command, args []string) error {
				telemetry.DisableTelemetry()
				log.Info("telemetry has been disabled")
				return nil
			},
		},
		&cobra.Command{
			Use:     "status",
			Short:   "Show current telemetry status",
			Long:    "Display whether telemetry collection is currently enabled or disabled after resolving the active CLI configuration.",
			Example: `  rudder-cli telemetry status`,
			RunE: func(cmd *cobra.Command, args []string) error {
				status := "enabled"
				if config.GetConfig().Telemetry.Disabled {
					status = "disabled"
				}

				fmt.Printf("telemetry is currently %s\n", status)
				return nil
			},
		},
	)

	return telemetryCmd
}
