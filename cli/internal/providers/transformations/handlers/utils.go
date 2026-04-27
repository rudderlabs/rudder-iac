package handlers

import (
	"fmt"

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

// ToImportSpec creates a clean Spec for a transformation-related resource.
// Import metadata travels separately via importmanifest.ImportEntry slices
// returned from FormatForExport — it is no longer embedded here.
func ToImportSpec(
	kind string,
	metadataName string,
	specData map[string]any,
) (*specs.Spec, error) {
	metadata := specs.Metadata{
		Name: metadataName,
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
