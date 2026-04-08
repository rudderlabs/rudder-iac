package project

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

// LegacySpecDeprecationWarning is shown when any loaded spec uses the v0.1 format.
// Replace <migration-guide-url> when the upstream migration guide is published.
const LegacySpecDeprecationWarning = `v0.1 spec format is deprecated and will be removed in a future release.
         Run ` + "`rudder-cli migrate`" + ` to upgrade to v1.
         See: <migration-guide-url>`

// HasLegacySpecs reports whether any spec in specMap uses a legacy (v0.1) version.
func HasLegacySpecs(specMap map[string]*specs.Spec) bool {
	for _, s := range specMap {
		if s != nil && s.IsLegacyVersion() {
			return true
		}
	}
	return false
}

// PrintLegacySpecDeprecationIfNeeded prints a deprecation warning once per run when specMap
// contains any v0.1 spec. It is a no-op when there are no legacy specs.
func PrintLegacySpecDeprecationIfNeeded(specMap map[string]*specs.Spec) {
	if !HasLegacySpecs(specMap) {
		return
	}
	ui.PrintDeprecationWarning(LegacySpecDeprecationWarning)
}
