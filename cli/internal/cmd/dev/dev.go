// Package dev holds the hidden, experimental `dev` commands (DEX-1017). The
// server and client live in cli/pkg/exp/devlisten; this package is flag
// glue and printing.
package dev

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
)

var log = logger.New("root", logger.Attr{Key: "cmd", Value: "dev"})

var errDisabled = errors.New("dev commands are experimental: run rudder-cli experimental enable devListen " +
	"(needs RUDDERSTACK_CLI_EXPERIMENTAL=true), or set RUDDERSTACK_CLI_EXPERIMENTAL=true and RUDDERSTACK_X_DEV_LISTEN=true")

// Deps are read at run time, after the root command loaded the config.
type Deps struct {
	// Enabled reports the devListen experimental flag.
	Enabled func() bool
	// DevURL is RUDDERSTACK_DEV_URL, bound in config.InitConfig.
	DevURL func() string
	// Track records one command run, as telemetry.TrackCommand does.
	Track func(command string, err error, extras ...telemetry.KV)
}

func NewCmdDev(deps Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dev <command>",
		Short:   "Capture and inspect local SDK requests (experimental)",
		Long:    devLong,
		Example: devExample,
		Args:    cobra.NoArgs,
		Hidden:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !deps.Enabled() {
				return errDisabled
			}
			return nil
		},
	}
	cmd.SetFlagErrorFunc(flagError)
	cmd.AddCommand(newCmdListen(deps))
	cmd.AddCommand(newCmdEvents(deps))
	cmd.AddCommand(newCmdRequests(deps))
	return cmd
}
