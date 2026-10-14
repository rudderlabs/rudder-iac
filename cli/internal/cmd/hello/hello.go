package hello

import "github.com/spf13/cobra"

func NewCmdHello() *cobra.Command {
	return &cobra.Command{
		Use:   "hello",
		Short: "Print a hello world greeting",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("Hello, World!")
		},
	}
}
