package provider

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

const (
	MetadataNameProperties  = "properties"
	MetadataNameEvents      = "events"
	MetadataNameCategories  = "categories"
	MetadataNameCustomTypes = "custom-types"
)

func toImportSpec(
	version string,
	kind string,
	metadataName string,
	workspaceMetadata specs.WorkspaceImportMetadata,
	data map[string]any,
) (*specs.Spec, error) {
	metadata := specs.Metadata{
		Name: metadataName,
		Import: &specs.WorkspacesImportMetadata{
			Workspaces: []specs.WorkspaceImportMetadata{workspaceMetadata},
		},
	}

	metadataMap, err := metadata.ToMap()
	if err != nil {
		return nil, err
	}

	return &specs.Spec{
		Version:  version,
		Kind:     kind,
		Metadata: metadataMap,
		Spec:     data,
	}, nil
}

// importEntriesFromWorkspace flattens a workspace's inline import resources into
// the import-manifest entries that travel alongside the exported spec.
func importEntriesFromWorkspace(workspaceMetadata specs.WorkspaceImportMetadata) []importmanifest.ImportEntry {
	entries := make([]importmanifest.ImportEntry, 0, len(workspaceMetadata.Resources))
	for _, r := range workspaceMetadata.Resources {
		entries = append(entries, importmanifest.ImportEntry{
			WorkspaceID: workspaceMetadata.WorkspaceID,
			URN:         r.URN,
			RemoteID:    r.RemoteID,
		})
	}
	return entries
}
