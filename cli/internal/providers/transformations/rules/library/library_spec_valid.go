package library

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	libraryhandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/library"
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
				Message:   fmt.Sprintf("'import_name' must be camelCase of 'name': expected '%s', got '%s'", expectedImportName, spec.ImportName),
			})
		}
	}

	if spec.File != "" {
		resolvedPath, err := resolveSpecRelativePath(filePath, spec.File)
		if err != nil {
			results = append(results, rules.ValidationResult{
				Reference: "/file",
				Message:   err.Error(),
			})
		} else {
			results = append(results, validateSpecFile(resolvedPath)...)
		}

		// Validate file extension matches language
		ext := filepath.Ext(spec.File)
		var expectedExt string

		switch spec.Language {
		case "javascript":
			expectedExt = ".js"
		case "python":
			expectedExt = ".py"
		}

		if ext != expectedExt {
			results = append(results, rules.ValidationResult{
				Reference: "/file",
				Message:   fmt.Sprintf("file extension must be '%s' for language '%s', got '%s'", expectedExt, spec.Language, ext),
			})
		}
	}

	return results
}

func resolveSpecRelativePath(specFilePath, targetPath string) (string, error) {
	if filepath.IsAbs(targetPath) {
		return "", errors.New("path must be relative to the spec file directory")
	}

	if slices.Contains(splitPathSegments(targetPath), "..") {
		return "", errors.New("path must not contain '..' segments")
	}

	return filepath.Join(filepath.Dir(specFilePath), targetPath), nil
}

func splitPathSegments(path string) []string {
	return strings.FieldsFunc(path, func(r rune) bool {
		return r == '/' || r == '\\'
	})
}

func validateSpecFile(resolvedPath string) []rules.ValidationResult {
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return []rules.ValidationResult{{
			Reference: "/file",
			Message:   "path does not exist or is not accessible",
		}}
	}

	if info.IsDir() {
		return []rules.ValidationResult{{
			Reference: "/file",
			Message:   "path must be a file",
		}}
	}

	return nil
}

func NewLibrarySpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"transformations/transformation-library/spec-syntax-valid",
		rules.Error,
		"transformation library spec syntax must be valid",
		rules.Examples{},
		prules.NewPathAwarePatternValidator(
			prules.V1VersionPatterns(libraryhandler.HandlerMetadata.SpecKind),
			validateLibrarySpec,
		),
	)
}
