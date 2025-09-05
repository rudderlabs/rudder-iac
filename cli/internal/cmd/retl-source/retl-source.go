package retlsource

import (
	"github.com/spf13/cobra"
)

func NewCmdRetlSources() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "retl-sources",
		Short: "Manage RETL sources",
		Long:  "Manage RETL sources in your RudderStack workspace",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newCmdPreview())
	cmd.AddCommand(newCmdValidate())

	return cmd
}
