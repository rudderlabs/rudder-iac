package migrate

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/formatter"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	migrateLog = logger.New("root", logger.Attr{
		Key:   "cmd",
		Value: "migrate",
	})
)

func NewCmdMigrate() *cobra.Command {
	var (
		deps        app.Deps
		err         error
		location    string
		confirm     bool
		loadedSpecs map[string]*specs.Spec
	)

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate project from spec rudder/0.1 to rudder/1",
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
			// Initialize dependencies
			deps, err = app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			// Validate project before migration
			fmt.Println("Validating project...")
			p := project.New(location, deps.CompositeProvider())
			if err := p.Load(); err != nil {
				return fmt.Errorf("loading and validating project: %w", err)
			}
			fmt.Println("✅ Project validation successful")
			fmt.Println()

			// Load specs for migration
			l := &loader.Loader{}
			loadedSpecs, err = l.Load(location)
			if err != nil {
				return fmt.Errorf("failed to load specs: %w", err)
			}

			// Display files to be migrated
			fmt.Println("The following files will be migrated:")
			for path, spec := range loadedSpecs {
				fmt.Printf("  - %s (kind: %s)\n", path, spec.Kind)
			}
			fmt.Println()

			// Get user confirmation
			if confirm {
				fmt.Println("⚠️  WARNING: This will modify your files in place!")
				proceed, err := ui.Confirm("Do you want to proceed with the migration?")
				if err != nil {
					return fmt.Errorf("failed to get confirmation: %w", err)
				}
				if !proceed {
					return fmt.Errorf("migration cancelled by user")
				}
				fmt.Println()
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			migrateLog.Debug("migrate", "location", location, "confirm", confirm)
			migrateLog.Info("migrating project spec from rudder/0.1 to rudder/1")

			// Migrate each spec and collect results
			migratedSpecs := make(map[string]*specs.Spec)
			for path, spec := range loadedSpecs {
				if spec.Version != "rudder/0.1" {
					return fmt.Errorf("file %s has an invalid version: %s, expected rudder/0.1", path, spec.Version)
				}
				fmt.Printf("Migrating file: %s (kind: %s)\n", path, spec.Kind)
				migrateLog.Info("migrating file", "path", path, "kind", spec.Kind)

				migratedSpec, err := deps.CompositeProvider().MigrateSpec(path, spec)
				if err != nil {
					return fmt.Errorf("migrating file %s: %w", path, err)
				}
				migratedSpecs[path] = migratedSpec
			}

			// Write migrated specs back to files
			fmt.Println("\nWriting migrated specs to files...")
			formatters := formatter.Setup(&formatter.YAMLFormatter{})
			for path, migratedSpec := range migratedSpecs {
				migrateLog.Info("writing migrated file", "path", path)
				entity := writer.FormattableEntity{
					Content:      migratedSpec,
					RelativePath: path,
				}
				if err := writer.OverwriteFile(formatters, entity); err != nil {
					return fmt.Errorf("writing file %s: %w", path, err)
				}
			}

			fmt.Println("\n✅ Migration completed successfully")
			migrateLog.Info("migration completed")
			return nil
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the directory containing the project files or a specific file")
	cmd.Flags().BoolVar(&confirm, "confirm", true, "Confirm migration before proceeding")

	return cmd
}
