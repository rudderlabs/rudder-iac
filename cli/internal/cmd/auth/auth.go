package auth

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/auth"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/spf13/cobra"
)

func NewCmdAuth() *cobra.Command {

	var authCmd = &cobra.Command{
		Use:     "auth",
		Short:   "Authentication commands",
		Long:    "Authenticate rudder-cli with RudderStack so workspace and project commands can call the Public API. Credentials are stored in the selected CLI configuration file.",
		Example: `  rudder-cli auth login`,
	}

	var loginCmd = &cobra.Command{
		Use:     "login",
		Short:   "Login with an access token",
		Long:    "Prompt for a RudderStack access token, verify it, and save it to the selected CLI configuration file for later workspace API requests.",
		Example: `  rudder-cli auth login`,
		Args:    cobra.NoArgs,
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
