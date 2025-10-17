package validate

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	validateLog = logger.New("root", logger.Attr{
		Key:   "cmd",
		Value: "validate",
	})
)

func NewCmdValidate() *cobra.Command {
	var (
		deps     app.Deps
		p        project.Project
		err      error
		location string
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate project configuration",
		Long: heredoc.Doc(`
			Validates the project configuration files for correctness and consistency.
			This includes checking for valid syntax, required fields, and relationships
			between resources.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli validate --location </path/to/dir or file>
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			deps, err = app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			p = project.New(location, deps.CompositeProvider())
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			validateLog.Debug("validate", "location", location)
			validateLog.Debug("validating project configuration")

			defer func() {
				telemetry.TrackCommand("validate", err, []telemetry.KV{
					{K: "location", V: location},
				}...)
			}()

			// Load and validate the project
			// The Load method internally calls the provider's Validate method
			if err := p.Load(); err != nil {
				return fmt.Errorf("validating project: %w", err)
			}

			validateLog.Info("Project configuration is valid")
			fmt.Println(ui.Color("✔", ui.Green), "Project configuration is valid")
			return nil
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the directory containing the project files or a specific file")
	return cmd
}
