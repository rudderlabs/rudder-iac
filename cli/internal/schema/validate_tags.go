package schema

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/invopop/jsonschema"
)

// enricher folds go-playground/validator `validate` tag constraints into an
// already-reflected schema. The invopop reflector understands `json` tags but
// ignores `validate`, so field-level rules (required, oneof, eq, and bounds)
// that the CLI enforces at load time would otherwise be absent.
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
	defs         jsonschema.Definitions
	visited      map[reflect.Type]bool
	visitedNodes map[enrichmentVisit]bool
}

type enrichmentVisit struct {
	schema *jsonschema.Schema
	typeOf reflect.Type
}

// walk enriches the schema node describing Go type t. Local references are
// followed to their shared definition. Visits are tracked by both schema node
// and Go type: a recursive root may be expanded inline and also have a distinct
// $defs node, and both need enrichment before back-edges are stopped.
func (e *enricher) walk(s *jsonschema.Schema, t reflect.Type) {
	if s == nil || t == nil {
		return
	}
	t = deref(t)
	s = e.resolveSchema(s)
	if s == nil {
		return
	}

	if e.visitedNodes == nil {
		e.visitedNodes = make(map[enrichmentVisit]bool)
	}
	visit := enrichmentVisit{schema: s, typeOf: t}
	if e.visitedNodes[visit] {
		return
	}
	e.visitedNodes[visit] = true
	if e.visited != nil {
		e.visited[t] = true
	}
	e.walkNode(s, t)
}

// resolveSchema follows chains of local $refs. Tracking the schema pointers in
// the chain makes malformed or mutually-referential $defs harmless too.
func (e *enricher) resolveSchema(s *jsonschema.Schema) *jsonschema.Schema {
	seen := map[*jsonschema.Schema]bool{}
	for s != nil && s.Ref != "" {
		if seen[s] {
			return nil
		}
		seen[s] = true
		s = e.resolveRef(s.Ref)
	}
	return s
}

func (e *enricher) walkNode(s *jsonschema.Schema, t reflect.Type) {
	switch t.Kind() {
	case reflect.Struct:
		e.walkStruct(s, t)
	case reflect.Slice, reflect.Array:
		e.walk(s.Items, t.Elem())
	case reflect.Map:
		if s.AdditionalProperties != nil {
			e.walk(s.AdditionalProperties, t.Elem())
			return
		}
		for _, child := range s.PatternProperties {
			e.walk(child, t.Elem())
		}
	}
}

func (e *enricher) walkStruct(s *jsonschema.Schema, t reflect.Type) {
	if s.Properties == nil {
		return
	}

	// The reflector derives `required` from json `omitempty`, but the CLI only
	// truly requires fields carrying validate:"required". Rebuild the complete
	// set, including inline embedded fields, from validate tags.
	s.Required = mergeRequired(nil, e.walkStructFields(s, t))
}

func (e *enricher) walkStructFields(s *jsonschema.Schema, t reflect.Type) []string {
	var required []string
	fields := schemaFields(s, t)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		// Anonymous structs without an explicit JSON name contribute their fields
		// at this level. Process them inline without marking their type as globally
		// visited: the same type may also appear as a named $ref.
		if embedded(field) {
			embeddedType := deref(field.Type)
			if embeddedType.Kind() == reflect.Struct {
				required = append(required, e.walkStructFields(s, embeddedType)...)
			}
			continue
		}

		name, prop, ok := schemaProperty(s, field)
		if !ok {
			continue
		}

		rules := parseValidate(field.Tag.Get("validate"))
		if rules.required {
			required = append(required, name)
		}
		applyRules(prop, field.Type, rules)
		applyConditionalRules(s, name, rules, fields)

		e.walk(prop, field.Type)
		if !rules.required && e.valueStructFieldRequired(field, prop) {
			required = append(required, name)
		}
	}
	return required
}

