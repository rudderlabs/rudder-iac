package provider

import (
	"fmt"

	"github.com/go-viper/mapstructure/v2"
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
		Import: specs.WorkspacesImportMetadata{
			Workspaces: []specs.WorkspaceImportMetadata{workspaceMetadata},
		},
	}

	metadataMap := make(map[string]any)
	err := mapstructure.Decode(metadata, &metadataMap)
	if err != nil {
		return nil, fmt.Errorf("decoding metadata: %w", err)
	}

	return &specs.Spec{
		Version:  specs.SpecVersion,
		Kind:     kind,
		Metadata: metadataMap,
		Spec:     data,
	}, nil
}
