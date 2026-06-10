package export

import (
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
)

var secretStringType = reflect.TypeOf(secret.String{})

// scaffoldSecretRefs walks data (typically a *Spec about to be exported) and
// replaces every secret.String with a "{{ .VAR }}" variable reference. A remote
// never returns a secret's real value, so serializing it would only produce a
// useless masked literal; a variable reference plus a scaffolded var file gives
// the user a place to supply the value, injected back by variable substitution
// on apply.
//
// Variable names are built from pathPrefix (the spec's relative path) plus the
// field's JSON path, so they are deterministic and stable across re-imports.
// Slice elements are identified by their "id" field when they have one, since
// positional indices are not stable identities for exported resources.
//
// Sanitizing path components into variable names can alias distinct fields
// (e.g. ids "a-b" and "a_b") onto the same name; that would silently feed one
// value to two different secrets, so it is reported as an error instead.
func scaffoldSecretRefs(data any, pathPrefix []string) error {
	v := reflect.ValueOf(data)
	if !v.IsValid() {
		return nil
	}

	s := &secretScaffolder{claimed: make(map[string]string)}
	s.replaceSecrets(v, pathPrefix)
	return s.err
}

// secretScaffolder tracks which field path claimed each variable name so
// sanitization collisions surface as errors instead of aliased variables.
type secretScaffolder struct {
	claimed map[string]string
	err     error
}

func (s *secretScaffolder) tokenFor(path []string) string {
	var (
		name      = varName(path)
		fieldPath = strings.Join(path, ".")
	)
	if existing, ok := s.claimed[name]; ok && s.err == nil {
		s.err = fmt.Errorf("variable name %s for secret at %q collides with the one for %q; rename one of the resources", name, fieldPath, existing)
	}
	s.claimed[name] = fieldPath
	return fmt.Sprintf("{{ .%s }}", name)
}

func (s *secretScaffolder) replaceSecrets(v reflect.Value, path []string) {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return
		}
		s.replaceSecrets(v.Elem(), path)

	case reflect.Interface:
		if v.IsNil() || !v.CanSet() {
			return
		}
		// The concrete value inside an interface is not addressable; mutate an
		// addressable copy and store it back.
		elem := reflect.New(v.Elem().Type()).Elem()
		elem.Set(v.Elem())
		s.replaceSecrets(elem, path)
		v.Set(elem)

	case reflect.Struct:
		if v.Type() == secretStringType {
			if v.CanSet() {
				v.Set(reflect.ValueOf(secret.NewRef(s.tokenFor(path))))
			}
			return
		}
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			if !field.IsExported() {
				continue
			}
			name, inline, skip := jsonFieldName(field)
			if skip {
				continue
			}
			if inline {
				s.replaceSecrets(v.Field(i), path)
				continue
			}
			s.replaceSecrets(v.Field(i), append(path, name))
		}

	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			s.replaceSecrets(v.Index(i), append(path, elementName(v.Index(i), i)))
		}

	case reflect.Map:
		for _, key := range v.MapKeys() {
			// Map values are not addressable; mutate an addressable copy and
			// store it back.
			elem := reflect.New(v.Type().Elem()).Elem()
			elem.Set(v.MapIndex(key))
			s.replaceSecrets(elem, append(path, fmt.Sprintf("%v", key.Interface())))
			v.SetMapIndex(key, elem)
		}
	}
}

// jsonFieldName resolves the name a struct field will have after json.Marshal.
// inline is true for embedded fields without a tag (their fields are promoted),
// and skip is true for fields json.Marshal omits entirely.
func jsonFieldName(field reflect.StructField) (name string, inline bool, skip bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", false, true
	}
	name = strings.Split(tag, ",")[0]
	if name == "" {
		if field.Anonymous {
			return "", true, false
		}
		name = field.Name
	}
	return name, false, false
}

// elementName identifies a slice element in a variable name. Exported resource
// items conventionally carry an "id" field holding their stable external ID;
// fall back to the positional index when there is none.
func elementName(elem reflect.Value, index int) string {
	for elem.Kind() == reflect.Pointer || elem.Kind() == reflect.Interface {
		if elem.IsNil() {
			return strconv.Itoa(index)
		}
		elem = elem.Elem()
	}

	switch elem.Kind() {
	case reflect.Struct:
		for i := 0; i < elem.NumField(); i++ {
			name, _, skip := jsonFieldName(elem.Type().Field(i))
			if skip || name != "id" {
				continue
			}
			if id := elem.Field(i); id.Kind() == reflect.String && id.String() != "" {
				return id.String()
			}
		}
	case reflect.Map:
		if elem.Type().Key().Kind() == reflect.String {
			if id := elem.MapIndex(reflect.ValueOf("id")); id.IsValid() {
				if s, ok := id.Interface().(string); ok && s != "" {
					return s
				}
			}
		}
	}

	return strconv.Itoa(index)
}

// varPathPrefix converts a spec's relative path into the leading variable-name
// components, e.g. "retl/sql-models/my-source.yaml" -> [retl sql-models my-source].
func varPathPrefix(relativePath string) []string {
	trimmed := strings.TrimSuffix(filepath.ToSlash(relativePath), filepath.Ext(relativePath))
	return strings.Split(trimmed, "/")
}

var (
	camelBoundary  = regexp.MustCompile(`([a-z0-9])([A-Z])`)
	nonVarChars    = regexp.MustCompile(`[^A-Za-z0-9]+`)
	underscoreRuns = regexp.MustCompile(`_+`)
	leadingDigit   = regexp.MustCompile(`^[0-9]`)
)

// varName flattens path components into a SCREAMING_SNAKE_CASE variable name
// that satisfies the substitutor's variable-name grammar.
func varName(path []string) string {
	parts := make([]string, 0, len(path))
	for _, p := range path {
		p = camelBoundary.ReplaceAllString(p, "${1}_${2}")
		p = nonVarChars.ReplaceAllString(p, "_")
		p = strings.Trim(underscoreRuns.ReplaceAllString(p, "_"), "_")
		if p == "" {
			continue
		}
		parts = append(parts, strings.ToUpper(p))
	}

	name := strings.Join(parts, "_")
	if leadingDigit.MatchString(name) {
		name = "_" + name
	}
	return name
}
