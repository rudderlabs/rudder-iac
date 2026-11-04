package auth

import (
	"os"

	"github.com/rudderlabs/rudder-iac/cli/internal/auth"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/spf13/cobra"
)

func NewCmdAuth() *cobra.Command {

	var authCmd = &cobra.Command{
		Use:   "auth",
		Short: "Authentication commands",
	}

	var (
		withToken  bool
		noBrowser  bool
		jsonOutput bool
	)

	var loginCmd = &cobra.Command{
		Use:   "login",
		Short: "Sign in through the browser and store a workspace token in the OS keychain",
		Long: `Sign in through the browser and store a workspace token in the OS keychain.

Use --with-token in CI or where no keychain exists: it reads a token from stdin
and stores it in the config file.`,
		Example: `  rudder-cli auth login
  rudder-cli auth login --no-browser
  echo "$TOKEN" | rudder-cli auth login --with-token`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			defer func() {
				telemetry.TrackCommand(telemetry.CommandName(cmd), err)
			}()

			if withToken {
				return auth.LoginWithToken(os.Stdin)
			}

			return auth.Login(cmd.Context(), auth.LoginOptions{
				NoBrowser:  noBrowser,
				JSONOutput: jsonOutput,
				UserAgent:  "rudder-cli/" + cmd.Root().Version,
				Stdout:     cmd.OutOrStdout(),
				Stderr:     cmd.ErrOrStderr(),
			})
		},
	}
	loginCmd.Flags().BoolVar(&withToken, "with-token", false, "Read an access token from stdin instead of signing in through the browser")
	loginCmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Print the sign-in URL without opening a browser")
	loginCmd.Flags().BoolVar(&jsonOutput, "json", false, "Print the result as one JSON line on stdout")
	loginCmd.MarkFlagsMutuallyExclusive("with-token", "no-browser")

	var logoutCmd = &cobra.Command{
		Use:   "logout",
		Short: "Remove the stored access token",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			defer func() {
				telemetry.TrackCommand(telemetry.CommandName(cmd), err)
			}()

			return auth.Logout(cmd.ErrOrStderr())
		},
	}

	authCmd.AddCommand(loginCmd, logoutCmd)

	return authCmd
}
