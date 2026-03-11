package transformation

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	libraryhandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/library"
	transformationhandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/transformation"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/parser"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// validateTransformationSemanticValid primary responsibility is to validate
// that the imports in the transformation code resolve to libraries in the graph
func validateTransformationSemanticValid(
	_ string,
	_ string,
	_ map[string]any,
	spec specs.TransformationSpec,
	graph *resources.Graph,
) []vrules.ValidationResult {
	resource, exists := graph.GetResource(resources.URN(spec.ID, transformationhandler.HandlerMetadata.ResourceType))
	if !exists {
		return []vrules.ValidationResult{{
			Reference: "/id",
			Message:   "'transformation' resource not found in graph",
		}}
	}

	transformationResource, ok := resource.RawData().(*model.TransformationResource)
	if !ok {
		return []vrules.ValidationResult{{
			Reference: "/id",
			Message:   "'transformation' resource must be valid in the graph",
		}}
	}

	codeParser, err := parser.NewParser(transformationResource.Language)
	if err != nil {
		return []vrules.ValidationResult{{
			Reference: sourceReference(spec),
			Message:   fmt.Sprintf("failed to create parser for language %s: %s", transformationResource.Language, err.Error()),
		}}
	}

	imports, err := codeParser.ExtractImports(transformationResource.Code)
	if err != nil {
		return []vrules.ValidationResult{{
			Reference: sourceReference(spec),
			Message:   fmt.Sprintf("failed to extract imports from transformation code: %s", err.Error()),
		}}
	}

	availableHandles := make(map[string]struct{})
	for _, libraryResource := range graph.ResourcesByType(libraryhandler.HandlerMetadata.ResourceType) {
		libraryData, ok := libraryResource.RawData().(*model.LibraryResource)
		if !ok || libraryData.ImportName == "" {
			continue
		}

		availableHandles[libraryData.ImportName] = struct{}{}
	}

	reference := sourceReference(spec)
	seenMissingHandles := make(map[string]struct{})

	var results []vrules.ValidationResult
	for _, handleName := range imports {
		if _, exists := availableHandles[handleName]; exists {
			continue
		}

		if _, seen := seenMissingHandles[handleName]; seen {
			continue
		}
		seenMissingHandles[handleName] = struct{}{}

		results = append(results, vrules.ValidationResult{
			Reference: reference,
			Message:   fmt.Sprintf("'%s' imported library not found", handleName),
		})
	}

	return results
}

func sourceReference(spec specs.TransformationSpec) string {
	if spec.File != "" {
		return "/file"
	}

	return "/code"
}

func NewTransformationImportsSemanticValidRule() vrules.Rule {
	return rules.NewTypedRule(
		"transformations/transformation/semantic-valid",
		vrules.Error,
		"transformation imports must resolve to existing transformation libraries",
		vrules.Examples{},
		rules.NewSemanticPatternValidator(
			rules.V1VersionPatterns(transformationhandler.HandlerMetadata.SpecKind),
			validateTransformationSemanticValid,
		),
	)
}
