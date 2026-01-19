package provider

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

const (
	MetadataNameProperties  = "properties"
	MetadataNameEvents      = "events"
	MetadataNameCategories  = "categories"
	MetadataNameCustomTypes = "custom-types"
)

func toImportSpec(
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
		Version:  specs.SpecVersionV0_1,
		Kind:     kind,
		Metadata: metadataMap,
		Spec:     data,
	}, nil
}
