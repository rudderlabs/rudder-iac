package showdefaultevents

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/testorchestrator"
)

func NewCmdShowDefaultEvents() *cobra.Command {
	var err error

	cmd := &cobra.Command{
		Use:   "show-default-events",
		Short: "Show default test events",
		Long: heredoc.Doc(`
			Displays the built-in default test events that are used when testing
			transformations without custom test inputs.

			These default events cover common RudderStack event types (track, identify,
			page, screen, group, alias) and can be used as a reference when creating
			custom test inputs for your transformations.
		`),
		Example: heredoc.Doc(`
			# Show all default test events
			$ rudder-cli transformations show-default-events
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			defer func() {
				telemetry.TrackCommand("transformations show-default-events", err)
			}()

			if err = testorchestrator.ShowDefaultEvents(); err != nil {
				return fmt.Errorf("showing default events: %w", err)
			}

			return nil
		},
	}

	return cmd
}
