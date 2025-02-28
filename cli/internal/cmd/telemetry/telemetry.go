package telemetry

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
	"github.com/spf13/cobra"
)

var log = logger.New("telemetry")

const (
	CommandExecutedEvent = "Command Executed"
)

type KV struct {
	K string
	V interface{}
}

func TrackCommand(command string, err error, extras ...KV) {

	props := map[string]interface{}{
		"command": command,
		"errored": err != nil,
	}

	for _, extra := range extras {
		props[extra.K] = extra.V
	}

	telemetry.TrackEvent(CommandExecutedEvent, props)
}

var telemetryCmd = &cobra.Command{
	Use:   "telemetry",
	Short: "Manage telemetry settings",
	Long: heredoc.Doc(`
		Manage telemetry settings for the CLI.
		
		Telemetry helps us understand how the CLI is being used and helps us improve it.
		No sensitive information is collected. The data collected includes:
		- Command usage statistics
		- Error occurrences (without sensitive details)
		- Basic system information
		
		Use 'status' to check current telemetry settings
		Use 'enable' or 'disable' to modify telemetry collection
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
		conf := config.GetConfig()
		status := "enabled"
		if conf.Telemetry.Disabled {
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
