package apply

import (
	"errors"

	"github.com/spf13/cobra"
)

func NewCmdTPApply() *cobra.Command {
	cmd := &cobra.Command{
		Use:        "apply",
		Short:      "Apply the changes to upstream catalog",
		Deprecated: "use `rudder-cli apply` instead",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("tp apply is deprecated: use `rudder-cli apply` instead")
		},
	}

	return cmd
}