func (e *enricher) valueStructFieldRequired(field reflect.StructField, prop *jsonschema.Schema) bool {
	if field.Type.Kind() == reflect.Pointer || strings.Contains(field.Tag.Get("json"), "omitempty") || strings.Contains(field.Tag.Get("mapstructure"), "omitempty") {
		return false
	}
	if deref(field.Type).Kind() != reflect.Struct {
		return false
	}
	return hasRequiredDescendant(e.resolveSchema(prop), map[*jsonschema.Schema]bool{})
}

func hasRequiredDescendant(s *jsonschema.Schema, seen map[*jsonschema.Schema]bool) bool {
	if s == nil || seen[s] {
		return false
	}
	seen[s] = true
	if len(s.Required) > 0 {
		return true
	}
	for _, child := range schemaChildren(s) {
		if hasRequiredDescendant(child, seen) {
			return true
		}
	}
	return false
}

func schemaChildren(s *jsonschema.Schema) []*jsonschema.Schema {
	children := append(append(append([]*jsonschema.Schema{}, s.AllOf...), s.AnyOf...), s.OneOf...)
	children = append(children, s.Not, s.If, s.Then, s.Else, s.Items, s.Contains, s.AdditionalProperties, s.PropertyNames, s.ContentSchema)
	children = append(children, s.PrefixItems...)
	if s.Properties != nil {
		for pair := s.Properties.Oldest(); pair != nil; pair = pair.Next() {
			children = append(children, pair.Value)
		}
	}
	for _, definitions := range []map[string]*jsonschema.Schema{s.PatternProperties, s.DependentSchemas, s.Definitions} {
		for _, child := range definitions {
			children = append(children, child)
		}
	}
	return children
}

type schemaField struct {
	name   string
	typeOf reflect.Type
}

func schemaFields(s *jsonschema.Schema, t reflect.Type) map[string]schemaField {
	fields := make(map[string]schemaField)
	collectSchemaFields(fields, s, t)
	return fields
}

func collectSchemaFields(fields map[string]schemaField, s *jsonschema.Schema, t reflect.Type) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		if embedded(field) {
			embeddedType := deref(field.Type)
			if embeddedType.Kind() == reflect.Struct {
				collectSchemaFields(fields, s, embeddedType)
			}
			continue
		}
		name, _, ok := schemaProperty(s, field)
		if !ok {
			continue
		}
		fields[field.Name] = schemaField{name: name, typeOf: field.Type}
	}
}

func (e *enricher) resolveRef(ref string) *jsonschema.Schema {
	const prefix = "#/$defs/"
	if !strings.HasPrefix(ref, prefix) {
		return nil
	}
	return e.defs[strings.TrimPrefix(ref, prefix)]
}

// validateRules is the mapped subset of a field's validator tag. Values stay
// textual until applied because the field's Go kind determines both the JSON
// Schema keyword and the JSON value type.
type validateRules struct {
	required        bool
	oneof           []string
	min             string
	max             string
	eq              string
	requiredWithout []string
	requiredIf      []requiredCondition
	requiredUnless  []requiredCondition
	item            *validateRules
	hasMin          bool
	hasMax          bool
	hasEq           bool
}

type requiredCondition struct {
	field string
	value string
}

var oneofValuePattern = regexp.MustCompile(`'(?:[^']*)'|\S+`)

func parseValidate(tag string) validateRules {
	var r validateRules
	if tag == "" {
		return r
	}
	target := &r
	for _, tok := range strings.Split(tag, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "dive" {
			r.item = &validateRules{}
			target = r.item
			continue
		}
		applyToken(target, tok)
	}
	return r
}

func applyToken(r *validateRules, tok string) {
	switch {
	case tok == "required":
		r.required = true
	case strings.HasPrefix(tok, "required_without="):
		r.requiredWithout = strings.Fields(afterEq(tok))
	case strings.HasPrefix(tok, "required_if="):
		r.requiredIf = parseRequiredConditions(afterEq(tok))
	case strings.HasPrefix(tok, "required_unless="):
		r.requiredUnless = parseRequiredConditions(afterEq(tok))
	case strings.HasPrefix(tok, "oneof="):
		r.oneof = parseOneof(strings.TrimPrefix(tok, "oneof="))
	case strings.HasPrefix(tok, "eq="):
		r.eq = afterEq(tok)
		r.hasEq = true
	case strings.HasPrefix(tok, "gte="), strings.HasPrefix(tok, "min="):
		r.min = afterEq(tok)
		r.hasMin = true
	case strings.HasPrefix(tok, "lte="), strings.HasPrefix(tok, "max="):
		r.max = afterEq(tok)
		r.hasMax = true
	}
}

