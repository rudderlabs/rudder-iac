package project

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

// LegacySpecDeprecationWarning is shown when any loaded spec uses the v0.1 format.
const LegacySpecDeprecationWarning = `v0.1 spec format is deprecated and will be removed in a future release.
         Run ` + "`rudder-cli migrate`" + ` to upgrade existing specs to v1.
         See: https://github.com/rudderlabs/rudder-iac/blob/main/BREAKING_CHANGES.md`

// ImportLegacySpecWarning is shown during import when the project has existing v0.1 specs.
// It informs users that newly imported resources will be in v1 format and nudges migration of existing specs.
const ImportLegacySpecWarning = `Imported resources will be generated in v1 spec format.
         v0.1 spec format is deprecated and will be removed in a future release.
         Run ` + "`rudder-cli migrate`" + ` to upgrade existing specs to v1.
         See: https://github.com/rudderlabs/rudder-iac/blob/main/BREAKING_CHANGES.md`

// HasLegacySpecs reports whether any spec in specMap uses a legacy (v0.1) version.
func HasLegacySpecs(specMap map[string]*specs.Spec) bool {
	for _, s := range specMap {
		if s != nil && s.IsLegacyVersion() {
			return true
		}
	}
	return false
}
