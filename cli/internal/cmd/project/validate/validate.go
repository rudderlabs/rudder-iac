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
		varFiles []string
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
			// Validation is local-only by design so it can run in CI without
			// credentials: no token check and no workspace lookup. Without a
			// workspace ID, workspace-aware import-manifest rules check every
			// workspace in the manifest instead of only the one apply targets.
			deps, err = app.NewOfflineDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			projectOpts, err := app.NewProjectOptions(varFiles)
			if err != nil {
				return err
			}

			p = deps.NewProject(projectOpts...)
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			validateLog.Debug("validate", "location", location)

			defer func() {
				telemetry.TrackCommand("validate", err, []telemetry.KV{
					{K: "location", V: location},
				}...)
			}()

			// Load and validate the project (validation engine + RuleProvider rules)
			if err := p.Load(location); err != nil {
				return fmt.Errorf("validating project: %w", err)
			}

			if project.HasLegacySpecs(p.Specs()) {
				ui.PrintDeprecationWarning(project.LegacySpecDeprecationWarning)
			}

			validateLog.Info("Project configuration is valid")
			ui.PrintSuccess("Project configuration is valid")
			return nil
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the directory containing the project files or a specific file")
	cmd.Flags().StringArrayVar(&varFiles, "var-file", nil, "Path to a variable file ending in .vars.yaml or .vars.yml (repeatable; later files take priority)")
	return cmd
}
