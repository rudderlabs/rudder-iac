package validate

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/telemetry"
	"github.com/spf13/cobra"
)

func NewCmdTPValidate() *cobra.Command {
	var (
		location string
		err      error
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate locally defined catalog",
		Long:  "Validate locally defined catalog",
		Example: heredoc.Doc(`
			$ rudder-cli tp validate --location <path-to-catalog-dir or file>
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			defer func() {
				telemetry.TrackCommand("tp validate", err)
			}()

			deps, err := app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			p := project.New(location, deps.Providers().DataCatalog)

			if err := p.Load(); err != nil {
				return fmt.Errorf("loading project: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", "", "Path to the directory containing the catalog files or catalog file itself")
	return cmd
}
