package transformations

import (
	"errors"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	testCmd "github.com/rudderlabs/rudder-iac/cli/internal/cmd/transformations/test"
)

func NewCmdTransformations() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transformations <command>",
		Short: "Manage transformations",
		Long:  "Manage the lifecycle of transformations and libraries",
		Hidden: true,
		Example: heredoc.Doc(`
			$ rudder-cli transformations test my-transformation-id
			$ rudder-cli transformations test --all
			$ rudder-cli transformations test --modified
		`),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !viper.GetBool("experimental") {
				return errors.New("experimental commands are disabled")
			}
			if !viper.GetBool("flags.transformations") {
				return errors.New("transformations command is disabled, enable it by running `rudder-cli experimental enable transformations`")
			}

			return nil
		},
	}

	cmd.AddCommand(testCmd.NewCmdTest())

	return cmd
}
