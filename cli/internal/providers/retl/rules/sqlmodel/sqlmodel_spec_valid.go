package sqlmodel

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateSQLModelSpec = func(
	_ string,
	_ string,
	_ map[string]any,
	spec sqlmodel.SQLModelSpec,
) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(spec, "")
	if err != nil {
		return []rules.ValidationResult{{
			Message: err.Error(),
		}}
	}

	return funcs.ParseValidationErrors(validationErrors, reflect.TypeOf(spec))
}

var validateSQLModelV1Spec = func(
	filePath string,
	_ string,
	_ string,
	_ map[string]any,
	spec sqlmodel.SQLModelSpec,
) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(spec, "")
	if err != nil {
		return []rules.ValidationResult{{
			Message: err.Error(),
		}}
	}

	results := funcs.ParseValidationErrors(validationErrors, reflect.TypeOf(spec))

	if spec.File != nil && spec.SQL == nil {
		sqlContent, validationErr := resolveAndValidateSQLFile(filePath, *spec.File)
		if validationErr != nil {
			results = append(results, *validationErr)
		} else if strings.TrimSpace(sqlContent) == "" {
			results = append(results, rules.ValidationResult{
				Reference: "/file",
				Message:   "'sql' content is empty after resolving 'file'",
			})
		}
	}

	return results
}

func resolveAndValidateSQLFile(specFilePath, sqlFilePath string) (string, *rules.ValidationResult) {
	resolvedPath := sqlFilePath
	if !filepath.IsAbs(sqlFilePath) {
		specDir := filepath.Dir(specFilePath)
		resolvedPath = filepath.Clean(filepath.Join(specDir, sqlFilePath))
	}

	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		return "", &rules.ValidationResult{
			Reference: "/file",
			Message:   "failed to read sql file: " + err.Error(),
		}
	}

	return string(content), nil
}

func NewSQLModelSpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"retl/sqlmodel/spec-syntax-valid",
		rules.Error,
		"retl sql model spec syntax must be valid",
		rules.Examples{},
		prules.NewPatternValidator(
			prules.LegacyVersionPatterns(sqlmodel.ResourceKind),
			validateSQLModelSpec,
		),
		prules.NewPatternValidatorWithPath(
			[]rules.MatchPattern{rules.MatchKindVersion(sqlmodel.ResourceKind, specs.SpecVersionV1)},
			validateSQLModelV1Spec,
		),
	)
}
