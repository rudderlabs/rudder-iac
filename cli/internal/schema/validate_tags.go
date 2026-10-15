package schema

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/invopop/jsonschema"
)

// enricher folds go-playground/validator `validate` tag constraints into an
// already-reflected schema. The invopop reflector understands `json` tags but
// ignores `validate`, so field-level rules (required, oneof, gte/lte length
// bounds) that the CLI enforces at load time would otherwise be absent.
//
// Only the subset of validator semantics that maps cleanly to JSON Schema is
// translated; unmapped tokens (cross-field `excluded_with`, named `pattern`
// rules) are ignored so the schema stays a sound over-approximation of what the
// loader accepts — it never rejects a spec the loader would allow.
//
// Nested types are represented via $ref into the reflector's $defs map. The
// enricher resolves each ref to its definition and tracks visited Go types so
// recursive spec structs terminate.
type enricher struct {
	defs    jsonschema.Definitions
	visited map[reflect.Type]bool
}

// walk enriches the schema node describing Go type t. If the node is a $ref it
// is resolved to the shared definition (enriched once, since $defs entries are
// shared by name).
func (e *enricher) walk(s *jsonschema.Schema, t reflect.Type) {
	if s == nil {
		return
	}
	t = deref(t)

	if s.Ref != "" {
		target := e.resolveRef(s.Ref)
		if target == nil || e.visited[t] {
			return
		}
		e.visited[t] = true
		e.walk(target, t)
		return
	}

	switch t.Kind() {
	case reflect.Struct:
		if e.visited[t] {
			return
		}
		e.visited[t] = true
		e.walkStruct(s, t)
	case reflect.Slice, reflect.Array:
		e.walk(s.Items, t.Elem())
	}
}

func (e *enricher) walkStruct(s *jsonschema.Schema, t reflect.Type) {
	if s.Properties == nil {
		return
	}

	// The reflector derives `required` from json `omitempty`, but the CLI only
	// truly requires fields carrying validate:"required". Reset and rebuild the
	// required set from the validate tags so the schema mirrors loader
	// semantics rather than the presence/absence of omitempty.
	s.Required = nil

	var required []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		// Embedded structs contribute their fields at this level (mapstructure
		// squash / json inline), so recurse into the same schema node.
		if field.Anonymous {
			e.walkStruct(s, deref(field.Type))
			continue
		}

		name := jsonName(field)
		if name == "" || name == "-" {
			continue
		}

		prop, ok := s.Properties.Get(name)
		if !ok {
			continue
		}

		rules := parseValidate(field.Tag.Get("validate"))
		if rules.required {
			required = append(required, name)
		}
		applyRules(prop, rules)

		e.walk(prop, field.Type)
	}

	s.Required = mergeRequired(nil, required)
}

func (e *enricher) resolveRef(ref string) *jsonschema.Schema {
	const prefix = "#/$defs/"
	if !strings.HasPrefix(ref, prefix) {
		return nil
	}
	return e.defs[strings.TrimPrefix(ref, prefix)]
}

// validateRules is the mapped subset of a field's validator tag.
type validateRules struct {
	required bool
	oneof    []string
	min      *uint64 // string minLength / lower bound
	max      *uint64 // string maxLength / upper bound
	eq       string  // exact string value (validator `eq=`)
}

func parseValidate(tag string) validateRules {
	var r validateRules
	if tag == "" {
		return r
	}
	for _, tok := range strings.Split(tag, ",") {
		tok = strings.TrimSpace(tok)
		switch {
		case tok == "required":
			r.required = true
		case strings.HasPrefix(tok, "oneof="):
			r.oneof = strings.Fields(strings.TrimPrefix(tok, "oneof="))
		case strings.HasPrefix(tok, "eq="):
			r.eq = strings.TrimPrefix(tok, "eq=")
		case strings.HasPrefix(tok, "gte="), strings.HasPrefix(tok, "min="):
			if v, ok := parseUint(afterEq(tok)); ok {
				r.min = &v
			}
		case strings.HasPrefix(tok, "lte="), strings.HasPrefix(tok, "max="):
			if v, ok := parseUint(afterEq(tok)); ok {
				r.max = &v
			}
		}
	}
	return r
}

func applyRules(prop *jsonschema.Schema, r validateRules) {
	if len(r.oneof) > 0 {
		enum := make([]any, 0, len(r.oneof))
		for _, v := range r.oneof {
			enum = append(enum, v)
		}
		prop.Enum = enum
	}
	if r.eq != "" {
		prop.Const = r.eq
	}
	// Length bounds only apply to string-typed properties; validator gte/lte on
	// strings mean character length, which is JSON Schema minLength/maxLength.
	if prop.Type == "string" {
		if r.min != nil {
			prop.MinLength = r.min
		}
		if r.max != nil {
			prop.MaxLength = r.max
		}
	}
}

// jsonName returns the JSON property name for a struct field, preferring the
// `json` tag and falling back to `mapstructure`, mirroring how the loader keys
// fields when decoding the YAML spec map.
func jsonName(f reflect.StructField) string {
	for _, tagKey := range []string{"json", "mapstructure"} {
		tag := f.Tag.Get(tagKey)
		if tag == "" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name == "-" {
			return "-"
		}
		if name != "" {
			return name
		}
	}
	return f.Name
}

func afterEq(tok string) string {
	_, v, _ := strings.Cut(tok, "=")
	return v
}

func parseUint(s string) (uint64, bool) {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func mergeRequired(existing, added []string) []string {
	seen := make(map[string]struct{}, len(existing)+len(added))
	out := make([]string, 0, len(existing)+len(added))
	for _, list := range [][]string{existing, added} {
		for _, v := range list {
			if _, ok := seen[v]; ok {
				continue
			}
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
