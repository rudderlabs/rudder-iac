package helloworld

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/spf13/cobra"
)

func NewCmdHelloWorld() *cobra.Command {
	return &cobra.Command{
		Use:   "hello-world",
		Short: "Print a friendly greeting",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			if _, err := fmt.Fprintln(w, ui.Color("Debug", ui.ColorYellow)); err != nil {
				return err
			}

			_, err := fmt.Fprintln(w, "Hello, World!")
			return err
		},
	}
}
