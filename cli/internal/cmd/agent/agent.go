package agent

import "github.com/spf13/cobra"

func NewCmdAgent() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Prepare a project for a coding agent",
		// Hidden until the command name is settled. The name is an open decision.
		Hidden: true,
	}

	cmd.AddCommand(NewCmdInit())

	return cmd
}
