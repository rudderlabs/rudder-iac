package library

import (
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	trules "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/rules"
	ttypes "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/samber/lo"
)

func validateLibrarySpec(
	_ string,
	_ string,
	filePath string,
	_ map[string]any,
	spec specs.TransformationLibrarySpec,
) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(spec, "")
	if err != nil {
		return []rules.ValidationResult{{
			Reference: "/",
			Message:   err.Error(),
		}}
	}

	results := funcs.ParseValidationErrors(
		validationErrors,
		reflect.TypeOf(spec),
	)

	// Validate import_name is camelCase of name
	if spec.Name != "" && spec.ImportName != "" {
		expectedImportName := lo.CamelCase(spec.Name)
		if spec.ImportName != expectedImportName {
			results = append(results, rules.ValidationResult{
				Reference: "/import_name",
				Message:   "'import_name' must be camelCase of 'name'",
			})
		}
	}

	if spec.File != "" {
		resolvedPath, err := trules.ResolveSpecRelativePath(filePath, spec.File)
		if err != nil {
			results = append(results, rules.ValidationResult{
				Reference: "/file",
				Message:   err.Error(),
			})
		} else {
			results = append(results, trules.ValidateSpecFile(resolvedPath)...)
		}

		// Validate file extension matches language
		ext := filepath.Ext(spec.File)
		expectedExt := trules.GetExpectedExtension(spec.Language)

		if expectedExt != "" && ext != expectedExt {
			results = append(results, rules.ValidationResult{
				Reference: "/file",
				Message:   fmt.Sprintf("file extension must be '%s'", expectedExt),
			})
		}
	}

	return results
}

func NewLibrarySpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"transformations/transformation-library/spec-syntax-valid",
		rules.Error,
		"transformation library spec syntax must be valid",
		rules.Examples{},
		prules.NewPathAwarePatternValidator(
			prules.V1VersionPatterns(ttypes.LibrarySpecKind),
			validateLibrarySpec,
		),
	)
}
