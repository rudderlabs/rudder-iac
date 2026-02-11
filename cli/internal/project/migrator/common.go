package migrator

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
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

// MigrateImportMetadataToURN converts LocalID-based import metadata to URN format
func MigrateImportMetadataToURN(spec *specs.Spec, resourceType string) error {
	metadata, err := spec.CommonMetadata()
	if err != nil {
		return err
	}

	if metadata.Import == nil {
		return nil
	}

	for wi, workspace := range metadata.Import.Workspaces {
		for ri, resource := range workspace.Resources {
			// Skip if already using URN
			if resource.URN != "" {
				continue
			}

			// Convert LocalID to URN
			if resource.LocalID != "" {
				urn := resources.URN(resource.LocalID, resourceType)
				metadata.Import.Workspaces[wi].Resources[ri].URN = urn
				// Clear LocalID for clean migration
				metadata.Import.Workspaces[wi].Resources[ri].LocalID = ""
			}
		}
	}

	// Update spec metadata
	metadataMap, err := metadata.ToMap()
	if err != nil {
		return err
	}
	spec.Metadata = metadataMap

	return nil
}
