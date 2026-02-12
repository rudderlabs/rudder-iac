package migrator

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

// CommonMigration represents a migration that applies to all specs
// regardless of their kind or provider
type CommonMigration interface {
	// Name returns a descriptive name for the migration
	Name() string
	// Apply applies the migration to the spec
	Apply(spec *specs.Spec) error
}

// CommonMigrations is a collection of common migrations to apply
type CommonMigrations []CommonMigration

// GetCommonMigrations returns common migrations to apply BEFORE provider-specific migrations
func GetCommonMigrations() CommonMigrations {
	return CommonMigrations{
		VersionUpdateMigration{},
	}
}
