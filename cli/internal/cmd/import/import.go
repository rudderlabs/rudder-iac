package importcmd

import (
	"github.com/spf13/cobra"
)

func NewCmdImport() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <command>",
		Short: "Import remote resources to local configuration",
		Long:  "Import remote resources from various providers into local YAML configuration files",
	}

	cmd.AddCommand(NewCmdRetlSource())
	cmd.AddCommand(NewCmdWorkspaceImport())

	return cmd
}
