package importmanifest

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest/manifestspec"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

// parseWorkspaces strict-decodes spec.workspaces from a full spec, delegating to
// the manifestspec leaf so the provider and the validation rules decode the same way.
func parseWorkspaces(s *specs.Spec) ([]specs.WorkspaceImportMetadata, error) {
	return manifestspec.DecodeWorkspaces(s.Spec)
}
