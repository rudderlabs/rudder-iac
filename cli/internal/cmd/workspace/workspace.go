package workspace

import (
	"github.com/spf13/cobra"
)

func NewCmdWorkspace() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workspace",
		Short:   "Inspect workspace resources",
		Long:    "Inspect the authenticated RudderStack workspace and list supported remote resource types without changing them. Listing commands use table output by default and support --json for automation.",
		Example: "  rudder-cli workspace info\n  rudder-cli workspace event-stream-sources list --json",
	}

	cmd.AddCommand(NewCmdInfo())
	cmd.AddCommand(NewCmdAccounts())
	cmd.AddCommand(NewCmdRetlSource())
	cmd.AddCommand(NewCmdTrackingPlans())
	cmd.AddCommand(NewCmdEventStreamSources())

	return cmd
}
