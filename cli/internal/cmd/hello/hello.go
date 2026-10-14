package hello

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewCmdHello() *cobra.Command {
	return &cobra.Command{
		Use:   "hello",
		Short: "Print a hello-world greeting",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), "Hello Foo!")
		},
	}
}