func parseRequiredConditions(param string) []requiredCondition {
	fields := strings.Fields(param)
	conditions := make([]requiredCondition, 0, len(fields)/2)
	for i := 0; i+1 < len(fields); i += 2 {
		conditions = append(conditions, requiredCondition{field: fields[i], value: fields[i+1]})
	}
	return conditions
}

func parseOneof(param string) []string {
	values := oneofValuePattern.FindAllString(param, -1)
	for i, value := range values {
		values[i] = strings.Trim(value, "'")
	}
	return values
}

func applyRules(prop *jsonschema.Schema, t reflect.Type, r validateRules) {
	t = deref(t)
	applyScalarRules(prop, t.Kind(), r)

	if r.item == nil {
		return
	}
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		applyScalarRules(prop.Items, deref(t.Elem()).Kind(), *r.item)
	case reflect.Map:
		elemKind := deref(t.Elem()).Kind()
		applyScalarRules(prop.AdditionalProperties, elemKind, *r.item)
		for _, child := range prop.PatternProperties {
			applyScalarRules(child, elemKind, *r.item)
		}
	}
}

func applyScalarRules(prop *jsonschema.Schema, kind reflect.Kind, r validateRules) {
	if prop == nil {
		return
	}
	if len(r.oneof) > 0 {
		if enum, ok := validationValues(r.oneof, kind); ok {
			prop.Enum = enum
		}
	}
	if r.hasEq {
		if value, ok := validationValue(r.eq, kind); ok {
			prop.Const = value
		}
	}

	switch kind {
	case reflect.String:
		if r.hasMin {
			prop.MinLength = uintValue(r.min)
		}
		if r.hasMax {
			prop.MaxLength = uintValue(r.max)
		}
	case reflect.Slice, reflect.Array:
		if r.hasMin {
			prop.MinItems = uintValue(r.min)
		}
		if r.hasMax {
			prop.MaxItems = uintValue(r.max)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		if r.hasMin {
			prop.Minimum = numberValue(r.min, kind)
		}
		if r.hasMax {
			prop.Maximum = numberValue(r.max, kind)
		}
	}
}

func applyConditionalRules(parent *jsonschema.Schema, fieldName string, r validateRules, fields map[string]schemaField) {
	if len(r.requiredWithout) > 0 {
		requiredAlternatives := []string{fieldName}
		for _, goName := range r.requiredWithout {
			field, ok := fields[goName]
			if !ok {
				continue
			}
			requiredAlternatives = append(requiredAlternatives, field.name)
		}
		appendRequiredAnyOf(parent, requiredAlternatives)
	}
	for _, conditions := range [][]requiredCondition{r.requiredIf} {
		ifSchema, ok := requiredConditionSchema(conditions, fields)
		if !ok {
			continue
		}
		parent.AllOf = append(parent.AllOf, &jsonschema.Schema{
			If:   ifSchema,
			Then: requiredSchema(fieldName),
		})
	}
	for _, conditions := range [][]requiredCondition{r.requiredUnless} {
		unlessSchema, ok := requiredConditionSchema(conditions, fields)
		if !ok {
			continue
		}
		parent.AllOf = append(parent.AllOf, &jsonschema.Schema{
			If: &jsonschema.Schema{
				Not: unlessSchema,
			},
			Then: requiredSchema(fieldName),
		})
	}
}

func appendRequiredAnyOf(parent *jsonschema.Schema, names []string) {
	names = uniqueNonEmpty(names)
	if len(names) == 0 {
		return
	}
	branches := make([]*jsonschema.Schema, 0, len(names))
	for _, name := range names {
		branches = append(branches, requiredSchema(name))
	}
	parent.AllOf = append(parent.AllOf, &jsonschema.Schema{AnyOf: branches})
}

func requiredConditionSchema(conditions []requiredCondition, fields map[string]schemaField) (*jsonschema.Schema, bool) {
	if len(conditions) == 0 {
		return nil, false
	}
	props := jsonschema.NewProperties()
	required := make([]string, 0, len(conditions))
	for _, condition := range conditions {
		field, ok := fields[condition.field]
		if !ok {
			return nil, false
		}
		value, ok := validationValue(condition.value, deref(field.typeOf).Kind())
		if !ok {
			return nil, false
		}
		props.Set(field.name, &jsonschema.Schema{Const: value})
		required = append(required, field.name)
	}
	return &jsonschema.Schema{
		Type:       "object",
		Properties: props,
		Required:   uniqueNonEmpty(required),
	}, true
}

func requiredSchema(name string) *jsonschema.Schema {
	return &jsonschema.Schema{Required: []string{name}}
}

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func schemaProperty(s *jsonschema.Schema, f reflect.StructField) (string, *jsonschema.Schema, bool) {
	for _, name := range propertyNames(f) {
		if name == "-" {
			return "", nil, false
		}
		if prop, ok := s.Properties.Get(name); ok {
			return name, prop, true
		}
	}
	return "", nil, false
}

// propertyNames returns candidate names in decoder precedence order. The
// reflector uses json tags, while some loader structs only provide
// mapstructure tags; checking both preserves those constraints without
// changing reflected property names.
func propertyNames(f reflect.StructField) []string {
	names := make([]string, 0, 3)
	for _, tagKey := range []string{"json", "mapstructure"} {
		tag, ok := f.Tag.Lookup(tagKey)
		if !ok {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name == "-" {
			return []string{"-"}
		}
		if name != "" {
			names = appendUnique(names, name)
		}
	}
	return appendUnique(names, f.Name)
}

func embedded(f reflect.StructField) bool {
	if !f.Anonymous {
		return false
	}
	if tag, ok := f.Tag.Lookup("json"); ok {
		name := strings.Split(tag, ",")[0]
		if name == "-" || name != "" {
			return false
		}
	}
	return deref(f.Type).Kind() == reflect.Struct
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func afterEq(tok string) string {
	_, v, _ := strings.Cut(tok, "=")
	return v
}

func uintValue(s string) *uint64 {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return nil
	}
	return &v
}

func numberValue(s string, kind reflect.Kind) json.Number {
	if _, ok := validationValue(s, kind); !ok {
		return ""
	}
	return json.Number(s)
}

func validationValues(values []string, kind reflect.Kind) ([]any, bool) {
	converted := make([]any, 0, len(values))
	for _, raw := range values {
		value, ok := validationValue(raw, kind)
		if !ok {
			return nil, false
		}
		converted = append(converted, value)
	}
	return converted, true
}

func validationValue(raw string, kind reflect.Kind) (any, bool) {
	switch kind {
	case reflect.String:
		return raw, true
	case reflect.Bool:
		value, err := strconv.ParseBool(raw)
		return value, err == nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value, err := strconv.ParseInt(raw, 10, kindBits(kind))
		return json.Number(strconv.FormatInt(value, 10)), err == nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value, err := strconv.ParseUint(raw, 10, kindBits(kind))
		return json.Number(strconv.FormatUint(value, 10)), err == nil
	case reflect.Float32, reflect.Float64:
		value, err := strconv.ParseFloat(raw, kindBits(kind))
		return json.Number(strconv.FormatFloat(value, 'g', -1, kindBits(kind))), err == nil
	default:
		return nil, false
	}
}

func kindBits(kind reflect.Kind) int {
	switch kind {
	case reflect.Int8, reflect.Uint8:
		return 8
	case reflect.Int16, reflect.Uint16:
		return 16
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		return 32
	default:
		return 64
	}
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
