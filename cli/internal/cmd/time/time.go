package timecmd

import (
	"fmt"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/spf13/cobra"
)

func NewCmdTime() *cobra.Command {
	return newCmdTime(time.Now)
}

func newCmdTime(now func() time.Time) *cobra.Command {
	return &cobra.Command{
		Use:   "time",
		Short: "Print the current local date and time",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			timestamp := ui.Color(now().Format(time.RFC3339), ui.ColorBlue)
			_, err := fmt.Fprintln(cmd.OutOrStdout(), timestamp)
			return err
		},
	}
}
