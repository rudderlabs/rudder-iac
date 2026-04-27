// Package importmanifest parses and emits the `import-manifest` spec kind,
// which holds URN → remote-ID mappings previously embedded inline in each
// imported resource spec under metadata.import. See docs/import-manifest-hld.md.
package importmanifest

import (
	"fmt"
	"sort"

	"github.com/go-viper/mapstructure/v2"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

// ImportEntry is one row in the aggregated manifest emitted during
// `rudder-cli import workspace`. Handlers produce these from FormatForExport
// in place of embedding metadata.import into every resource spec.
type ImportEntry struct {
	WorkspaceID string
	URN         string
	RemoteID    string
}

// parseWorkspaces strictly decodes `spec.workspaces` from a manifest spec.
// Unknown fields are rejected so typos surface as errors rather than being
// silently dropped.
func parseWorkspaces(s *specs.Spec) ([]specs.WorkspaceImportMetadata, error) {
	var payload struct {
		Workspaces []specs.WorkspaceImportMetadata `yaml:"workspaces"`
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused: true,
		TagName:     "yaml",
		Result:      &payload,
	})
	if err != nil {
		return nil, fmt.Errorf("creating manifest decoder: %w", err)
	}

	if err := decoder.Decode(s.Spec); err != nil {
		return nil, fmt.Errorf("decoding manifest spec: %w", err)
	}

	return payload.Workspaces, nil
}

// BuildSpec assembles a single well-formed import-manifest spec from a flat
// list of entries collected across providers during `import workspace`.
//
// Entries are grouped by WorkspaceID and sorted deterministically so repeated
// imports over identical remote state produce byte-identical YAML — committed
// manifests should not churn on re-import.
func BuildSpec(entries []ImportEntry) *specs.Spec {
	byWorkspace := make(map[string][]specs.ImportIds)
	for _, e := range entries {
		byWorkspace[e.WorkspaceID] = append(byWorkspace[e.WorkspaceID], specs.ImportIds{
			URN:      e.URN,
			RemoteID: e.RemoteID,
		})
	}

	workspaceIDs := make([]string, 0, len(byWorkspace))
	for id := range byWorkspace {
		workspaceIDs = append(workspaceIDs, id)
	}
	sort.Strings(workspaceIDs)

	workspaces := make([]specs.WorkspaceImportMetadata, 0, len(workspaceIDs))
	for _, id := range workspaceIDs {
		ids := byWorkspace[id]
		sort.Slice(ids, func(i, j int) bool { return ids[i].URN < ids[j].URN })
		workspaces = append(workspaces, specs.WorkspaceImportMetadata{
			WorkspaceID: id,
			Resources:   ids,
		})
	}

	return &specs.Spec{
		Version: specs.SpecVersionV1,
		Kind:    specs.KindImportManifest,
		Metadata: map[string]any{
			"name": "import-manifest",
		},
		Spec: map[string]any{
			"workspaces": workspaces,
		},
	}
}
