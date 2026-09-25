package importcmd

import (
	"github.com/spf13/cobra"
)

func NewCmdImport() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "import <command>",
		Short:   "Import remote resources to local configuration",
		Long:    "Export existing RudderStack resources into local YAML specs that can be managed as code. Import a complete synchronized workspace or select a supported resource-specific workflow.",
		Example: "  rudder-cli import workspace --location ./project\n  rudder-cli import retl-sources --local-id orders --remote-id 2abc123",
	}

	cmd.AddCommand(NewCmdRetlSource())
	cmd.AddCommand(NewCmdWorkspaceImport())

	return cmd
}
