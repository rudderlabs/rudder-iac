package testsandbox

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/spf13/cobra"
)

func NewCmdTestSandbox() *cobra.Command {
	var err error

	cmd := &cobra.Command{
		Use:   "test-sandbox",
		Short: "Print a sandbox test message",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			defer func() {
				telemetry.TrackCommand("test-sandbox", err)
			}()

			_, err = fmt.Fprintln(cmd.OutOrStdout(), "Hello from the sandbox!")
			if err != nil {
				return fmt.Errorf("writing sandbox message: %w", err)
			}

			return nil
		},
	}

	return cmd
}
