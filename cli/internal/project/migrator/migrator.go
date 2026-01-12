package migrator

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/formatter"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

var (
	migratorLog = logger.New("migrator")
)

// Migrator handles the migration of project specs from rudder/0.1 to rudder/1
type Migrator struct {
	project  project.Project
	provider provider.Provider
}

// New creates a new Migrator instance
func New(proj project.Project, p provider.Provider) *Migrator {
	return &Migrator{
		project:  proj,
		provider: p,
	}
}

// DisplayFilesToMigrate shows the list of files that will be migrated in a table
func (m *Migrator) DisplayFilesToMigrate(loadedSpecs map[string]*specs.Spec) {
	columns := []table.Column{
		{Title: "File", Width: 80},
		{Title: "Current Version", Width: 20},
		{Title: "Needs Migration?", Width: 20},
	}

	rows := make([]table.Row, 0, len(loadedSpecs))
	for path, spec := range loadedSpecs {
		willMigrate := "Yes"
		if spec.Version != specs.SpecVersionV0_1 {
			willMigrate = "No (already rudder/1)"
		}
		rows = append(rows, table.Row{path, spec.Version, willMigrate})
	}

	fmt.Println("The following files will be processed:")
	ui.PrintTable(columns, rows)
	fmt.Println()
}

// ConfirmMigration asks the user to confirm the migration
func (m *Migrator) ConfirmMigration() (bool, error) {
	fmt.Println("⚠️  WARNING: This will modify your files in place!")
	proceed, err := ui.Confirm("Do you want to proceed with the migration?")
	if err != nil {
		return false, fmt.Errorf("failed to get confirmation: %w", err)
	}
	if !proceed {
		return false, nil
	}
	fmt.Println()
	return true, nil
}

// MigrateSpecs migrates all loaded specs to rudder/1
func (m *Migrator) MigrateSpecs(loadedSpecs map[string]*specs.Spec) (map[string]*specs.Spec, error) {
	migratedSpecs := make(map[string]*specs.Spec)
	for path, spec := range loadedSpecs {
		fmt.Printf("Migrating file: %s (kind: %s)\n", path, spec.Kind)
		migratorLog.Info("migrating file", "path", path, "kind", spec.Kind)

		migratedSpec, err := m.provider.MigrateSpec(spec)
		if err != nil {
			return nil, fmt.Errorf("migrating file %s: %w", path, err)
		}
		migratedSpecs[path] = migratedSpec
	}
	return migratedSpecs, nil
}

// WriteSpecs writes the migrated specs back to files
func (m *Migrator) WriteSpecs(migratedSpecs map[string]*specs.Spec) error {
	fmt.Println("\nWriting migrated specs to files...")
	formatters := formatter.Setup(&formatter.YAMLFormatter{})
	for path, migratedSpec := range migratedSpecs {
		migratorLog.Info("writing migrated file", "path", path)
		entity := writer.FormattableEntity{
			Content:      migratedSpec,
			RelativePath: path,
		}
		if err := writer.OverwriteFile(formatters, entity); err != nil {
			return fmt.Errorf("writing file %s: %w", path, err)
		}
	}
	return nil
}

// Migrate performs the complete migration process
func (m *Migrator) Migrate(confirm bool) error {
	migratorLog.Debug("migrate", "location", m.project.Location(), "confirm", confirm)
	migratorLog.Info("migrating project spec from rudder/0.1 to rudder/1")

	// Load specs for migration
	loadedSpecs := m.project.Specs()

	// Display files to be migrated
	m.DisplayFilesToMigrate(loadedSpecs)

	// Get user confirmation
	if confirm {
		proceed, err := m.ConfirmMigration()
		if err != nil {
			return err
		}
		if !proceed {
			return fmt.Errorf("migration cancelled by user")
		}
	}

	// Migrate specs
	migratedSpecs, err := m.MigrateSpecs(loadedSpecs)
	if err != nil {
		return err
	}

	// Write migrated specs back to files
	if err := m.WriteSpecs(migratedSpecs); err != nil {
		return err
	}

	fmt.Println("\n✅ Migration completed successfully")
	migratorLog.Info("migration completed")
	return nil
}
