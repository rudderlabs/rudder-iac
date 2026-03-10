package transformation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	transformationhandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/transformation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

func validateTransformationSpec(
	_ string,
	_ string,
	filePath string,
	_ map[string]any,
	spec specs.TransformationSpec,
) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(spec, "")
	if err != nil {
		return []rules.ValidationResult{{
			Reference: "/",
			Message:   err.Error(),
		}}
	}

	results := funcs.ParseValidationErrors(validationErrors, reflect.TypeOf(spec))

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
	}

	for idx, test := range spec.Tests {
		if test.Name != "" && strings.TrimSpace(test.Name) == "" {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/tests/%d/name", idx),
				Message:   "test name must not be blank or whitespace-only",
			})
			continue
		}

		if test.Name != "" && !transformationhandler.TestNameRegex.MatchString(test.Name) {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/tests/%d/name", idx),
				Message:   `test name must match ^[A-Za-z0-9 _/\-]+$`,
			})
		}

		for _, field := range []struct {
			name string
			path string
		}{
			{name: "input", path: test.Input},
			{name: "output", path: test.Output},
		} {
			if field.path == "" {
				continue
			}

			resolvedPath, err := resolveSpecRelativePath(filePath, field.path)
			if err != nil {
				results = append(results, rules.ValidationResult{
					Reference: fmt.Sprintf("/tests/%d/%s", idx, field.name),
					Message:   err.Error(),
				})
				continue
			}

			dirResults := validateTestDirectory(idx, field.name, resolvedPath)
			results = append(results, dirResults...)
			if len(dirResults) > 0 {
				continue
			}

			results = append(results, validateTestJSONFiles(idx, field.name, resolvedPath)...)
		}
	}

	return results
}

func resolveSpecRelativePath(specFilePath, targetPath string) (string, error) {
	if filepath.IsAbs(targetPath) {
		return "", fmt.Errorf("path must be relative to the spec file directory: %q", targetPath)
	}

	for _, segment := range splitPathSegments(targetPath) {
		if segment == ".." {
			return "", fmt.Errorf("path must not contain '..' segments: %q", targetPath)
		}
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
			Message:   fmt.Sprintf("path %q does not exist or is not accessible", resolvedPath),
		}}
	}

	if info.IsDir() {
		return []rules.ValidationResult{{
			Reference: "/file",
			Message:   fmt.Sprintf("path %q must be a file", resolvedPath),
		}}
	}

	return nil
}

func validateTestDirectory(testIdx int, fieldName, resolvedPath string) []rules.ValidationResult {
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return []rules.ValidationResult{{
			Reference: fmt.Sprintf("/tests/%d/%s", testIdx, fieldName),
			Message:   fmt.Sprintf("path %q does not exist or is not accessible", resolvedPath),
		}}
	}

	if !info.IsDir() {
		return []rules.ValidationResult{{
			Reference: fmt.Sprintf("/tests/%d/%s", testIdx, fieldName),
			Message:   fmt.Sprintf("path %q must be a directory", resolvedPath),
		}}
	}

	return nil
}

func validateTestJSONFiles(testIdx int, fieldName, dir string) []rules.ValidationResult {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []rules.ValidationResult{{
			Reference: fmt.Sprintf("/tests/%d/%s", testIdx, fieldName),
			Message:   fmt.Sprintf("reading directory %q: %v", dir, err),
		}}
	}

	var results []rules.ValidationResult
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		contents, err := os.ReadFile(filePath)
		if err != nil {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/tests/%d/%s/%s", testIdx, fieldName, entry.Name()),
				Message:   fmt.Sprintf("reading file %q: %v", filePath, err),
			})
			continue
		}

		if !json.Valid(contents) {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/tests/%d/%s/%s", testIdx, fieldName, entry.Name()),
				Message:   "file must contain valid JSON",
			})
		}
	}

	return results
}

func NewTransformationSpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"transformations/transformation/spec-syntax-valid",
		rules.Error,
		"transformation spec syntax must be valid",
		rules.Examples{},
		prules.NewPathAwarePatternValidator(
			prules.V1VersionPatterns(transformationhandler.HandlerMetadata.SpecKind),
			validateTransformationSpec,
		),
	)
}
