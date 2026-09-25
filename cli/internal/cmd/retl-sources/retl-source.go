package retlsource

import (
	"github.com/spf13/cobra"
)

func NewCmdRetlSources() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "retl-sources",
		Short:   "Manage RETL sources",
		Long:    "Validate or preview RETL SQL model source specs from a local project before applying them. Select a source by its external ID and use --location when the project is not in the current directory.",
		Example: "  rudder-cli retl-sources validate orders-model --location ./project\n  rudder-cli retl-sources preview orders-model --limit 5",
		Args:    cobra.NoArgs,
	}

	cmd.AddCommand(newCmdPreview())
	cmd.AddCommand(newCmdValidate())

	return cmd
}
