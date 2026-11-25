package reseed

import (
	"errors"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importer"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

// NewCmdReseed builds the reseed command: write a whole workspace as a project
// into an empty directory.
//
// It sits beside apply, validate and destroy rather than under import because
// it takes a directory the way they do and shares none of import's flags: with
// no project to reconcile, the sync guard, --merge and --skip-existing have
// nothing to act on.
func NewCmdReseed() *cobra.Command {
	var (
		deps     app.Deps
		location string
	)

	cmd := &cobra.Command{
		Use:   "reseed",
		Short: "Write a whole workspace as a project into an empty directory",
		Long: heredoc.Doc(`
			Write every resource in the workspace as specs in an empty directory.

			Resources the CLI already manages keep their externalId as the spec id,
			so a following apply matches them instead of recreating them. Resources
			it does not manage get specs plus an import-manifest.yaml, so apply
			adopts them. A workspace holding both reseeds in one pass, and
			'apply --dry-run' on the result reports no changes.

			The directory must hold no project YAML. That is what lets reseed skip
			the checks a plain import runs: there is nothing local to lose and
			nothing to reconcile. Reseed recovers what the specs say, not how they
			were written, so file layout, comments and variables are lost and
			secret values come back as '{{ .VAR }}' placeholders to fill in.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli reseed --location </path/to/empty_dir>
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			flags := config.GetConfig().ExperimentalFlags
			if !flags.ImportManaged {
				return fmt.Errorf("reseed requires the %q experimental flag to be enabled", "importManaged")
			}
			// Unmanaged resources are adopted through an import-manifest, and
			// importMerge is what makes that kind load. Without it a reseeded
			// project holding one fails to load.
			if !flags.ImportMerge {
				return fmt.Errorf("reseed requires the %q experimental flag to be enabled, so apply recognises the import-manifest it writes", "importMerge")
			}

			var err error
			deps, err = app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			defer func() {
				telemetry.TrackCommand(telemetry.CommandName(cmd), err)
			}()

			p, ok := deps.CompositeProvider().(importer.ReseedProvider)
			if !ok {
				return errors.New("the configured providers cannot reseed a workspace")
			}

			ui.StartSpinner("Reseeding ...")
			err = importer.Reseed(cmd.Context(), location, p)
			ui.StopSpinner()
			if err != nil {
				return err
			}

			// Continuation lines are indented to align under the text after "Warning: ".
			ui.PrintWarning(`Secret values are not reseeded from the workspace.
         The specs reference them as '{{ .VAR }}' placeholders; fill the
         generated var file in and pass it to apply with --var-file.`)
			ui.PrintSuccess("Done")
			return nil
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the directory to write the project into. It must hold no project YAML")
	return cmd
}
