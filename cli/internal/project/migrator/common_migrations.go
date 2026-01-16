package migrator

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

// VersionUpdateMigration updates the spec version from legacy to v1
type VersionUpdateMigration struct{}

func (v VersionUpdateMigration) Name() string {
	return "version-update-from-0.1-to-v1"
}

func (v VersionUpdateMigration) Apply(spec *specs.Spec) error {
	// Update version from rudder/0.1 to rudder/v1
	if spec.IsLegacyVersion() {
		v := spec.Version
		spec.Version = specs.SpecVersionV1
		migratorLog.Debug("updated version", "from", v, "to", specs.SpecVersionV1)
	}
	return nil
}

// TODO: add migration to transform all references from path based to URN based