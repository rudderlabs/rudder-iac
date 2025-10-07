package provider

import (
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

const (
	SpecVersion = "rudder/v0.1"
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
	workspaceMetadata importremote.WorkspaceImportMetadata,
	data map[string]any,
) (*specs.Spec, error) {
	metadata := importremote.Metadata{
		Name: metadataName,
		Import: importremote.WorkspacesImportMetadata{
			Workspaces: []importremote.WorkspaceImportMetadata{workspaceMetadata},
		},
	}

	metadataMap := make(map[string]any)
	err := mapstructure.Decode(metadata, &metadataMap)
	if err != nil {
		return nil, fmt.Errorf("decoding metadata: %w", err)
	}

	return &specs.Spec{
		Version:  SpecVersion,
		Kind:     kind,
		Metadata: metadataMap,
		Spec:     data,
	}, nil
}
