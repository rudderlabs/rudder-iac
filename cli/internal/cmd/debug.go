package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	debugCmd.AddCommand(configCmd)
	debugCmd.AddCommand(configFileCmd)
}

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
	Run: func(cmd *cobra.Command, args []string) {
		config := viper.AllSettings()
		configJSON, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			fmt.Println("Error marshalling config to JSON:", err)
			return
		}
		fmt.Println(string(configJSON))
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
