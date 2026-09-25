package workspace

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

func NewCmdWorkspace() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Manage workspace resources",
		Long:  "Inspect the authenticated RudderStack workspace and list supported remote resource types without changing them.",
		Example: heredoc.Doc(`
			rudder-cli workspace info
			rudder-cli workspace tracking-plans list --json
		`),
	}

	cmd.AddCommand(NewCmdInfo())
	cmd.AddCommand(NewCmdAccounts())
	cmd.AddCommand(NewCmdRetlSource())
	cmd.AddCommand(NewCmdTrackingPlans())
	cmd.AddCommand(NewCmdEventStreamSources())

	return cmd
}
