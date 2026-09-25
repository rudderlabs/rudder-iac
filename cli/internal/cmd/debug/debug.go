package debug

import (
	"encoding/json"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdDebug() *cobra.Command {
	var debugCmd = &cobra.Command{
		Use:     "debug",
		Short:   "Debug commands",
		Long:    "Inspect the effective CLI configuration, configuration file location, and the latest recorded panic stack trace. Enable debug mode in the CLI configuration before using these diagnostics.",
		Example: "  rudder-cli debug config\n  rudder-cli debug stacktrace",
		Hidden:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !viper.GetBool("debug") {
				return fmt.Errorf("debug commands are disabled")
			}
			return nil
		},
	}

	var configCmd = &cobra.Command{
		Use:     "config",
		Short:   "Dump the active configuration",
		Long:    "Print the effective rudder-cli configuration as JSON after file values, environment variables, and defaults have been resolved.",
		Example: `  rudder-cli debug config`,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.GetConfig()
			configJSON, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return err
			}

			fmt.Println(string(configJSON))
			return nil
		},
	}

	var configFileCmd = &cobra.Command{
		Use:     "config-file",
		Short:   "Print the path to the active configuration file",
		Long:    "Print the configuration file selected by discovery or the global --config flag. Use this when diagnosing which persisted settings rudder-cli loaded.",
		Example: `  rudder-cli debug config-file\n  rudder-cli --config ./rudder.yaml debug config-file`,
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(viper.ConfigFileUsed())
		},
	}

	debugCmd.AddCommand(configCmd)
	debugCmd.AddCommand(configFileCmd)
	debugCmd.AddCommand(newCmdStacktrace())

	return debugCmd
}
