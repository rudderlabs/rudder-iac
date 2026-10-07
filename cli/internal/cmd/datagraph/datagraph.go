package datagraph

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	validateCmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/datagraph/validate"
)

func NewCmdDataGraph() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "data-graphs <command>",
		Short: "Manage data graphs",
		Long:  "Validate data graph models and relationships in a local project against their warehouse account. Use the validate subcommand to check all, modified, or one selected resource.",
		Example: heredoc.Doc(`
			$ rudder-cli data-graphs validate --all
			$ rudder-cli data-graphs validate --modified
			$ rudder-cli data-graphs validate model my-model-id
		`),
	}

	cmd.AddCommand(validateCmd.NewCmdValidate())

	return cmd
}
