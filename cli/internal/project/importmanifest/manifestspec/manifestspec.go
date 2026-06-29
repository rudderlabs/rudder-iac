// Package manifestspec holds the import-manifest spec vocabulary (its kind) and
// the decode of its spec.workspaces payload. It is a leaf imported by both the
// importmanifest provider and the manifest validation rules, so neither has to
// import the other (which would cycle).
package manifestspec

import (
	"fmt"

	"github.com/go-viper/mapstructure/v2"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

// KindImportManifest is the spec kind the import-manifest provider owns.
const KindImportManifest = "import-manifest"

// DecodeWorkspaces strict-decodes a manifest's spec.workspaces payload, rejecting
// unknown fields so malformed manifests surface as errors rather than silently
// dropped keys. Both the provider and the manifest rules decode through here.
func DecodeWorkspaces(spec map[string]any) ([]specs.WorkspaceImportMetadata, error) {
	var m specs.WorkspacesImportMetadata
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused: true,
		Result:      &m,
		TagName:     "yaml",
	})
	if err != nil {
		return nil, fmt.Errorf("building decoder: %w", err)
	}
	if err := decoder.Decode(spec); err != nil {
		return nil, fmt.Errorf("decoding spec: %w", err)
	}
	return m.Workspaces, nil
}
