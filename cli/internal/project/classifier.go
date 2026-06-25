package project

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

type SpecLevel int

const (
	ResourceSpec SpecLevel = iota
	ProjectSpec
)

// classify reports whether a spec is resource-level or project-level. It lives
// in package project, not specs, so it can route by the owning provider's kind
// without making the spec data model depend on a provider.
func classify(s *specs.Spec) SpecLevel {
	if s.Kind == importmanifest.KindImportManifest {
		return ProjectSpec
	}
	return ResourceSpec
}
