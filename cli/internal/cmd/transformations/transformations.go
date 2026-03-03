package transformations

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	testCmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/transformations/test"
)

func NewCmdTransformations() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transformations <command>",
		Short: "Manage transformations",
		Long:  "Manage the lifecycle of transformations and libraries",
		Example: heredoc.Doc(`
			$ rudder-cli transformations test my-transformation-id
			$ rudder-cli transformations test --all
			$ rudder-cli transformations test --modified
		`),
	}

	cmd.AddCommand(testCmd.NewCmdTest())

	return cmd
}
