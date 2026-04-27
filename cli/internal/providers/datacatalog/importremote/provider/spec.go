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

// toImportSpec builds a clean data-catalog spec. Import metadata no longer
// lives inline on the spec — callers pair each spec with ImportEntry rows
// that feed the aggregated import-manifest.yaml emitted by the importer.
func toImportSpec(
	version string,
	kind string,
	metadataName string,
	data map[string]any,
) (*specs.Spec, error) {
	metadata := specs.Metadata{
		Name: metadataName,
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
