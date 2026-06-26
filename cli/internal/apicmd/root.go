package apicmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"

	"github.com/kyokomi/emoji/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	deletecmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/delete"
	describecmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/describe"
	getcmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/get"
	setexternalid "github.com/rudderlabs/rudder-iac/cli/internal/cmd/setexternalid"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ponytail: the init/recovery/Execute boilerplate is duplicated from
// cli/internal/cmd rather than extracted into a shared bootstrap — a single
// second binary doesn't justify refactoring the working rudder-cli root.
// Extract a shared helper if a third entrypoint ever appears.

var (
	cfgFile string
	log     = logger.New("rudder-api")
)

var rootCmd = &cobra.Command{
	Use:   "rudder-api",
	Short: "Imperative resource operations over the RudderStack API",
	Long: `rudder-api exposes kubectl-style verbs (get, describe, delete,
set-external-id) for managing RudderStack resources directly. It reuses the same
provider layer as rudder-cli, but surfaces the resource verbs as first-class
commands instead of the experimental, IaC-oriented surface.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	cobra.OnInitialize(initConfig, initLogger, initAppDependencies, initTelemetry)

	rootCmd.PersistentFlags().StringVarP(
		&cfgFile,
		"config",
		"c",
		config.DefaultConfigFile(),
		fmt.Sprintf("config file (default is '%s')", config.DefaultConfigFile()),
	)

	// The same verb constructors rudder-cli registers — here un-gated and
	// first-class, since imperative resource ops are this binary's whole purpose.
	// The constructors default to Hidden:true (experimental in rudder-cli); flip
	// them visible here.
	for _, c := range []*cobra.Command{
		getcmd.NewCmdGet(),
		describecmd.NewCmdDescribe(),
		setexternalid.NewCmdSetExternalID(),
		deletecmd.NewCmdDelete(),
	} {
		c.Hidden = false
		rootCmd.AddCommand(c)
	}
}

func initConfig() {
	config.InitConfig(cfgFile)
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

// SetVersion sets the binary version, mirroring cmd.SetVersion for rudder-cli.
func SetVersion(v string) {
	rootCmd.Version = v
}

func recovery() {
	if r := recover(); r != nil {
		log.Error("panic detected", "error", r)
		log.Error(string(debug.Stack()))

		if viper.GetBool("debug") {
			fmt.Println("\n🔍 Debug Mode: Panic Details")
			fmt.Printf("Error: %v\n", r)
			fmt.Println("\nStack Trace:")
			fmt.Println(string(debug.Stack()))
			os.Exit(1)
		}

		fmt.Println(emoji.Sprintf("\n:skull:Oops! Unexpected error occurred. Please contact tech support.\n"))
		os.Exit(1)
	}
}

// Execute runs the rudder-api root command, mirroring cmd.Execute's error
// handling (SilentError skips the stderr print).
func Execute() {
	defer recovery()

	if err := rootCmd.Execute(); err != nil {
		var silent *cmderrors.SilentError
		if !errors.As(err, &silent) {
			ui.PrintError(err)
		}
		os.Exit(1)
	}
}
