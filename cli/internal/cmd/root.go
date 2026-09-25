package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"

	"github.com/kyokomi/emoji/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/auth"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	datagraphPkg "github.com/rudderlabs/rudder-iac/cli/internal/cmd/datagraph"
	d "github.com/rudderlabs/rudder-iac/cli/internal/cmd/debug"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/experimental"
	importcmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/import"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/project/apply"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/project/destroy"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/project/migrate"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/project/validate"
	retlsource "github.com/rudderlabs/rudder-iac/cli/internal/cmd/retl-sources"
	telemetryCmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/transformations"
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
	cfgFile = config.DefaultConfigFile()
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
	datagraphCmd    *cobra.Command
)

func init() {
	cobra.OnInitialize(initConfig)
	cobra.OnInitialize(initLogger)
	cobra.OnInitialize(initAppDependencies)
	cobra.OnInitialize(initTelemetry)

	debugCmd, _, _ = rootCmd.Find([]string{"debug"})
	experimentalCmd, _, _ = rootCmd.Find([]string{"experimental"})
	datagraphCmd, _, _ = rootCmd.Find([]string{"data-graphs"})
}

func initConfig() {
	if flag := rootCmd.PersistentFlags().Lookup("config"); flag != nil {
		cfgFile = flag.Value.String()
	}
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

var rootCmd = newRootCmd()

// NewDocumentationRootCmd constructs a fresh command tree for documentation and
// command-shape tests without triggering configuration or API setup. Do not
// execute the returned command: runtime initialization is intentionally bound to
// the package-level root command used by Execute.
func NewDocumentationRootCmd() *cobra.Command {
	return newRootCmd()
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rudder-cli",
		Short: "Manage RudderStack resources as code",
		Long: `Manage RudderStack workspace resources from declarative YAML project files.

Use rudder-cli to validate and apply local infrastructure-as-code specs, preview
changes before updating a workspace, import existing remote resources, inspect
workspace entities, and generate tracking-plan types. Authentication, telemetry,
and experimental settings are stored in the CLI configuration file selected by
--config. Start with auth login, then use validate and apply for the normal
project workflow.`,
		Example: `  # Authenticate, validate a project, and preview its changes
  rudder-cli auth login
  rudder-cli validate --location ./project
  rudder-cli apply --location ./project --dry-run`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}
	cmd.CompletionOptions.HiddenDefaultCmd = true
	cmd.SetHelpCommand(newHelpCommand())

	cmd.PersistentFlags().StringP(
		"config",
		"c",
		config.DefaultConfigFile(),
		fmt.Sprintf("config file (default is '%s')", config.DefaultConfigFile()),
	)

	cmd.AddCommand(
		auth.NewCmdAuth(),
		trackingplan.NewCmdTrackingPlan(),
		telemetryCmd.NewCmdTelemetry(),
		workspace.NewCmdWorkspace(),
		importcmd.NewCmdImport(),
		retlsource.NewCmdRetlSources(),
		apply.NewCmdApply(),
		validate.NewCmdValidate(),
		destroy.NewCmdDestroy(),
		migrate.NewCmdMigrate(),
		d.NewCmdDebug(),
		experimental.NewCmdExperimental(),
		typer.NewCmdTyper(),
		transformations.NewCmdTransformations(),
		datagraphPkg.NewCmdDataGraph(),
	)

	return cmd
}

func newHelpCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "help [command]",
		Short: "Show help for a command",
		Long: `Show detailed help for rudder-cli or one of its commands.

Use this command when you want the same information printed by --help but prefer
subcommand syntax, or when you need help for a nested command path. The command
accepts any visible command path and prints its usage, flags, examples, and child
commands without contacting RudderStack APIs.`,
		Example: `  # Show root help
  rudder-cli help

  # Show help for a nested command
  rudder-cli help import workspace`,
		ValidArgsFunction: func(command *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
			var completions []cobra.Completion
			target, _, err := command.Root().Find(args)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			if target == nil {
				target = command.Root()
			}
			for _, subCommand := range target.Commands() {
				if subCommand.IsAvailableCommand() && strings.HasPrefix(subCommand.Name(), toComplete) {
					completions = append(completions, cobra.CompletionWithDesc(subCommand.Name(), subCommand.Short))
				}
			}
			return completions, cobra.ShellCompDirectiveNoFileComp
		},
		Run: func(command *cobra.Command, args []string) {
			target, _, err := command.Root().Find(args)
			if target == nil || err != nil {
				command.Printf("Unknown help topic %#q\n", args)
				cobra.CheckErr(command.Root().Usage())
				return
			}
			cobra.CheckErr(target.Help())
		},
	}
}

// Execute runs the root command. If the command returns an error, it is printed
// to stderr and the process exits with code 1. Errors wrapped in SilentError
// skip the stderr output — the command is expected to have already communicated
// the failure through its primary output (e.g., JSON to stdout).
func Execute() {
	defer recovery()

	if flag := rootCmd.PersistentFlags().Lookup("config"); flag != nil {
		cfgFile = flag.Value.String()
	}
	if err := rootCmd.Execute(); err != nil {
		var silent *cmderrors.SilentError
		if !errors.As(err, &silent) {
			ui.PrintError(err)
		}
		os.Exit(1)
	}
}
