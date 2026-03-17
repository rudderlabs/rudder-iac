package validate

import (
	"context"
	"errors"
	"fmt"
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/validator"
)

var validateLog = logger.New("datagraph", logger.Attr{
	Key:   "cmd",
	Value: "validate",
})

func NewCmdValidate() *cobra.Command {
	var (
		deps       app.Deps
		p          project.Project
		err        error
		location   string
		all        bool
		modified   bool
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "validate [type] [id]",
		Short: "Validate data graph resources",
		Long: heredoc.Doc(`
			Validates data graph resources (models and relationships) against the warehouse.

			Checks include table existence, column existence, type compatibility, and more.
			You can validate all resources, only modified ones, or a specific resource by type and ID.
		`),
		Example: heredoc.Doc(`
			# Validate all resources
			$ rudder-cli data-graphs validate --all

			# Validate only modified resources
			$ rudder-cli data-graphs validate --modified

			# Validate a specific model
			$ rudder-cli data-graphs validate model my-model-id

			# Validate a specific relationship
			$ rudder-cli data-graphs validate relationship my-relationship-id

			# Output as JSON
			$ rudder-cli data-graphs validate --all --json
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.GetConfig()
			if !cfg.ExperimentalFlags.DataGraph {
				return fmt.Errorf("data-graphs commands require the experimental flag 'data_graph' to be enabled in your configuration")
			}

			if err := validateFlags(args, all, modified); err != nil {
				return err
			}

			deps, err = app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			p = deps.NewProject()

			if err := p.Load(location); err != nil {
				return fmt.Errorf("loading and validating project: %w", err)
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			defer func() {
				telemetry.TrackCommand("data-graphs validate", err, []telemetry.KV{
					{K: "location", V: location},
					{K: "all", V: all},
					{K: "modified", V: modified},
					{K: "json", V: jsonOutput},
				}...)
			}()

			validateLog.Debug("validate", "location", location, "all", all, "modified", modified, "json", jsonOutput)

			ctx := context.Background()

			workspace, err := deps.Client().Workspaces.GetByAuthToken(ctx)
			if err != nil {
				return fmt.Errorf("fetching workspace information: %w", err)
			}

			var mode validator.Mode
			if all {
				mode = validator.ModeAll{}
			} else if modified {
				mode = validator.ModeModified{}
			} else {
				mode = validator.ModeSingle{
					ResourceType: args[0],
					TargetID:     args[1],
				}
			}

			w := cmd.OutOrStdout()
			var displayFunc validator.DisplayFunc
			if jsonOutput {
				displayFunc = validator.NewJSONDisplayer(w).Display
			} else {
				displayFunc = validator.NewTerminalDisplayer(w).Display
			}

			err = validator.Validate(ctx, p, deps.Providers().DataGraph, validator.Config{
				Mode:        mode,
				WorkspaceID: workspace.ID,
				JSONOutput:  jsonOutput,
				Writer:      w,
				DisplayFunc: displayFunc,
				Concurrency: config.GetConfig().Concurrency.DataGraph,
			})
			if err != nil {
				if jsonOutput && errors.Is(err, validator.ErrValidationFailed) {
					return &cmderrors.SilentError{Err: err}
				}
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the directory containing the project files or a specific file")
	cmd.Flags().BoolVar(&all, "all", false, "Validate all data graph resources in the project")
	cmd.Flags().BoolVar(&modified, "modified", false, "Validate only new or modified data graph resources")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output results as JSON")

	return cmd
}

// validateFlags validates the command flags and arguments
func validateFlags(args []string, all, modified bool) error {
	modes := 0
	hasArgs := len(args) > 0
	if hasArgs {
		modes++
	}
	if all {
		modes++
	}
	if modified {
		modes++
	}

	if modes == 0 {
		return fmt.Errorf("must specify either <type> <id>, --all, or --modified")
	}
	if modes > 1 {
		return fmt.Errorf("cannot combine validation modes: specify only one of <type> <id>, --all, or --modified")
	}

	if hasArgs {
		if len(args) != 2 {
			return fmt.Errorf("expected exactly 2 arguments: <type> <id>, got %d", len(args))
		}

		resourceType := args[0]
		if resourceType != "model" && resourceType != "relationship" {
			return fmt.Errorf("invalid resource type %q: must be 'model' or 'relationship'", resourceType)
		}
	}

	return nil
}
