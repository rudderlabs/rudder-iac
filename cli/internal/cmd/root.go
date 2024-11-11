package cmd

import (
	"fmt"
	"os"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
)

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", fmt.Sprintf("config file (default is '%s')", config.DefaultConfigFile()))

	// Add subcommands to the root command
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(debugCmd)
}

func initConfig() {
	config.InitConfig(cfgFile)

	// only add debug command if enabled in config
	if viper.GetBool("debug") {
		debugCmd.Hidden = false
	}
}

func SetVersion(v string) {
	rootCmd.Version = v
}

var rootCmd = &cobra.Command{
	Use:   "rudder-cli",
	Short: "Rudder CLI",
	Long:  `Rudder is a CLI tool for managing your projects.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
