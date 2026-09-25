package schema

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"

	invopop "github.com/invopop/jsonschema"
)

type enricher struct {
	defs    invopop.Definitions
	visited map[reflect.Type]bool
}

func (e *enricher) walk(s *invopop.Schema, t reflect.Type) {
	if s == nil || t == nil {
		return
	}
	t = deref(t)

	if s.Ref != "" {
		e.walkRef(s.Ref, t)
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

func (e *enricher) walkRef(ref string, t reflect.Type) {
	target := e.resolveRef(ref)
	if target == nil || e.visited[t] {
		return
	}
	e.walk(target, t)
}

func (e *enricher) walkStruct(s *invopop.Schema, t reflect.Type) {
	if s.Properties == nil {
		return
	}

	s.Required = nil
	for i := 0; i < t.NumField(); i++ {
		e.walkField(s, t.Field(i))
	}
}

func (e *enricher) walkField(s *invopop.Schema, field reflect.StructField) {
	if !field.IsExported() {
		return
	}
	if field.Anonymous {
		e.walkStruct(s, deref(field.Type))
		return
	}

	name := fieldName(field)
	if name == "" || name == "-" {
		return
	}
	prop, ok := s.Properties.Get(name)
	if !ok {
		return
	}

	rules := parseValidate(field.Tag.Get("validate"))
	if rules.required {
		s.Required = appendUnique(s.Required, name)
	}
	applyRules(prop, field.Type, rules)
	e.walk(prop, field.Type)
	if !rules.required && e.valueStructFieldRequired(field, prop) {
		s.Required = appendUnique(s.Required, name)
	}
}

func (e *enricher) resolveRef(ref string) *invopop.Schema {
	const prefix = "#/$defs/"
	if !strings.HasPrefix(ref, prefix) {
		return nil
	}
	return e.defs[strings.TrimPrefix(ref, prefix)]
}

type validateRules struct {
	required bool
	dive     *validateRules
	oneof    []string
	min      *json.Number
	max      *json.Number
	eq       string
}

func parseValidate(tag string) validateRules {
	var rules validateRules
	current := &rules
	for _, token := range strings.Split(tag, ",") {
		token = strings.TrimSpace(token)
		if token == "dive" {
			rules.dive = &validateRules{}
			current = rules.dive
			continue
		}
		parseValidateToken(current, token)
	}
	return rules
}

func parseValidateToken(rules *validateRules, token string) {
	switch {
	case token == "required":
		rules.required = true
	case strings.HasPrefix(token, "oneof="):
		rules.oneof = strings.Fields(strings.TrimPrefix(token, "oneof="))
	case strings.HasPrefix(token, "eq="):
		rules.eq = strings.TrimPrefix(token, "eq=")
	case strings.HasPrefix(token, "gte="), strings.HasPrefix(token, "min="):
		rules.min = numberAfterEquals(token)
	case strings.HasPrefix(token, "lte="), strings.HasPrefix(token, "max="):
		rules.max = numberAfterEquals(token)
	}
}

func applyRules(prop *invopop.Schema, fieldType reflect.Type, rules validateRules) {
	if rules.dive != nil && prop.Items != nil {
		applyRules(prop.Items, deref(fieldType).Elem(), *rules.dive)
	}
	kind := deref(fieldType).Kind()
	if len(rules.oneof) > 0 {
		prop.Enum = make([]any, len(rules.oneof))
		for i, value := range rules.oneof {
			prop.Enum[i] = coerceValidateValue(value, kind)
		}
	}
	if rules.eq != "" {
		prop.Const = coerceValidateValue(rules.eq, kind)
	}

	if rules.min != nil {
		applyMinimum(prop, kind, *rules.min)
	}
	if rules.max != nil {
		applyMaximum(prop, kind, *rules.max)
	}
}

func (e *enricher) valueStructFieldRequired(field reflect.StructField, prop *invopop.Schema) bool {
	fieldType := field.Type
	if fieldType.Kind() == reflect.Pointer || strings.Contains(field.Tag.Get("json"), "omitempty") || strings.Contains(field.Tag.Get("mapstructure"), "omitempty") {
		return false
	}
	fieldType = deref(fieldType)
	if fieldType.Kind() != reflect.Struct {
		return false
	}
	if prop.Ref == "" {
		return hasRequiredDescendant(prop)
	}
	return hasRequiredDescendant(e.resolveRef(prop.Ref))
}

func hasRequiredDescendant(s *invopop.Schema) bool {
	return hasRequiredDescendantSeen(s, map[*invopop.Schema]bool{})
}

func hasRequiredDescendantSeen(s *invopop.Schema, seen map[*invopop.Schema]bool) bool {
	if s == nil || seen[s] {
		return false
	}
	seen[s] = true
	if len(s.Required) > 0 {
		return true
	}
	for _, child := range childSchemas(s) {
		if hasRequiredDescendantSeen(child, seen) {
			return true
		}
	}
	return false
}

func coerceValidateValue(value string, kind reflect.Kind) any {
	switch kind {
	case reflect.Bool:
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			return parsed
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			return parsed
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err == nil {
			return parsed
		}
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(value, 64)
		if err == nil {
			return parsed
		}
	}
	return value
}

func applyMinimum(prop *invopop.Schema, kind reflect.Kind, value json.Number) {
	n, err := strconv.ParseUint(string(value), 10, 64)
	if err != nil {
		return
	}
	switch kind {
	case reflect.String:
		prop.MinLength = &n
	case reflect.Slice, reflect.Array:
		prop.MinItems = &n
	default:
		prop.Minimum = value
	}
}

func applyMaximum(prop *invopop.Schema, kind reflect.Kind, value json.Number) {
	n, err := strconv.ParseUint(string(value), 10, 64)
	if err != nil {
		return
	}
	switch kind {
	case reflect.String:
		prop.MaxLength = &n
	case reflect.Slice, reflect.Array:
		prop.MaxItems = &n
	default:
		prop.Maximum = value
	}
}

func fieldName(field reflect.StructField) string {
	for _, tagName := range []string{"json", "mapstructure", "yaml"} {
		tag := field.Tag.Get(tagName)
		if tag == "" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name != "" {
			return name
		}
	}
	return field.Name
}

func numberAfterEquals(token string) *json.Number {
	_, value, ok := strings.Cut(token, "=")
	if !ok {
		return nil
	}
	number := json.Number(value)
	if _, err := number.Float64(); err != nil {
		return nil
	}
	return &number
}

func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
