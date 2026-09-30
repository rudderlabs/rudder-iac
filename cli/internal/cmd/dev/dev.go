// Package dev holds the hidden, experimental `dev` commands (DEX-1017). The
// server and client live in cli/pkg/exp/devlisten; this package is flag
// glue, discovery and printing.
package dev

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
)

var log = logger.New("dev")

var errDisabled = errors.New("dev commands are experimental: set RUDDERSTACK_CLI_EXPERIMENTAL=true and RUDDERSTACK_X_DEV_LISTEN=true")

// Deps are read at run time, after the root command loaded the config.
type Deps struct {
	// Enabled reports the devListen experimental flag.
	Enabled func() bool
	// DevURL is RUDDERSTACK_DEV_URL, bound in config.InitConfig.
	DevURL func() string
	// ConfigDir holds the state file; it follows -c.
	ConfigDir func() string
}

func NewCmdDev(deps Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Capture and inspect local SDK requests (experimental)",
		Long: "Run a local endpoint that accepts RudderStack SDK requests and keeps them for inspection.\n\n" +
			"Start with dev listen, point the SDK at its URL, then read what arrived with dev summary,\n" +
			"dev events list and dev requests show.",
		Example: "  rudder-cli dev listen --detach\n" +
			"  rudder-cli dev summary --since 0 --json\n" +
			"  rudder-cli dev stop",
		Hidden: true,
		// Machine mode owns stdout and stderr; each command prints its errors.
		SilenceUsage:  true,
		SilenceErrors: true,
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
	cmd.AddCommand(newCmdSummary(deps))
	cmd.AddCommand(newCmdInfo(deps))
	cmd.AddCommand(newCmdCursor(deps))
	cmd.AddCommand(newCmdReset(deps))
	cmd.AddCommand(newCmdStop(deps))
	cmd.AddCommand(newCmdSend(deps))
	cmd.AddCommand(newCmdExec())
	return cmd
}
