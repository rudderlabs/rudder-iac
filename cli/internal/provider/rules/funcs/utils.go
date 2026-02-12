package funcs

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// arrayIndexRegex matches array indices like [0], [1], etc.
var arrayIndexRegex = regexp.MustCompile(`\[(\d+)\]`)

// ParseValidationErrors is a helper function which converts the validation errors from
// the struct validator to the validation results
func ParseValidationErrors(errs validator.ValidationErrors) []rules.ValidationResult {
	results := []rules.ValidationResult{}

	for _, err := range errs {
		results = append(results, rules.ValidationResult{
			Reference: namespaceToJSONPointer(err.Namespace()),
			Message:   getErrorMessage(err),
		})
	}

	return results
}

// NamespaceToJSONPointer converts validator's StructNamespace to JSON Pointer format.
// Example: "Spec.inners[1].surname" → "/inners/1/surname"
func namespaceToJSONPointer(namespace string) string {
	// Remove root struct name (everything before first dot)
	if idx := strings.Index(namespace, "."); idx != -1 {
		namespace = namespace[idx+1:]
	}

	namespace = arrayIndexRegex.ReplaceAllString(namespace, "/$1")
	namespace = strings.ReplaceAll(namespace, ".", "/")

	return fmt.Sprintf("/%s", namespace)
}

// getFieldTagName returns the yaml/json tag name for a field in the same struct as the error field.
// It uses the same tag resolution logic as the validator's tagNameFunc: yaml -> json -> lowercase.
func getFieldTagName(err validator.FieldError, fieldName string) string {
	// Get the parent struct type by navigating the type tree from the root
	parentType := getParentStructType(err)
	if parentType == nil || parentType.Kind() != reflect.Struct {
		// Fallback: return original field name if we can't find the parent type
		return fieldName
	}

	// Look up the field in the parent struct
	field, found := parentType.FieldByName(fieldName)
	if !found {
		// Fallback: return original field name if field not found
		return fieldName
	}

	if yamlTag, ok := field.Tag.Lookup("yaml"); ok {
		return strings.SplitN(yamlTag, ",", 2)[0]
	}

	if jsonTag, ok := field.Tag.Lookup("json"); ok {
		return strings.SplitN(jsonTag, ",", 2)[0]
	}

	return fieldName
}

// getParentStructType navigates the type tree using the error's namespace to find
// the struct type that contains the field that failed validation.
func getParentStructType(err validator.FieldError) reflect.Type {
	// Get the struct namespace (uses actual struct field names)
	// Example: "Metadata.Import.Workspaces[0].Resources[0].LocalID"
	namespace := err.StructNamespace()

	// Remove array indices from namespace: "Workspaces[0]" -> "Workspaces"
	// This is needed because the validator includes indices but we only care about types
	namespace = arrayIndexRegex.ReplaceAllString(namespace, "")

	// Parse the namespace to get the parent path
	// Remove the field name to get the parent struct path
	parts := strings.Split(namespace, ".")
	if len(parts) < 2 {
		// No parent (top-level field)
		return nil
	}

	// Remove the last part (field name) to get parent path
	// Example: "Metadata.Import.Workspaces.Resources"
	parentPath := parts[:len(parts)-1]

	// Start from the root type (specs.Metadata)
	// We know we're validating Metadata based on the call site
	currentType := reflect.TypeFor[specs.Metadata]()

	// Navigate through the type tree following the path
	// Skip the first element since it's the root type name
	for i := 1; i < len(parentPath); i++ {
		fieldName := parentPath[i]

		// Handle pointer types
		if currentType.Kind() == reflect.Ptr {
			currentType = currentType.Elem()
		}

		// Handle slice/array types (e.g., "Workspaces" or "Resources")
		if currentType.Kind() == reflect.Slice || currentType.Kind() == reflect.Array {
			currentType = currentType.Elem()
			if currentType.Kind() == reflect.Ptr {
				currentType = currentType.Elem()
			}
		}

		// Navigate to the next field
		if currentType.Kind() == reflect.Struct {
			field, found := currentType.FieldByName(fieldName)
			if !found {
				return nil
			}
			currentType = field.Type
		} else {
			return nil
		}
	}

	// Handle final pointer/slice unwrapping
	if currentType.Kind() == reflect.Ptr {
		currentType = currentType.Elem()
	}
	if currentType.Kind() == reflect.Slice || currentType.Kind() == reflect.Array {
		currentType = currentType.Elem()
		if currentType.Kind() == reflect.Ptr {
			currentType = currentType.Elem()
		}
	}

	if currentType.Kind() != reflect.Struct {
		return nil
	}

	return currentType
}

func getErrorMessage(err validator.FieldError) string {
	fieldName := err.Field()

	switch err.ActualTag() {
	case "required":
		return fmt.Sprintf("'%s' is required", fieldName)

	case "required_without":
		// Get the yaml tag name of the referenced field instead of using struct field name
		paramFieldName := getFieldTagName(err, err.Param())
		return fmt.Sprintf("'%s' is required when '%s' is not supplied", fieldName, paramFieldName)

	case "pattern":
		if msg, ok := getPatternErrorMessage(err.Param()); ok {
			return fmt.Sprintf("'%s' is not valid: %s", fieldName, msg)
		}
		return fmt.Sprintf("'%s' does not match the required pattern", fieldName)

	case "oneof":
		return fmt.Sprintf("'%s' must be one of [%s]", fieldName, err.Param())

	case "gte":
		if err.Kind() == reflect.String || err.Kind() == reflect.Slice {
			// For string and slice, the gte tag is used to validate the length
			// and therefore it needs a different error message.
			return fmt.Sprintf("'%s' length must be greater than or equal to %s", fieldName, err.Param())
		}

		return fmt.Sprintf("'%s' must be greater than or equal to %s",
			fieldName,
			err.Param(),
		)

	case "lte":
		if err.Kind() == reflect.String || err.Kind() == reflect.Slice {
			// For string and slice, the lte tag is used to validate the length
			// and therefore it needs a different error message.
			return fmt.Sprintf("'%s' length must be less than or equal to %s", fieldName, err.Param())
		}

		return fmt.Sprintf("'%s' must be less than or equal to %s",
			fieldName,
			err.Param(),
		)

	case "min":
		if err.Kind() == reflect.String || err.Kind() == reflect.Slice || err.Kind() == reflect.Array {
			return fmt.Sprintf("'%s' length must be greater than or equal to %s", fieldName, err.Param())
		}
		return fmt.Sprintf("'%s' must be greater than or equal to %s", fieldName, err.Param())

	case "max":
		if err.Kind() == reflect.String || err.Kind() == reflect.Slice || err.Kind() == reflect.Array {
			return fmt.Sprintf("'%s' length must be less than or equal to %s", fieldName, err.Param())
		}
		return fmt.Sprintf("'%s' must be less than or equal to %s", fieldName, err.Param())

	case "eq":
		return fmt.Sprintf("'%s' must equal '%s'", fieldName, err.Param())

	default:
		return fmt.Sprintf("'%s' is not valid: %s", fieldName, err.Error())
	}
}
