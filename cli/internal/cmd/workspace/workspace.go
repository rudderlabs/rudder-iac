package workspace

import (
	"github.com/spf13/cobra"
)

func NewCmdWorkspace() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Manage workspace resources",
	}

	cmd.AddCommand(NewCmdAccounts())
	cmd.AddCommand(NewCmdRetlSource())

	return cmd
}
