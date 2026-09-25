package transformations

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	showDefaultEventsCmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/transformations/showdefaultevents"
	testCmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/transformations/test"
)

func NewCmdTransformations() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transformations <command>",
		Short: "Manage transformations",
		Long:  "Test transformation specs and inspect the default event payloads used by local test suites. Use --all or --modified to select multiple project transformations.",
		Example: heredoc.Doc(`
			$ rudder-cli transformations test my-transformation-id
			$ rudder-cli transformations test --all
			$ rudder-cli transformations test --modified
		`),
	}

	cmd.AddCommand(testCmd.NewCmdTest())
	cmd.AddCommand(showDefaultEventsCmd.NewCmdShowDefaultEvents())

	return cmd
}
