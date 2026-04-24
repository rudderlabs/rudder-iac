package testsandbox

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewCmdTestSandbox() *cobra.Command {
	return &cobra.Command{
		Use:   "test-sandbox",
		Short: "Print a sandbox smoke-test message",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Hello from the sandbox using the new image!")
			return err
		},
	}
}
