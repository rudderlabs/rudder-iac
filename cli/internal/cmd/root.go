package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"

	"github.com/kyokomi/emoji/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	log     = logger.New("root")
)

func recovery() {
	if r := recover(); r != nil {
		fmt.Println(emoji.Sprintf("\n:skull:Oops! Unexpected error occurred. Please contact tech support.\n"))
		log.Error("panic detected", "error", r)
		log.Error(string(debug.Stack()))

		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	cobra.OnInitialize(initLogger)
	cobra.OnInitialize(initAppDependencies)

	rootCmd.PersistentFlags().StringVarP(
		&cfgFile,
		"config",
		"c",
		config.DefaultConfigFile(),
		fmt.Sprintf("config file (default is '%s')", config.DefaultConfigFile()),
	)

	// Add subcommands to the root command
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(debugCmd)
	rootCmd.AddCommand(trackingplan.NewCmdTrackingPlan())

}

func initConfig() {
	config.InitConfig(cfgFile)

	// only add debug command if enabled in config
	if viper.GetBool("debug") {
		debugCmd.Hidden = false
	}
}

func initLogger() {
	if viper.GetBool("debug") {
		logger.SetLogLevel(slog.LevelDebug)
	}
}

func initAppDependencies() {
	if err := app.Initialise(rootCmd.Version); err != nil {
		ui.ShowError(err)
		os.Exit(1)
	}
}

func SetVersion(v string) {
	rootCmd.Version = v
}

var rootCmd = &cobra.Command{
	Use:           "rudder-cli",
	Short:         "Rudder CLI",
	Long:          `Rudder is a CLI tool for managing your projects.`,
	SilenceUsage:  true,
	SilenceErrors: true, // We will handle errors directly in Execute
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() {
	defer recovery()

	if err := rootCmd.Execute(); err != nil {
		ui.ShowError(err)
		os.Exit(1)
	}
}
