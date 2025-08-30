package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:

  $ source <(rudder-cli completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ rudder-cli completion bash > /etc/bash_completion.d/rudder-cli
  # macOS:
  $ rudder-cli completion bash > /usr/local/etc/bash_completion.d/rudder-cli

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ rudder-cli completion zsh > "${fpath[1]}/_rudder-cli"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ rudder-cli completion fish | source

  # To load completions for each session, execute once:
  $ rudder-cli completion fish > ~/.config/fish/completions/rudder-cli.fish

PowerShell:

  PS> rudder-cli completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> rudder-cli completion powershell > rudder-cli.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}
