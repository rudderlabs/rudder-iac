package docs

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/docs/rules"
	"github.com/spf13/cobra"
)

func NewCmdDocs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Generate documentation artifacts (validation rules and related metadata)",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}
	cmd.AddCommand(rules.NewCmdRules())
	return cmd
}
