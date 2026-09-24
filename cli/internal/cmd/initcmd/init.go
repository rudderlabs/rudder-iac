package initcmd

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/spf13/cobra"
)

var log = logger.New("init")

func NewCmdInit() *cobra.Command {
	var location string

	cmd := &cobra.Command{
		Use:   "init [provider...]",
		Short: "Write a whole workspace into a new project",
		Long: heredoc.Doc(`
			Write every resource in the workspace — those already managed by the CLI
			as well as those that are not — into a new project directory as YAML specs.

			This is the command for starting from a workspace rather than from a
			project: a fresh checkout, a new sandbox, or any machine that has
			credentials for a workspace but not the specs describing it. Applying an
			untouched project written by init reports no changes.

			The target directory must not already contain spec files. To bring remote
			resources into a project you already have, use 'rudder-cli import workspace'.

			Naming one or more providers limits the clone to those providers.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli init --location ./my-project
			$ rudder-cli init eventstream destination --location ./my-project
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			defer func() {
				telemetry.TrackCommand("init", err, []telemetry.KV{
					{K: "location", V: location},
					{K: "providers", V: fmt.Sprint(args)},
				}...)
			}()

			log.Debug("cloning workspace", "location", location, "providers", args)

			deps, err := app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			p, err := scopedProvider(deps.CompositeProvider(), args)
			if err != nil {
				return err
			}

			spinner := ui.NewSpinner("Reading workspace ...")
			spinner.Start()

			err = importer.WorkspaceInit(cmd.Context(), location, p)

			spinner.Stop()
			if err != nil {
				return err
			}

			// Continuation lines are indented to align under the text after "Warning: ".
			ui.PrintWarning(`Secrets are not readable from the remote workspace.
         Add any required secret fields to the written specs using
         variable substitution ({{ .VAR }}), and pass them with
         --var-file to apply.`)
			ui.PrintSuccess(fmt.Sprintf("Workspace written to %s", location))

			return nil
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".", "Directory to write the project into")
	return cmd
}

// scopedProvider narrows the composite provider to the named providers. With no
// names the whole composite is used, which is the point of a plain init.
func scopedProvider(composite provider.Provider, names []string) (importer.InitProvider, error) {
	if len(names) == 0 {
		return composite, nil
	}

	cp, ok := composite.(*provider.CompositeProvider)
	if !ok {
		return nil, fmt.Errorf("naming providers is not supported by a %T", composite)
	}

	return cp.Subset(names)
}
