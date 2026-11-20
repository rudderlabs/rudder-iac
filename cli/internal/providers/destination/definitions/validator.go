package definitions

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-viper/mapstructure/v2"

	prulefuncs "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

func validateConfigModel(config map[string]any, configType reflect.Type, basePath string, extraFuncs ...rules.CustomValidateFunc) []ConfigError {
	configType = derefType(configType)
	if configType == nil || configType.Kind() != reflect.Struct {
		return []ConfigError{{
			Path:    basePath,
			Message: "config model must be a struct",
		}}
	}

	errors := findUnknownKeys(config, configType, basePath)

	decoded := reflect.New(configType).Interface()
	if err := decodeConfig(config, decoded); err != nil {
		return append(errors, DecodeErrors(err, configType, basePath)...)
	}

	errors = append(errors, structValidationErrors(decoded, basePath, configType, extraFuncs...)...)
	return errors
}

func decodeConfig(config any, dest any) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           dest,
		WeaklyTypedInput: false,
	})
	if err != nil {
		return fmt.Errorf("creating config decoder: %w", err)
	}

	if err := decoder.Decode(config); err != nil {
		return fmt.Errorf("decoding config: %w", err)
	}
	return nil
}

// DecodeErrors reports each field a mapstructure decode into root failed on as
// its own error, anchored on the field and naming the type the model expects
// instead of the decoder's Go type names.
func DecodeErrors(err error, root reflect.Type, basePath string) []ConfigError {
	decodeErrors := flattenDecodeErrors(err)
	if len(decodeErrors) == 0 {
		return []ConfigError{{Path: basePath, Message: err.Error()}}
	}

	errors := make([]ConfigError, 0, len(decodeErrors))
	for _, decodeErr := range decodeErrors {
		segments := decodeNameSegment.FindAllString(decodeErr.Name(), -1)
		for i, segment := range segments {
			segments[i] = strings.TrimSuffix(strings.TrimPrefix(segment, "["), "]")
		}
		path := basePath
		for _, segment := range segments {
			path = joinConfigPath(path, segment)
		}
		errors = append(errors, ConfigError{
			Path:    path,
			Message: decodeErrorMessage(root, segments, decodeErr),
		})
	}
	return errors
}

// decodeNameSegment matches one key of a mapstructure field name such as
// consent_management[web][0].provider: a struct field name, or a list index or
// map key in brackets, kept whole because a source-type key can hold a dot.
var decodeNameSegment = regexp.MustCompile(`\[[^\]]*\]|[^.\[\]]+`)

// flattenDecodeErrors collects the DecodeError on every branch of the joined
// error tree mapstructure returns; each one carries its field's path.
func flattenDecodeErrors(err error) []*mapstructure.DecodeError {
	if decodeErr, ok := err.(*mapstructure.DecodeError); ok {
		return []*mapstructure.DecodeError{decodeErr}
	}

	switch wrapper := err.(type) {
	case interface{ Unwrap() []error }:
		var collected []*mapstructure.DecodeError
		for _, wrapped := range wrapper.Unwrap() {
			collected = append(collected, flattenDecodeErrors(wrapped)...)
		}
		return collected
	case interface{ Unwrap() error }:
		return flattenDecodeErrors(wrapper.Unwrap())
	}
	return nil
}

// decodeErrorMessage reads the expected type off the model rather than the
// error: mapstructure reports slice and struct mismatches as untyped text.
func decodeErrorMessage(root reflect.Type, segments []string, decodeErr *mapstructure.DecodeError) string {
	noun := typeNoun(typeAt(root, segments))
	if len(segments) == 0 || noun == "" {
		return decodeErr.Error()
	}

	last := len(segments) - 1
	subject := fmt.Sprintf("'%s'", segments[last])
	if parent := typeAt(root, segments[:last]); parent.Kind() == reflect.Slice || parent.Kind() == reflect.Array {
		subject = "list item"
	}
	return subject + " must be " + noun
}

// typeAt follows decode path segments from root through struct fields, map
// values and list items, returning nil once the path leaves the model.
func typeAt(root reflect.Type, segments []string) reflect.Type {
	typ := derefType(root)
	for _, segment := range segments {
		switch typ.Kind() {
		case reflect.Struct:
			field, ok := structFieldsByMapstructureTag(typ)[segment]
			if !ok {
				return nil
			}
			typ = derefType(field.Type)
		case reflect.Map, reflect.Slice, reflect.Array:
			typ = derefType(typ.Elem())
		default:
			return nil
		}
	}
	return typ
}

func typeNoun(typ reflect.Type) string {
	if typ == nil {
		return ""
	}

	switch typ.Kind() {
	case reflect.String:
		return "a string"
	case reflect.Bool:
		return "a boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "an integer"
	case reflect.Float32, reflect.Float64:
		return "a number"
	case reflect.Slice, reflect.Array:
		return "an array"
	case reflect.Map, reflect.Struct:
		return "an object"
	default:
		return ""
	}
}

func structValidationErrors(decoded any, basePath string, rootType reflect.Type, extraFuncs ...rules.CustomValidateFunc) []ConfigError {
	validationErrors, err := rules.ValidateStructWithTagPriority(decoded, []string{"mapstructure"}, append(configValidateFuncs(), extraFuncs...)...)
	if err != nil {
		return []ConfigError{{
			Path:    basePath,
			Message: err.Error(),
		}}
	}

	if len(validationErrors) == 0 {
		return nil
	}

	results := prulefuncs.ParseMapstructureValidationErrors(validationErrors, rootType)
	errors := make([]ConfigError, 0, len(results))
	for _, result := range results {
		path := basePath
		if result.Reference != "" {
			path = joinConfigPath(basePath, strings.TrimPrefix(result.Reference, "/"))
		}

		errors = append(errors, ConfigError{
			Path:    path,
			Message: result.Message,
		})
	}

	return errors
}
