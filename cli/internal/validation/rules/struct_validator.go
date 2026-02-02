package rules

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var tagNameFunc = func(fld reflect.StructField) string {

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

// CustomValidateRule is a validation rule
// that is defined by the user and be added as a tag to the struct.
// go-validator will then use the function to validate the field
// on which the tag is added.
type CustomValidateFunc struct {
	Tag  string
	Func validator.Func
}

// ValidateStruct validates a struct using go-playground/validator tags and returns
// validation results with JSON Pointer references. The basePath is prepended to all
// references to support nested struct validation (e.g., basePath="/metadata" for
// validating metadata produces references like "/metadata/name").
func ValidateStruct(data any, basePath string, funcs ...CustomValidateFunc) (validator.ValidationErrors, error) {

	v := validator.New()
	v.RegisterTagNameFunc(tagNameFunc)

	for _, fn := range funcs {
		err := v.RegisterValidation(fn.Tag, fn.Func)
		if err == nil {
			continue
		}
		return nil, fmt.Errorf("registering validation rule: %w", err)
	}

	if err := v.Struct(data); err != nil {
		// This check is needed because we can pass in a nil pointer to this function
		// so the validator returns an invalid validation error.
		var invalidValidationError *validator.InvalidValidationError
		if errors.As(
			err,
			&invalidValidationError,
		) {
			return nil, fmt.Errorf("invalid validation error: %w", err)
		}

		var errs validator.ValidationErrors
		errors.As(err, &errs)

		return errs, nil
	}

	return nil, nil
}
