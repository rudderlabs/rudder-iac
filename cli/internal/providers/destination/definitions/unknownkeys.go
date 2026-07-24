package definitions

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

const unknownConfigFieldMessage = "unknown config field %q"

// findUnknownKeys walks a decoded config map against a struct type and reports
// every key present in the input that is not declared on the struct model.
func findUnknownKeys(value any, typ reflect.Type, basePath string) []ConfigError {
	typ = derefType(typ)
	if typ == nil || typ.Kind() != reflect.Struct {
		return nil
	}

	raw, ok := value.(map[string]any)
	if !ok {
		return nil
	}

	fieldsByTag := structFieldsByMapstructureTag(typ)
	errors := make([]ConfigError, 0)

	for key, nested := range raw {
		field, known := fieldsByTag[key]
		if !known {
			errors = append(errors, ConfigError{
				Path:    joinConfigPath(basePath, key),
				Message: fmt.Sprintf(unknownConfigFieldMessage, key),
			})
			continue
		}

		errors = append(errors,
			findUnknownKeysForField(
				nested,
				field.Type,
				joinConfigPath(basePath, key),
			)...,
		)
	}

	return errors
}

func findUnknownKeysForField(value any, fieldType reflect.Type, basePath string) []ConfigError {
	fieldType = derefType(fieldType)

	switch fieldType.Kind() {

	case reflect.Struct:
		return findUnknownKeys(value, fieldType, basePath)

	case reflect.Slice, reflect.Array:
		items, ok := value.([]any)
		if !ok {
			return nil
		}

		elemType := derefType(fieldType.Elem())
		errors := make([]ConfigError, 0, len(items))
		for i, item := range items {
			errors = append(
				errors,
				findUnknownKeys(
					item,
					elemType,
					joinConfigPath(basePath, strconv.Itoa(i)),
				)...,
			)
		}
		return errors

	case reflect.Map:
		items, ok := value.(map[string]any)
		if !ok {
			return nil
		}

		var errors []ConfigError
		for _, key := range slices.Sorted(maps.Keys(items)) {
			errors = append(
				errors,
				findUnknownKeysForField(
					items[key],
					fieldType.Elem(),
					joinConfigPath(basePath, key),
				)...,
			)
		}
		return errors

	default:
		return nil
	}
}

func structFieldsByMapstructureTag(typ reflect.Type) map[string]reflect.StructField {
	typ = derefType(typ)
	fields := make(map[string]reflect.StructField)

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}

		tag, ok := mapstructureFieldTag(field)
		if !ok {
			continue
		}

		fields[tag] = field
	}

	return fields
}

func mapstructureFieldTag(field reflect.StructField) (string, bool) {
	tag, ok := field.Tag.Lookup("mapstructure")
	if !ok {
		return "", false
	}

	name, _, _ := strings.Cut(tag, ",")
	if name == "" || name == "-" {
		return "", false
	}

	return name, true
}

func joinConfigPath(basePath, segment string) string {
	if basePath == "" {
		return "/" + segment
	}
	return basePath + "/" + segment
}

func derefType(typ reflect.Type) reflect.Type {
	if typ == nil {
		return nil
	}
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ
}
