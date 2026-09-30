package auth

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/auth"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/spf13/cobra"
)

func NewCmdAuth() *cobra.Command {

	var authCmd = &cobra.Command{
		Use:   "auth",
		Short: "Authentication commands",
		Long:  "Authenticate the CLI with RudderStack. Credentials are stored in the configured local CLI configuration and used by commands that access a workspace.",
		Example: heredoc.Doc(`
			rudder-cli auth --help
		`),
	}

	var loginCmd = &cobra.Command{
		Use:   "login",
		Short: "Login with an access token",
		Long:  "Store a RudderStack access token for subsequent authenticated CLI operations.",
		Example: heredoc.Doc(`
			rudder-cli auth login
		`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			defer func() {
				telemetry.TrackCommand("auth login", err)
			}()

			err = auth.Login()
			return err
		},
	}

	authCmd.AddCommand(loginCmd)

	return authCmd
}
