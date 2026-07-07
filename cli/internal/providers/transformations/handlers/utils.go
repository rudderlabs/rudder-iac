package handlers

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

const (
	// TransformationsDir is the directory where transformation specs and code files are stored
	TransformationsDir = "transformations"

	// Supported Languages
	JavaScript = "javascript"
	Python     = "python"

	// File Extensions
	ExtensionJS = ".js"
	ExtensionPY = ".py"
)

// toImportSpec creates a Spec with import metadata for a transformation resource.
func ToImportSpec(
	kind string,
	metadataName string,
	workspaceMetadata specs.WorkspaceImportMetadata,
	specData map[string]any,
) (*specs.Spec, error) {
	metadata := specs.Metadata{
		Name: metadataName,
		Import: &specs.WorkspacesImportMetadata{
			Workspaces: []specs.WorkspaceImportMetadata{workspaceMetadata},
		},
	}

	metadataMap, err := metadata.ToMap()
	if err != nil {
		return nil, fmt.Errorf("converting metadata to map: %w", err)
	}

	return &specs.Spec{
		Version:  specs.SpecVersionV1,
		Kind:     kind,
		Metadata: metadataMap,
		Spec:     specData,
	}, nil
}

// ImportEntriesFromWorkspace flattens a workspace's inline import resources into
// the import-manifest entries that travel alongside the exported spec.
func ImportEntriesFromWorkspace(workspaceMetadata specs.WorkspaceImportMetadata) []importmanifest.ImportEntry {
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
