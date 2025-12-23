// Package lsp provides the LSP server command
package lsp

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/lsp/server"
)

// NewCmdLSP creates the LSP command
func NewCmdLSP() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lsp",
		Short: "Start the Language Server Protocol server",
		Long:  "Start the LSP server for Rudder CLI YAML file validation and IDE integration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLSP(cmd.Context())
		},
	}

	return cmd
}

// runLSP starts the LSP server
func runLSP(ctx context.Context) error {
	rudderServer := server.NewRudderLSPServer()

	// Run the server with stdio communication
	if err := rudderServer.RunStdio(ctx); err != nil {
		return fmt.Errorf("failed to run LSP server: %w", err)
	}

	return nil
}
