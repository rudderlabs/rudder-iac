package hello

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewCmdHello() *cobra.Command {
	return &cobra.Command{
		Use:   "hello",
		Short: "Print a hello world greeting",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Hello, World!")
			return err
		},
	}
}
