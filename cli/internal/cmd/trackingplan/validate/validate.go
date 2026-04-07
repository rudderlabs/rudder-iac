package validate

import (
	"errors"

	"github.com/spf13/cobra"
)

func NewCmdTPValidate() *cobra.Command {
	cmd := &cobra.Command{
		Use:        "validate",
		Short:      "Validate locally defined catalog",
		Deprecated: "use `rudder-cli validate` instead",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("tp validate is deprecated: use `rudder-cli validate` instead")
		},
	}

	return cmd
}
