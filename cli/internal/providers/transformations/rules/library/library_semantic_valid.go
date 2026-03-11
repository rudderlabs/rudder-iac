package library

import (
	"fmt"
	"sort"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	libraryhandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/library"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// validateLibrarySemanticValid validates import_name uniqueness
func validateLibrarySemanticValid(
	_ string,
	_ string,
	_ map[string]any,
	spec specs.TransformationLibrarySpec,
	graph *resources.Graph,
) []vrules.ValidationResult {
	resource, exists := graph.GetResource(resources.URN(spec.ID, libraryhandler.HandlerMetadata.ResourceType))
	if !exists {
		return []vrules.ValidationResult{{
			Reference: "/id",
			Message:   "'transformation-library' resource not found in graph",
		}}
	}

	_, ok := resource.RawData().(*model.LibraryResource)
	if !ok {
		return []vrules.ValidationResult{{
			Reference: "/id",
			Message:   "'transformation-library' resource must be valid in the graph",
		}}
	}

	var results []vrules.ValidationResult

	var allLibs []*model.LibraryResource
	for _, lib := range graph.ResourcesByType(libraryhandler.HandlerMetadata.ResourceType) {
		libData, _ := lib.RawData().(*model.LibraryResource)
		allLibs = append(allLibs, libData)
	}

	sort.Slice(allLibs, func(i, j int) bool {
		return allLibs[i].ID < allLibs[j].ID
	})

	seenImportNames := make(map[string]string)
	for _, libData := range allLibs {
		if existingID, exists := seenImportNames[libData.ImportName]; exists {
			// Only report the error for the current library being validated
			if libData.ID == spec.ID {
				results = append(results, vrules.ValidationResult{
					Reference: "/import_name",
					Message:   fmt.Sprintf("'import_name' '%s' is already used by library '%s'", libData.ImportName, existingID),
				})
			}
		} else {
			seenImportNames[libData.ImportName] = libData.ID
		}
	}

	return results
}

func NewLibrarySemanticValidRule() vrules.Rule {
	return rules.NewTypedRule(
		"transformations/transformation-library/semantic-valid",
		vrules.Error,
		"transformation library must be semantically valid",
		vrules.Examples{},
		rules.NewSemanticPatternValidator(
			rules.V1VersionPatterns(libraryhandler.HandlerMetadata.SpecKind),
			validateLibrarySemanticValid,
		),
	)
}
