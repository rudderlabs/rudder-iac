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
		Use:    "debug",
		Short:  "Debug commands",
		Hidden: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !viper.GetBool("debug") {
				return fmt.Errorf("debug commands are disabled")
			}
			return nil
		},
	}

	var configCmd = &cobra.Command{
		Use:   "config",
		Short: "Dump the active configuration",
		Args:  cobra.NoArgs,
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
		Use:   "config-file",
		Short: "Print the path to the active configuration file",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(viper.ConfigFileUsed())
		},
	}

	debugCmd.AddCommand(configCmd)
	debugCmd.AddCommand(configFileCmd)

	return debugCmd
}
