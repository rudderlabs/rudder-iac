package helloworld

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewCmdHelloWorld() *cobra.Command {
	return &cobra.Command{
		Use:   "hello-world",
		Short: "Print a friendly greeting",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Hello, World!")
			return err
		},
	}
}
