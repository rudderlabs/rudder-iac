package validation

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// arrayIndexRegex matches array indices like [0], [1], etc.
var arrayIndexRegex = regexp.MustCompile(`\[(\d+)\]`)

func tagNameFunc() func(fld reflect.StructField) string {

	return func(fld reflect.StructField) string {
		var (
			ymlTag, ymlOK   = fld.Tag.Lookup("yaml")
			jsonTag, jsonOK = fld.Tag.Lookup("json")
		)

		// By default, the field name to be used
		// is the lowercase of the struct's FieldName
		name := strings.ToLower(fld.Name)

		if ymlOK {
			name = strings.SplitN(ymlTag, ",", 2)[0]
		}

		// If both JSON and YAML tags are present,
		// then JSON tag overrides the YAML tag
		if jsonOK {
			name = strings.SplitN(jsonTag, ",", 2)[0]
		}

		return name
	}
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

// ValidateStruct validates a struct using go-playground/validator tags and returns
// validation results with JSON Pointer references. The basePath is prepended to all
// references to support nested struct validation (e.g., basePath="/metadata" for
// validating metadata produces references like "/metadata/name").
func ValidateStruct(data any, basePath string) []rules.ValidationResult {
	results := []rules.ValidationResult{}

	v := validator.New()
	v.RegisterTagNameFunc(tagNameFunc())

	if err := v.Struct(data); err != nil {
		var errs validator.ValidationErrors
		errors.As(err, &errs)

		for _, err := range errs {
			reference := namespaceToJSONPointer(err.Namespace())
			if basePath != "" {
				reference = basePath + reference
			}

			results = append(results, rules.ValidationResult{
				Reference: reference,
				Message:   getErrorMessage(err),
			})
		}
	}

	return results
}

func getErrorMessage(err validator.FieldError) string {
	fieldName := err.Field()

	switch err.ActualTag() {
	case "required":
		return fmt.Sprintf("'%s' is required", fieldName)

	default:
		return fmt.Sprintf("'%s' is not valid", fieldName)
	}
}
