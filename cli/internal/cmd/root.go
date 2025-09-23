package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"

	"github.com/kyokomi/emoji/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/auth"
	d "github.com/rudderlabs/rudder-iac/cli/internal/cmd/debug"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/experimental"
	importcmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/import"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/project/apply"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/project/destroy"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/project/validate"
	retlsource "github.com/rudderlabs/rudder-iac/cli/internal/cmd/retl-sources"
	telemetryCmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/typer"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/workspace"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	log     = logger.New("root")
)

func recovery() {
	if r := recover(); r != nil {
		// Always log to file
		log.Error("panic detected", "error", r)
		log.Error(string(debug.Stack()))

		// If debug mode is enabled, show detailed panic info in console and exit
		if viper.GetBool("debug") {
			fmt.Println("\n🔍 Debug Mode: Panic Details")
			fmt.Printf("Error: %v\n", r)
			fmt.Println("\nStack Trace:")
			fmt.Println(string(debug.Stack()))
			os.Exit(1)
		}

		// In non-debug mode, show the simple error message and exit
		fmt.Println(emoji.Sprintf("\n:skull:Oops! Unexpected error occurred. Please contact tech support.\n"))
		os.Exit(1)
	}
}

var (
	debugCmd        *cobra.Command
	experimentalCmd *cobra.Command
)

func init() {
	cobra.OnInitialize(initConfig)
	cobra.OnInitialize(initLogger)
	cobra.OnInitialize(initAppDependencies)
	cobra.OnInitialize(initTelemetry)

	rootCmd.PersistentFlags().StringVarP(
		&cfgFile,
		"config",
		"c",
		config.DefaultConfigFile(),
		fmt.Sprintf("config file (default is '%s')", config.DefaultConfigFile()),
	)

	// Add subcommands to the root command
	rootCmd.AddCommand(auth.NewCmdAuth())
	rootCmd.AddCommand(trackingplan.NewCmdTrackingPlan())
	rootCmd.AddCommand(telemetryCmd.NewCmdTelemetry())
	rootCmd.AddCommand(typer.NewCmdTyper())
	rootCmd.AddCommand(workspace.NewCmdWorkspace())
	rootCmd.AddCommand(importcmd.NewCmdImport())
	rootCmd.AddCommand(retlsource.NewCmdRetlSources())

	rootCmd.AddCommand(apply.NewCmdApply())
	rootCmd.AddCommand(validate.NewCmdValidate())
	rootCmd.AddCommand(destroy.NewCmdDestroy())

	debugCmd = d.NewCmdDebug()
	experimentalCmd = experimental.NewCmdExperimental()

	rootCmd.AddCommand(debugCmd)
	rootCmd.AddCommand(experimentalCmd)
}

func initConfig() {
	config.InitConfig(cfgFile)

	// only add debug command if enabled in config
	if config.GetConfig().Debug {
		debugCmd.Hidden = false
	}

	// reading this property from viper directly as it is not exposed in Config,
	// in order to avoid confusion between Experimental and ExperimentalFlags when used to toggle experimental features
	if viper.GetBool("experimental") {
		experimentalCmd.Hidden = false
	}
}

func initLogger() {
	if config.GetConfig().Debug {
		logger.SetLogLevel(slog.LevelDebug)
	}
}

func initAppDependencies() {
	app.Initialise(rootCmd.Version)
}

func initTelemetry() {
	telemetry.Initialise(rootCmd.Version)
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
