package project

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

// LegacySpecDeprecationWarning is shown when any loaded spec uses the v0.1 format.
// Replace <migration-guide-url> when the upstream migration guide is published.
const LegacySpecDeprecationWarning = `v0.1 spec format is deprecated and will be removed in a future release.
         Run ` + "`rudder-cli migrate`" + ` to upgrade to v1.
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
