package helloworld

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

func NewCmdHelloWorld() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hello-world",
		Short: "Print a hello world message",
		Long: heredoc.Doc(`
			Prints a hello world message.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli hello-world
		`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Hello, world!")
			return err
		},
	}

	return cmd
}
