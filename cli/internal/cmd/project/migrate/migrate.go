package migrate

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/migrator"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema/editor"
	"github.com/spf13/cobra"
)

func NewCmdMigrate() *cobra.Command {
	var (
		deps           app.Deps
		err            error
		location       string
		confirm        bool
		proj           project.Project
		varFiles       []string
		schemaModeline bool
		schemaURLBase  string
	)

	cmd := &cobra.Command{
		Use:    "migrate",
		Short:  "Migrate project from spec rudder/0.1 to rudder/1",
		Hidden: true, // Hidden until ready for general availability
		Long: heredoc.Doc(`
			Migrates project configuration from spec rudder/0.1 to rudder/1.
			This command transforms your existing project files to the new spec version.
			
			⚠️  WARNING: This command modifies files in place. Commit or backup your
			changes before running this command.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli migrate --location </path/to/dir or file>
			$ rudder-cli migrate --location </path/to/dir or file> --confirm=false
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			configuredSchemaURLBase := config.GetConfig().SchemaBaseURL
			if !cmd.Flags().Changed("schema-url-base") && configuredSchemaURLBase != "" {
				schemaURLBase = configuredSchemaURLBase
			}
			if cmd.Flags().Changed("schema-url-base") || configuredSchemaURLBase != "" {
				schemaModeline = true
			}
			if schemaModeline {
				schemaURLBase = editor.URLBase(schemaURLBase)
			}

			// Initialize dependencies
			deps, err = app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			// Wire variable substitution so specs containing {{ .VAR }} placeholders
			// resolve before they are validated and migrated; otherwise migration of a
			// project that uses substitution would fail on the literal tokens.
			projectOpts, err := app.NewProjectOptions(varFiles)
			if err != nil {
				return err
			}

			// Validate project before migration
			proj = deps.NewProject(projectOpts...)
			if err := proj.Load(location); err != nil {
				return fmt.Errorf("loading and validating project: %w", err)
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			defer func() {
				telemetry.TrackCommand("migrate", err, migrateTelemetryExtras(location, confirm)...)
			}()

			migratorOpts := make([]migrator.Option, 0, 1)
			if schemaModeline {
				migratorOpts = append(migratorOpts, migrator.WithSchemaModeline(schemaURLBase))
			}
			m := migrator.New(proj, deps.CompositeProvider(), migratorOpts...)
			err = m.Migrate(confirm)
			return err
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the directory containing the project files or a specific file")
	cmd.Flags().BoolVar(&confirm, "confirm", true, "Confirm migration before proceeding")
	cmd.Flags().StringArrayVar(&varFiles, "var-file", nil, "Path to a variable file ending in .vars.yaml or .vars.yml (repeatable; later files take priority)")
	cmd.Flags().BoolVar(&schemaModeline, "schema-modeline", false, "Add yaml-language-server schema modelines to migrated specs")
	cmd.Flags().StringVar(&schemaURLBase, "schema-url-base", "", "URL root for schemas (implies --schema-modeline)")

	return cmd
}

// migrateTelemetryExtras returns TrackCommand key-values for migrate (fixed from/to spec versions for this path).
func migrateTelemetryExtras(location string, confirm bool) []telemetry.KV {
	return []telemetry.KV{
		{K: "location", V: location},
		{K: "confirm", V: confirm},
		{K: "from_version", V: specs.SpecVersionV0_1},
		{K: "to_version", V: specs.SpecVersionV1},
	}
}
