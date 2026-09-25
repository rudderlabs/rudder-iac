package importcmd

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

func NewCmdImport() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <command>",
		Short: "Import remote resources to local configuration",
		Long:  "Export supported resources from the authenticated workspace into local declarative YAML project files.",
		Example: heredoc.Doc(`
			rudder-cli import workspace --location ./project
		`),
	}

	cmd.AddCommand(NewCmdRetlSource())
	cmd.AddCommand(NewCmdWorkspaceImport())

	return cmd
}
