package apply

import (
	"github.com/spf13/cobra"
)

func NewCmdTPDestroy() *cobra.Command {
	cmd := &cobra.Command{
		Use:        "destroy",
		Short:      "Delete all resources from both the upstream catalog and state",
		Deprecated: "use rudder-cli destroy instead. This command will be removed in a future release.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	return cmd
}
