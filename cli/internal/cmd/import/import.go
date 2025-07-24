package importcmd

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

func NewCmdImport() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <command>",
		Short: "Import remote resources to local configuration",
		Long:  "Import remote resources from various providers into local YAML configuration files",
		Example: heredoc.Doc(`
			$ rudder-cli import retl-source --local-id my-model --remote-id abc123 --workspace-id ws456
		`),
	}

	cmd.AddCommand(NewCmdRetlSource())

	return cmd
}
