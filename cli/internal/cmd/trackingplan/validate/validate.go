package validate

import (
	"github.com/spf13/cobra"
)

func NewCmdTPValidate() *cobra.Command {
	cmd := &cobra.Command{
		Use:        "validate",
		Short:      "Validate locally defined catalog",
		Deprecated: "use rudder-cli validate instead. This command will be removed in a future release.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	return cmd
}
