package columnmetadata

import (
	"github.com/spf13/cobra"
)

func NewCmdColumnMetadata() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "column-metadata",
		Short: "Manage column display name aliases on data graph models",
		Long:  "List, set, and clear display name aliases for warehouse columns on a data graph model.",
	}

	cmd.AddCommand(newCmdList())
	cmd.AddCommand(newCmdSet())
	cmd.AddCommand(newCmdClear())

	return cmd
}
