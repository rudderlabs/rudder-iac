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

var telemetryCmd = &cobra.Command{
	Use:   "telemetry",
	Short: "Manage telemetry settings",
	Long: heredoc.Doc(`
		Manage telemetry settings for the CLI.
		
		Telemetry helps us understand how the CLI is being used and helps us improve it.
		It is on until you turn it off. The CLI sends:
		- The command, its flags and their values, and whether it failed
		- The CLI version and a telemetry id made on this machine
		- The CI service, when the CLI runs in one
		- The coding agent running the CLI, when one is
		- The skill the agent follows, when the skill names itself

		While telemetry is enabled, API calls also carry the agent, the skill and
		the telemetry id in their User-Agent. That ties this usage to your
		workspace. Spec contents, access tokens and secrets are never sent.
		The full list: https://github.com/rudderlabs/rudder-iac#telemetry
		
		Use 'status' to check current telemetry settings
		Use 'enable' or 'disable' to modify telemetry collection
		RUDDERSTACK_CLI_TELEMETRY_DISABLED=true or DO_NOT_TRACK=1 also turns it off
	`),
}

var telemetryEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable telemetry",
	Long:  "Enable telemetry collection to help improve the CLI",
	RunE: func(cmd *cobra.Command, args []string) error {
		telemetry.EnableTelemetry()
		log.Info("telemetry has been enabled")
		return nil
	},
}

var telemetryDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable telemetry",
	Long:  "Disable telemetry collection",
	RunE: func(cmd *cobra.Command, args []string) error {
		telemetry.DisableTelemetry()
		log.Info("telemetry has been disabled")
		return nil
	},
}

var telemetryStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current telemetry status",
	Long:  "Display whether telemetry collection is currently enabled or disabled",
	RunE: func(cmd *cobra.Command, args []string) error {
		status := "enabled"

		if config.GetConfig().Telemetry.Disabled {
			status = "disabled"
		}

		fmt.Printf("telemetry is currently %s\n", status)
		return nil
	},
}

func NewCmdTelemetry() *cobra.Command {
	telemetryCmd.AddCommand(telemetryEnableCmd)
	telemetryCmd.AddCommand(telemetryDisableCmd)
	telemetryCmd.AddCommand(telemetryStatusCmd)

	return telemetryCmd
}
