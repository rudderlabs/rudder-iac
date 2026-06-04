package docs

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/docs/rules"
	"github.com/spf13/cobra"
)

// NewCmdDocs returns the parent `docs` command group, grouping documentation
// generation subcommands such as `rules`.
func NewCmdDocs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Generate documentation artifacts",
		Long: heredoc.Doc(`
			Generate documentation artifacts derived from the CLI itself, such as the
			validation rule catalog consumed by docs sites and tooling.
		`),
		Args: cobra.NoArgs,
	}

	cmd.AddCommand(rules.NewCmdRules())

	return cmd
}
