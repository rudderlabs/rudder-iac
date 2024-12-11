package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/iac"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(
		&cfgFile,
		"config",
		"c",
		config.DefaultConfigFile(),
		fmt.Sprintf("config file (default is '%s')", config.DefaultConfigFile()),
	)

	store, err := iac.NewStore(
		context.Background(),
		&iac.PulumiConfig{
			Project: "defaultproject",
			Stack: "dev",
			PulumiHome: filepath.Join(filepath.Dir(config.DefaultConfigFile()), "iac/pulumi"),
			ToolVersion: "3.137.0",
		},
	)

	if err != nil {
		log.Fatalf("unable to setup store: %s", err)
	}

	// Add subcommands to the root command
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(debugCmd)
	rootCmd.AddCommand(trackingplan.NewCmdTrackingPlan(store))

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
