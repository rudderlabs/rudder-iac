// Package docs exposes the CLI's own generated documentation artifacts.
//
// Unlike the rest of the command tree these commands describe rudder-cli rather
// than a workspace: they make no API calls and need no credentials, so they stay
// usable in CI, in a container, and from an agent that has not authenticated.
package docs

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

func NewCmdDocs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Export rudder-cli's generated documentation artifacts",
		Long: heredoc.Doc(`
			Exports machine-readable documentation about rudder-cli itself.

			These commands run offline and unauthenticated — they describe the CLI,
			not a workspace.
		`),
	}

	cmd.AddCommand(newCmdExportValidationRules())
	return cmd
}
