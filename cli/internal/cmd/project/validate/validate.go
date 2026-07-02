package validate

import (
	"context"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/renderer"
	"github.com/spf13/cobra"
)

const (
	formatText = "text"
	formatJSON = "json"
)

var (
	validateLog = logger.New("root", logger.Attr{
		Key:   "cmd",
		Value: "validate",
	})
)

func NewCmdValidate() *cobra.Command {
	var (
		deps      app.Deps
		p         project.Project
		workspace *client.Workspace
		err       error
		location  string
		varFiles  []string
		format    string
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate project configuration",
		Long: heredoc.Doc(`
			Validates the project configuration files for correctness and consistency.
			This includes checking for valid syntax, required fields, and relationships
			between resources.

			Use --format json to emit machine-readable diagnostics with stable error
			codes and source positions (file, line, column), suitable for editors,
			CI, and other tooling.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli validate --location </path/to/dir or file>
			$ rudder-cli validate --format json
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if format != formatText && format != formatJSON {
				return fmt.Errorf("invalid --format %q: must be one of %q, %q", format, formatText, formatJSON)
			}

			deps, err = app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			// Resolve the active workspace so validation scopes workspace-aware
			// rules (e.g. import-manifest orphaned-urn) to the same workspace apply
			// targets.
			workspace, err = deps.Client().Workspaces.GetByAuthToken(context.Background())
			if err != nil {
				return fmt.Errorf("fetching workspace information: %w", err)
			}

			projectOpts, err := app.NewProjectOptions(config.GetConfig(), varFiles)
			if err != nil {
				return err
			}
			projectOpts = append(projectOpts, project.WithWorkspaceID(workspace.ID))

			// In JSON mode, diagnostics are rendered as a machine-readable array to
			// stdout; the human-readable text renderer stays the default.
			if format == formatJSON {
				projectOpts = append(projectOpts, project.WithRenderer(renderer.NewJSONRenderer(cmd.OutOrStdout())))
			}

			p = deps.NewProject(projectOpts...)
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			validateLog.Debug("validate", "location", location, "format", format)

			defer func() {
				telemetry.TrackCommand("validate", err, []telemetry.KV{
					{K: "location", V: location},
					{K: "format", V: format},
				}...)
			}()

			// Load and validate the project (validation engine + RuleProvider rules).
			// The configured renderer emits diagnostics; on failure we return a
			// SilentError in JSON mode so cobra doesn't also write a message to
			// stderr and disturb the machine-readable output on stdout.
			if err = p.Load(location); err != nil {
				if format == formatJSON {
					return &cmderrors.SilentError{Err: err}
				}
				return fmt.Errorf("validating project: %w", err)
			}

			// Human-readable extras are suppressed in JSON mode so stdout stays a
			// single, parseable diagnostics document.
			if format == formatJSON {
				return nil
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
	cmd.Flags().StringVar(&format, "format", formatText, "Output format for validation results: text or json")
	return cmd
}
