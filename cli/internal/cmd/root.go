package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"

	"github.com/MakeNowJust/heredoc/v2"
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
	cfgFile         string
	debugCmd        *cobra.Command
	experimentalCmd *cobra.Command
	log             = logger.New("root")
	rootCmd         *cobra.Command
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

// Mode controls whether a command tree is created for normal execution or
// documentation. Documentation mode never registers runtime initializers and
// deliberately exposes debug and experimental commands before final filtering.
type Mode int

const (
	ModeRuntime Mode = iota
	ModeDocs
)

// NewRootCommand builds a fresh command tree. Keeping construction independent
// from the package-global runtime root lets documentation and tests inspect the
// CLI without reading config, initializing telemetry, or making network calls.
func NewRootCommand(mode Mode) *cobra.Command {
	var configFile string
	configTarget := &configFile
	configDefault := "~/.rudder/config.json"
	if mode == ModeRuntime {
		configTarget = &cfgFile
		configDefault = config.DefaultConfigFile()
	}

	root := &cobra.Command{
		Use:   "rudder-cli",
		Short: "Manage RudderStack resources as code",
		Long: `Manage RudderStack resources with declarative YAML project files.

Use apply, validate, destroy, and import to manage project state; inspect remote
resources with workspace listings; configure authentication and telemetry; and
use debug or experimental tools when needed. Run help for command guidance or
consult the generated command documentation for the complete reference.`,
		Example: `  # Safely validate local declarative YAML before applying changes
  rudder-cli validate --location ./project

  # Preview the changes without modifying the workspace
  rudder-cli apply --location ./project --dry-run

  # Browse available commands and documentation
  rudder-cli help`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetHelpCommand(newCmdInternalHelp(root))

	root.PersistentFlags().StringVarP(
		configTarget,
		"config",
		"c",
		configDefault,
		fmt.Sprintf("config file (default is '%s')", configDefault),
	)

	root.AddCommand(auth.NewCmdAuth())
	root.AddCommand(newCmdHelp(root))
	root.AddCommand(newCmdCompletion(root))
	root.AddCommand(trackingplan.NewCmdTrackingPlan())
	root.AddCommand(telemetryCmd.NewCmdTelemetry())
	root.AddCommand(workspace.NewCmdWorkspace())
	root.AddCommand(importcmd.NewCmdImport())
	root.AddCommand(retlsource.NewCmdRetlSources())
	root.AddCommand(apply.NewCmdApply())
	root.AddCommand(validate.NewCmdValidate())
	root.AddCommand(destroy.NewCmdDestroy())
	root.AddCommand(migrate.NewCmdMigrate())

	debugCmd := d.NewCmdDebug()
	experimentalCmd := experimental.NewCmdExperimental()
	if mode == ModeDocs {
		debugCmd.Hidden = false
		experimentalCmd.Hidden = false
	}
	root.AddCommand(debugCmd)
	root.AddCommand(experimentalCmd)

	root.AddCommand(typer.NewCmdTyper())
	root.AddCommand(transformations.NewCmdTransformations())
	root.AddCommand(datagraphPkg.NewCmdDataGraph())

	return root
}

func newCmdHelp(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "help [command]",
		Short: "Help about any command",
		Long: heredoc.Doc(`
			Show detailed help for rudder-cli or one of its subcommands.

			Use this command when you need command syntax, available flags, examples, or
			the list of child commands from the installed CLI. Passing a command path shows
			help for that command; omitting it shows the root command help.
		`),
		Example: heredoc.Doc(`
			rudder-cli help
			rudder-cli help apply
			rudder-cli help workspace tracking-plans list
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, _, err := root.Find(args)
			if err != nil || target == nil {
				return fmt.Errorf("unknown help topic %q", args)
			}
			return target.Help()
		},
	}
}

func newCmdInternalHelp(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:    "__help [command]",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			target, _, err := root.Find(args)
			if err != nil || target == nil {
				return fmt.Errorf("unknown help topic %q", args)
			}
			return target.Help()
		},
	}
}

func newCmdCompletion(root *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion <bash|fish|powershell|zsh>",
		Short: "Generate shell completion scripts",
		Long: heredoc.Doc(`
			Generate shell completion scripts for rudder-cli.

			Use this command to install tab completion for a supported shell. Each shell
			subcommand prints the script to standard output so you can source it for the
			current session or redirect it to the location your shell loads at startup.
		`),
		Example: heredoc.Doc(`
			# Load bash completions for the current shell
			source <(rudder-cli completion bash)

			# Install zsh completions for future shells
			rudder-cli completion zsh > "${fpath[1]}/_rudder-cli"
		`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newCmdCompletionBash(root))
	cmd.AddCommand(newCmdCompletionFish(root))
	cmd.AddCommand(newCmdCompletionPowerShell(root))
	cmd.AddCommand(newCmdCompletionZsh(root))
	return cmd
}

func newCmdCompletionBash(root *cobra.Command) *cobra.Command {
	var noDescriptions bool
	cmd := &cobra.Command{
		Use:   "bash",
		Short: "Generate the autocompletion script for bash",
		Long: heredoc.Doc(`
			Generate the autocompletion script for the bash shell.

			The generated script requires the bash-completion package. Source the script
			for the current shell or install it in your bash completion directory so new
			shells load rudder-cli completions automatically.
		`),
		Example: heredoc.Doc(`
			# Load completions for the current shell
			source <(rudder-cli completion bash)

			# Install completions system-wide on Linux
			rudder-cli completion bash > /etc/bash_completion.d/rudder-cli
		`),
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return root.GenBashCompletionV2(cmd.OutOrStdout(), !noDescriptions)
		},
	}
	cmd.Flags().BoolVar(&noDescriptions, "no-descriptions", false, "disable completion descriptions")
	return cmd
}

func newCmdCompletionFish(root *cobra.Command) *cobra.Command {
	var noDescriptions bool
	cmd := &cobra.Command{
		Use:   "fish",
		Short: "Generate the autocompletion script for fish",
		Long: heredoc.Doc(`
			Generate the autocompletion script for the fish shell.

			The generated script can be sourced for the current session or written to the
			fish completions directory so new shells load rudder-cli completions
			automatically.
		`),
		Example: heredoc.Doc(`
			# Load completions for the current shell
			rudder-cli completion fish | source

			# Install completions for future fish shells
			rudder-cli completion fish > ~/.config/fish/completions/rudder-cli.fish
		`),
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return root.GenFishCompletion(cmd.OutOrStdout(), !noDescriptions)
		},
	}
	cmd.Flags().BoolVar(&noDescriptions, "no-descriptions", false, "disable completion descriptions")
	return cmd
}

func newCmdCompletionPowerShell(root *cobra.Command) *cobra.Command {
	var noDescriptions bool
	cmd := &cobra.Command{
		Use:   "powershell",
		Short: "Generate the autocompletion script for powershell",
		Long: heredoc.Doc(`
			Generate the autocompletion script for PowerShell.

			The generated script can be evaluated for the current session or added to your
			PowerShell profile so new sessions load rudder-cli completions automatically.
		`),
		Example: heredoc.Doc(`
			# Load completions for the current shell
			rudder-cli completion powershell | Out-String | Invoke-Expression
		`),
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			if noDescriptions {
				return root.GenPowerShellCompletion(cmd.OutOrStdout())
			}
			return root.GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
		},
	}
	cmd.Flags().BoolVar(&noDescriptions, "no-descriptions", false, "disable completion descriptions")
	return cmd
}

func newCmdCompletionZsh(root *cobra.Command) *cobra.Command {
	var noDescriptions bool
	cmd := &cobra.Command{
		Use:   "zsh",
		Short: "Generate the autocompletion script for zsh",
		Long: heredoc.Doc(`
			Generate the autocompletion script for the zsh shell.

			Source the generated script for the current shell or install it in a directory
			listed in fpath so new zsh sessions load rudder-cli completions automatically.
		`),
		Example: heredoc.Doc(`
			# Load completions for the current shell
			source <(rudder-cli completion zsh)

			# Install completions for future zsh shells
			rudder-cli completion zsh > "${fpath[1]}/_rudder-cli"
		`),
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			if noDescriptions {
				return root.GenZshCompletionNoDesc(cmd.OutOrStdout())
			}
			return root.GenZshCompletion(cmd.OutOrStdout())
		},
	}
	cmd.Flags().BoolVar(&noDescriptions, "no-descriptions", false, "disable completion descriptions")
	return cmd
}

func init() {
	rootCmd = NewRootCommand(ModeRuntime)
	debugCmd, _, _ = rootCmd.Find([]string{"debug"})
	experimentalCmd, _, _ = rootCmd.Find([]string{"experimental"})
	cobra.OnInitialize(initConfig, initLogger, initAppDependencies, initTelemetry)
}

func initConfig() {
	config.InitConfig(cfgFile)
	if config.GetConfig().Debug {
		debugCmd.Hidden = false
	}
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

// DocsEligible is the shared contract for commands included in generated docs.
// Hidden is evaluated after docs-mode exposure, so intentionally documented
// debug and experimental commands are included while final hidden commands are not.
func DocsEligible(command *cobra.Command) bool {
	return command != nil && !command.Hidden && command.Deprecated == ""
}

// PrepareDocsTree removes commands outside the documentation contract and sets
// deterministic generation options on every remaining command.
func PrepareDocsTree(root *cobra.Command) {
	var prepare func(*cobra.Command)
	prepare = func(command *cobra.Command) {
		command.DisableAutoGenTag = true
		for _, child := range append([]*cobra.Command(nil), command.Commands()...) {
			if !DocsEligible(child) {
				command.RemoveCommand(child)
				continue
			}
			prepare(child)
		}
	}
	prepare(root)
}

func SetVersion(v string) {
	rootCmd.Version = v
}

// Execute runs the root command. If the command returns an error, it is printed
// to stderr and the process exits with code 1. Errors wrapped in SilentError
// skip the stderr output — the command is expected to have already communicated
// the failure through its primary output (e.g., JSON to stdout).
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
