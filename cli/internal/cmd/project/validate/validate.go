package validate

import (
	"errors"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/renderer"
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
		deps       app.Deps
		p          project.Project
		location   string
		varFiles   []string
		jsonOutput bool
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
			$ rudder-cli validate --json
		`),
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			// With no workspace lookup there is no workspace ID, so
			// workspace-aware import-manifest rules check every workspace block
			// instead of only the one apply targets.
			deps, err = app.NewOfflineDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			projectOpts, err := app.NewProjectOptions(varFiles)
			if err != nil {
				return err
			}

			if jsonOutput {
				projectOpts = append(projectOpts, project.WithRenderer(
					renderer.NewJSONRenderer(cmd.OutOrStdout()),
				))
			}

			p = deps.NewProject(projectOpts...)
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			validateLog.Debug("validate", "location", location, "json", jsonOutput)

			defer func() {
				telemetry.TrackCommand(telemetry.CommandName(cmd), err, []telemetry.KV{
					{K: "location", V: location},
					{K: "json", V: jsonOutput},
				}...)
			}()

			err = run(cmd, p, location, jsonOutput)
			return err
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the directory containing the project files or a specific file")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output diagnostics as JSON")
	cmd.Flags().StringArrayVar(&varFiles, "var-file", nil, "Path to a variable file ending in .vars.yaml or .vars.yml (repeatable; later files take priority)")
	return cmd
}

// run loads and validates the project. Under --json stdout holds only the
// renderer's document, so the success line and the deprecation notice stay off
// it; the notice goes to stderr, where CI logs and agents still see it.
func run(cmd *cobra.Command, p project.Project, location string, jsonOutput bool) error {
	err := p.Load(location)
	if err != nil {
		wrapped := fmt.Errorf("validating project: %w", err)
		// A failure the document already holds is not printed again on stderr,
		// which would break consumers that merge the streams. What the document
		// lacks keeps its stderr message, such as the --var-file hint that
		// accompanies a substitution failure.
		if jsonOutput && errors.Is(err, project.ErrValidationFailed) {
			return &cmderrors.SilentError{Err: wrapped}
		}
		return wrapped
	}

	if project.HasLegacySpecs(p.Specs()) {
		if jsonOutput {
			fmt.Fprintln(cmd.ErrOrStderr(), ui.Warning(project.LegacySpecDeprecationWarning))
		} else {
			ui.PrintDeprecationWarning(project.LegacySpecDeprecationWarning)
		}
	}

	validateLog.Info("Project configuration is valid")
	if !jsonOutput {
		ui.PrintSuccess("Project configuration is valid")
	}
	return nil
}
