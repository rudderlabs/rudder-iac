package definitions

import (
	"fmt"
	"strings"

	"github.com/kaptinlin/jsonschema"
)

func compileSchema(schemaBytes []byte) (*jsonschema.Schema, error) {
	compiler := jsonschema.NewCompiler().SetDefaultDialect(jsonschema.Draft7)
	schema, err := compiler.Compile(schemaBytes)
	if err != nil {
		return nil, fmt.Errorf("compiling schema: %w", err)
	}
	return schema, nil
}

func validateConfig(compiled *jsonschema.Schema, config map[string]any) []ConfigError {
	if compiled == nil {
		return nil
	}

	result := compiled.ValidateMap(config)
	if result == nil || result.IsValid() {
		return nil
	}

	return collectConfigErrors(result, "")
}

func collectConfigErrors(result *jsonschema.EvaluationResult, instancePath string) []ConfigError {
	if result == nil {
		return nil
	}

	currentPath := instancePath + result.InstanceLocation
	errors := make([]ConfigError, 0)

	for key, evalErr := range result.Errors {
		path := configErrorPath(currentPath, key, evalErr)
		errors = append(errors, ConfigError{
			Path:    path,
			Message: evalErr.Error(),
		})
	}

	for _, detail := range result.Details {
		errors = append(errors, collectConfigErrors(detail, currentPath)...)
	}

	return errors
}

func configErrorPath(instancePath, errorKey string, evalErr *jsonschema.EvaluationError) string {
	switch evalErr.Keyword {
	case "required":
		if property := paramPropertyName(evalErr.Params["property"]); property != "" {
			return joinInstancePath(instancePath, property)
		}
	case "additionalProperties":
		if property, ok := evalErr.Params["property"].(string); ok {
			return joinInstancePath(instancePath, property)
		}
	}

	if strings.HasPrefix(errorKey, "/") {
		return normalizeValidationPath(instancePath + errorKey)
	}

	if isValidationLeafKeyword(errorKey) {
		if instancePath != "" {
			return normalizeValidationPath(instancePath)
		}
	}

	if errorKey != "" && !isSchemaBranchKeyword(errorKey) {
		return normalizeValidationPath(joinInstancePath(instancePath, errorKey))
	}

	if instancePath != "" {
		return normalizeValidationPath(instancePath)
	}

	return normalizeValidationPath("/" + errorKey)
}

func paramPropertyName(value any) string {
	property, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.Trim(property, "'")
}

func joinInstancePath(basePath, property string) string {
	if property == "" {
		return normalizeValidationPath(basePath)
	}
	if basePath == "" {
		return normalizeValidationPath("/" + property)
	}
	return normalizeValidationPath(basePath + "/" + property)
}

func normalizeValidationPath(path string) string {
	if path == "" {
		return path
	}

	knownSuffixes := []string{
		"/type", "/enum", "/pattern", "/const", "/schema", "/format",
		"/minimum", "/maximum", "/minLength", "/maxLength", "/minItems", "/maxItems",
		"/properties", "/required",
	}
	for _, suffix := range knownSuffixes {
		if strings.HasSuffix(path, suffix) {
			return strings.TrimSuffix(path, suffix)
		}
	}
	return path
}

func isValidationLeafKeyword(keyword string) bool {
	switch keyword {
	case "type", "enum", "pattern", "const", "format", "minimum", "maximum", "minLength", "maxLength", "minItems", "maxItems", "schema":
		return true
	default:
		return false
	}
}

func isSchemaBranchKeyword(keyword string) bool {
	switch keyword {
	case "properties", "required", "allOf", "anyOf", "oneOf", "not", "if", "then", "else":
		return true
	default:
		return false
	}
}
