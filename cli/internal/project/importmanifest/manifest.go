package importmanifest

import (
	"fmt"

	"github.com/go-viper/mapstructure/v2"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

// parseWorkspaces strict-decodes spec.workspaces, rejecting unknown fields so
// malformed manifests surface as errors rather than silently dropped keys.
func parseWorkspaces(s *specs.Spec) ([]specs.WorkspaceImportMetadata, error) {
	var m specs.WorkspacesImportMetadata
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused: true,
		Result:      &m,
		TagName:     "yaml",
	})
	if err != nil {
		return nil, fmt.Errorf("building decoder: %w", err)
	}
	if err := decoder.Decode(s.Spec); err != nil {
		return nil, fmt.Errorf("decoding spec: %w", err)
	}
	return m.Workspaces, nil
}
