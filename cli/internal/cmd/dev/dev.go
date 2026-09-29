// Package dev holds the hidden, experimental `dev` commands (DEX-1017). The
// server and client live in cli/pkg/exp/devlisten; this package is flag glue.
package dev

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
)

var log = logger.New("dev")

var errDisabled = errors.New("dev commands are experimental: set RUDDERSTACK_CLI_EXPERIMENTAL=true and RUDDERSTACK_X_DEV_LISTEN=true")

// NewCmdDev builds the hidden `dev` tree. enabled reports the devListen
// experimental flag; it is read at run time, after config loads.
func NewCmdDev(enabled func() bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "dev",
		Short:  "Capture and inspect the events an app sends (experimental)",
		Hidden: true,
		// Under --json, stdout must hold JSON only; the root command prints errors.
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !enabled() {
				return errDisabled
			}
			return nil
		},
	}
	cmd.AddCommand(newCmdListen())
	cmd.AddCommand(newCmdEvents())
	return cmd
}
