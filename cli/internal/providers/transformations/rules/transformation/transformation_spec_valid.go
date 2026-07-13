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
	trules "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/rules"
	ttypes "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/types"
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

	results := funcs.ParseValidationErrors(
		validationErrors,
		reflect.TypeOf(spec),
	)

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

	for idx, test := range spec.Tests {
		if test.Name != "" && strings.TrimSpace(test.Name) == "" {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/tests/%d/name", idx),
				Message:   "'name' must not be blank or whitespace-only",
			})
			continue
		}

		if test.Name != "" && !transformationhandler.TestNameRegex.MatchString(test.Name) {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/tests/%d/name", idx),
				Message:   `'name' must match '^[A-Za-z0-9 _/\-]+$'`,
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

			resolvedPath, err := trules.ResolveSpecRelativePath(filePath, field.path)
			if err != nil {
				results = append(results, rules.ValidationResult{
					Reference: fmt.Sprintf("/tests/%d/%s", idx, field.name),
					Message:   err.Error(),
				})
				continue
			}

			dirResults := validateTestDirectory(
				idx,
				field.name,
				resolvedPath,
			)
			results = append(results, dirResults...)
			if len(dirResults) > 0 {
				continue
			}

			// only if the directory is valid, validate the JSON files
			results = append(
				results,
				validateTestJSONFiles(idx, field.name, resolvedPath)...,
			)
		}
	}

	return results
}

func validateTestDirectory(testIdx int, fieldName, resolvedPath string) []rules.ValidationResult {
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return []rules.ValidationResult{{
			Reference: fmt.Sprintf("/tests/%d/%s", testIdx, fieldName),
			Message:   "path does not exist or is not accessible",
		}}
	}

	if !info.IsDir() {
		return []rules.ValidationResult{{
			Reference: fmt.Sprintf("/tests/%d/%s", testIdx, fieldName),
			Message:   "path must be a directory",
		}}
	}

	return nil
}

// validateTestJSONFiles validates JSON fixtures and only accepts
// top-level JSON objects or arrays.
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
				Reference: fmt.Sprintf("/tests/%d/%s", testIdx, fieldName),
				Message:   fmt.Sprintf("file: %s unable to be read", entry.Name()),
			})
			continue
		}

		if !jsonValidObjectOrArray(contents) {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/tests/%d/%s", testIdx, fieldName),
				Message:   fmt.Sprintf("file: %s must contain valid object or array", entry.Name()),
			})
		}
	}

	return results
}

// jsonValidObjectOrArray returns true only for top-level JSON objects or arrays.
func jsonValidObjectOrArray(contents []byte) bool {
	var value any

	if err := json.Unmarshal(contents, &value); err != nil {
		return false
	}

	switch value.(type) {
	case map[string]any, []any:
		return true
	default:
		return false
	}
}

func NewTransformationSpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"transformations/transformation/spec-syntax-valid",
		rules.Error,
		"transformation spec syntax must be valid",
		rules.Examples{},
		prules.NewPathAwarePatternValidator(
			prules.V1VersionPatterns(ttypes.TransformationSpecKind),
			validateTransformationSpec,
		),
	)
}
