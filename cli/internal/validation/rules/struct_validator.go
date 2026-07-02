package rules

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// defaultTagNameFunc resolves field names using yaml then json tags (json takes priority).
var defaultTagNameFunc = func(fld reflect.StructField) string {
	var (
		ymlTag, ymlOK   = fld.Tag.Lookup("yaml")
		jsonTag, jsonOK = fld.Tag.Lookup("json")
	)

	name := strings.ToLower(fld.Name)

	if ymlOK {
		name = strings.SplitN(ymlTag, ",", 2)[0]
	}

	// json overrides yaml
	if jsonOK {
		name = strings.SplitN(jsonTag, ",", 2)[0]
	}

	return name
}

// CustomValidateRule is a validation rule
// that is defined by the user and be added as a tag to the struct.
// go-validator will then use the function to validate the field
// on which the tag is added.
type CustomValidateFunc struct {
	Tag  string
	Func validator.Func
}

// defaultValidators holds globally registered validators that are automatically
// included in every ValidateStruct call (e.g., the "pattern" validator)
var defaultValidators []CustomValidateFunc

// RegisterDefaultValidator adds a validator to the list of default validators
// that are automatically registered in ValidateStruct every time.
func RegisterDefaultValidator(fn CustomValidateFunc) {
	defaultValidators = append(defaultValidators, fn)
}

// ValidateStruct validates a struct using go-playground/validator tags and returns
// validation results with JSON Pointer references. The basePath is prepended to all
// references to support nested struct validation (e.g., basePath="/metadata" for
// validating metadata produces references like "/metadata/name").
// Field names in errors are resolved using yaml/json tags (json takes priority).
func ValidateStruct(data any, basePath string, funcs ...CustomValidateFunc) (validator.ValidationErrors, error) {
	return validateStruct(data, defaultTagNameFunc, funcs...)
}

// ValidateStructWithTagPriority validates like ValidateStruct but resolves field names
// using the provided tag priority list instead of the default yaml/json order.
// Earlier tags in the list take priority. Falls back to lowercase field name if no tag found.
// Example: tags=["mapstructure"] makes err.Field() and err.Namespace() use mapstructure tag names.
func ValidateStructWithTagPriority(data any, tags []string, funcs ...CustomValidateFunc) (validator.ValidationErrors, error) {
	return validateStruct(data, buildTagNameFunc(tags), funcs...)
}

func buildTagNameFunc(tags []string) func(reflect.StructField) string {
	return func(fld reflect.StructField) string {
		for _, tag := range tags {
			if val, ok := fld.Tag.Lookup(tag); ok {
				name, _, _ := strings.Cut(val, ",")
				if name != "" && name != "-" {
					return name
				}
			}
		}
		return strings.ToLower(fld.Name)
	}
}

func validateStruct(data any, tagFn func(reflect.StructField) string, funcs ...CustomValidateFunc) (validator.ValidationErrors, error) {
	v := validator.New()
	v.RegisterTagNameFunc(tagFn)

	for _, fn := range defaultValidators {
		if err := v.RegisterValidation(fn.Tag, fn.Func); err != nil {
			return nil, fmt.Errorf("registering default validation rule: %w", err)
		}
	}

	for _, fn := range funcs {
		if err := v.RegisterValidation(fn.Tag, fn.Func); err != nil {
			return nil, fmt.Errorf("registering validation rule: %w", err)
		}
	}

	if err := v.Struct(data); err != nil {
		var invalidValidationError *validator.InvalidValidationError
		if errors.As(err, &invalidValidationError) {
			return nil, fmt.Errorf("invalid validation error: %w", err)
		}

		var errs validator.ValidationErrors
		errors.As(err, &errs)

		return errs, nil
	}

	return nil, nil
}
