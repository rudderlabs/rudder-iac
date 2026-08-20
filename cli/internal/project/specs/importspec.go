package specs

import (
	"fmt"
)

// ToImportSpec creates a Spec carrying one workspace's import metadata, used by
// every provider that supports import to write the spec file that adopts remote
// resources into the project. The workspace metadata may name a single resource
// or many — event stream connections emit one spec covering every connection,
// while the other providers emit one spec per resource.
func ToImportSpec(
	kind string,
	metadataName string,
	workspaceMetadata WorkspaceImportMetadata,
	specData map[string]any,
) (*Spec, error) {
	metadata := Metadata{
		Name: metadataName,
		Import: &WorkspacesImportMetadata{
			Workspaces: []WorkspaceImportMetadata{workspaceMetadata},
		},
	}

	metadataMap, err := metadata.ToMap()
	if err != nil {
		return nil, fmt.Errorf("converting metadata to map: %w", err)
	}

	return &Spec{
		Version:  SpecVersionV1,
		Kind:     kind,
		Metadata: metadataMap,
		Spec:     specData,
	}, nil
}
