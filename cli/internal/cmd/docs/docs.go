package docs

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/docs/rules"
	"github.com/spf13/cobra"
)

// NewCmdDocs returns the `docs` command group — currently only `rules`,
// but designed to accommodate additional doc-generation subcommands later.
func NewCmdDocs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs <command>",
		Short: "Generate documentation artifacts from registered metadata",
	}
	cmd.AddCommand(rules.NewCmdRules())
	return cmd
}
