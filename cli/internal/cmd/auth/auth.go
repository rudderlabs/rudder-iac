package auth

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/auth"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/spf13/cobra"
)

func NewCmdAuth() *cobra.Command {

	var authCmd = &cobra.Command{
		Use:   "auth",
		Short: "Authentication commands",
	}

	var loginCmd = &cobra.Command{
		Use:   "login",
		Short: "Login with an access token",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			defer func() {
				telemetry.TrackCommand(telemetry.CommandName(cmd), err)
			}()

			return auth.Login()
		},
	}

	authCmd.AddCommand(loginCmd)

	return authCmd
}
