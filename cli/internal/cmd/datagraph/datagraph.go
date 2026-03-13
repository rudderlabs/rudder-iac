package datagraph

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	validateCmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/datagraph/validate"
)

func NewCmdDataGraph() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "data-graph <command>",
		Short: "Manage data graphs",
		Long:  "Manage the lifecycle of data graph resources (models and relationships)",
		Example: heredoc.Doc(`
			$ rudder-cli data-graph validate --all
			$ rudder-cli data-graph validate --modified
			$ rudder-cli data-graph validate model my-model-id
		`),
	}

	cmd.AddCommand(validateCmd.NewCmdValidate())

	return cmd
}
