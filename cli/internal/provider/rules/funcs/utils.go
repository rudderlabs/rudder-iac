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

func getErrorMessage(err validator.FieldError) string {
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

	case "excluded_unless":
		return fmt.Sprintf("'%s' is not allowed unless '%s'", fieldName, err.Param())

	case "value_types":
		return fmt.Sprintf("'%s' values must be one of [%s]", fieldName, err.Param())

	default:
		return fmt.Sprintf("'%s' is not valid: %s", fieldName, err.Error())
	}
}
