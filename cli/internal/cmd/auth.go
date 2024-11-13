package cmd

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/auth"
	"github.com/spf13/cobra"
)

func init() {
	authCmd.AddCommand(loginCmd)
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login with an access token",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		auth.Login()
	},
}
