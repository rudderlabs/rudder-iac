package definitions

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
)

// buildGatedKeyPaths derives the reverse index of source-type-gated config
// properties (keypath -> entitled source types), failing on definitions that
// gate unsupported source types or keys absent from the config model.
func buildGatedKeyPaths(def *DestinationDefinition, configType reflect.Type) (map[string][]string, error) {
	index := make(map[string][]string)

	for _, prop := range def.Properties {
		if len(prop.SourceTypes) == 0 {
			continue
		}

		if prop.LocalKey == "" {
			return nil, fmt.Errorf("gated config property has no local key")
		}

		for _, sourceType := range prop.SourceTypes {
			if !slices.Contains(def.SourceTypes, sourceType) {
				return nil, fmt.Errorf("config key %q gated on unsupported source type %q", prop.LocalKey, sourceType)
			}
		}

		if !configStructHasKeyPath(configType, prop.LocalKey) {
			return nil, fmt.Errorf("gated config key %q does not resolve to a config model field", prop.LocalKey)
		}

		keyPath := localKeyToPointer(prop.LocalKey)
		if _, exists := index[keyPath]; exists {
			return nil, fmt.Errorf("duplicate gated config key %q", prop.LocalKey)
		}
		index[keyPath] = append([]string(nil), prop.SourceTypes...)
	}

	return index, nil
}

// localKeyToPointer converts a gjson dot path over local keys to the JSON
// pointer form used by ConfigError paths ("a.b" -> "/a/b").
func localKeyToPointer(localKey string) string {
	return strings.ReplaceAll(localKey, ".", "/")
}

func configStructHasKeyPath(typ reflect.Type, localKey string) bool {
	current := derefType(typ)
	for _, segment := range strings.Split(localKey, ".") {
		current = elemStructType(current)
		if current == nil || current.Kind() != reflect.Struct {
			return false
		}
		field, ok := structFieldsByMapstructureTag(current)[segment]
		if !ok {
			return false
		}
		current = derefType(field.Type)
	}
	return true
}

// elemStructType unwraps slice/array/map layers so keypaths address fields of
// collection elements (e.g. "headers.to" on []webhookHeader).
func elemStructType(typ reflect.Type) reflect.Type {
	typ = derefType(typ)
	for typ != nil && (typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array || typ.Kind() == reflect.Map) {
		typ = derefType(typ.Elem())
	}
	return typ
}
