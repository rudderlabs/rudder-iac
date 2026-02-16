package funcs

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// arrayIndexRegex matches array indices like [0], [1], etc.
var arrayIndexRegex = regexp.MustCompile(`\[(\d+)\]`)

// ParseValidationErrors converts validation errors from the struct validator to validation results.
// rootType enables cross-field tags (required_without, excluded_with) to resolve struct field
// names to their JSON tag display names via reflection. Pass nil when not using cross-field tags.
func ParseValidationErrors(errs validator.ValidationErrors, rootType reflect.Type) []rules.ValidationResult {
	results := []rules.ValidationResult{}

	for _, err := range errs {
		results = append(results, rules.ValidationResult{
			Reference: namespaceToJSONPointer(err.Namespace()),
			Message:   getErrorMessage(err, rootType),
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

func getErrorMessage(err validator.FieldError, rootType reflect.Type) string {
	fieldName := err.Field()

	switch err.ActualTag() {
	case "required":
		return fmt.Sprintf("'%s' is required", fieldName)

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

	case "array_item_types":
		return fmt.Sprintf("'%s' values must be one of [%s]", fieldName, err.Param())

	case "excluded_unless":
		// `validate:"excluded_unless=Type value"`
		params := strings.Split(err.Param(), " ")

		// We need to resolve the "Type"
		otherField := resolveFieldDisplayName(params[0], err, rootType)
		return fmt.Sprintf("'%s' is not allowed unless '%s %s'", fieldName, otherField, params[1])

	case "required_without":
		otherField := resolveFieldDisplayName(err.Param(), err, rootType)
		return fmt.Sprintf("'%s' is required when '%s' is not specified", fieldName, otherField)

	case "excluded_with":
		otherField := resolveFieldDisplayName(err.Param(), err, rootType)
		return fmt.Sprintf("'%s' and '%s' cannot be specified together", fieldName, otherField)

	default:
		return fmt.Sprintf("'%s' is not valid: %s", fieldName, err.Error())
	}
}

// resolveFieldDisplayName is a helper function to help resolve the field display name
// from the struct to using json tag name. This is required because in cross-field tags like
// `required_without`, `excluded_unless` etc the go-validator doesn't make use of the registered tag name func
// and therefore the error message contains the actual field name (contained in param) on the struct which the
// customer may be not be aware of
// Example:
//
//	struct ABCSpec {
//	  FieldA string `json:"field_a" validate:"required_without=FieldB"`
//	  FieldB string `json:"field_b"`
//	}
//
// If FieldA is required without FieldB, the error message will be:
// "'field_a' is required when 'FieldB' is not specified"
//
//	instead of
//
// "'field_a' is required when 'field_b' is not specified"
//
// To solve for this, we leverage the StructNamespace on the field error
// which in example would be "ABCSpec.FieldA" and then locate the parent struct type
// and we then use reflection to get the desired structFieldName and extract it's json tag name
func resolveFieldDisplayName(structFieldName string, err validator.FieldError, rootType reflect.Type) string {
	if rootType == nil {
		return structFieldName
	}

	// Walk StructNamespace to find the parent struct type.
	// e.g., "OuterSpec.Inner.FieldA" → walk to InnerSpec, then look up param there.
	parts := strings.Split(err.StructNamespace(), ".")

	currentType := derefType(rootType)
	// Skip first element (root struct name) as it will be used
	// by default as the current type in the logic after loop
	// and last element (the field itself)
	for i := 1; i < len(parts)-1; i++ {
		fieldName := parts[i]

		// Strip array index suffix if present (e.g., "Items[0]" → "Items")
		if idx := strings.Index(fieldName, "["); idx != -1 {
			fieldName = fieldName[:idx]
		}

		field, found := currentType.FieldByName(fieldName)
		if !found {
			return structFieldName
		}

		currentType = derefType(field.Type)
		if currentType.Kind() == reflect.Slice || currentType.Kind() == reflect.Array {
			currentType = derefType(currentType.Elem())
		}

	}

	// currentType is now the parent struct containing the param field
	field, found := currentType.FieldByName(structFieldName)
	if !found {
		return structFieldName
	}

	if jsonTag := field.Tag.Get("json"); jsonTag != "" {
		if name, _, _ := strings.Cut(jsonTag, ","); name != "" && name != "-" {
			return name
		}
	}

	return structFieldName
}

// derefType unwraps pointer types to their underlying element type.
func derefType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		return t.Elem()
	}
	return t
}
