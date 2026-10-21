package definitions

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
)

// Description is what a registered definition declares about its config,
// read from the definition and its config model's struct tags. It is the
// source `rudder-cli docs destination` renders, so the documented keys are the
// ones validate checks against.
type Description struct {
	Type        string
	DisplayName string
	APIType     string
	Version     int64
	SourceTypes []string
	// ConnectionModes and ConnectionRequiredKeys are keyed by source token.
	ConnectionModes        map[string][]string
	ConnectionRequiredKeys map[string]map[string][]string
	SecretKeys             []string
	// Fields lists the config keys in struct order. Objects and lists of
	// objects appear only when a rule applies to the container itself; their
	// keys imply them otherwise.
	Fields []Field
	// Patterns holds every named pattern a field refers to.
	Patterns map[string]funcs.PatternInfo
	// CustomValidators are the definition's own validate tags. Only their Go
	// code says what they check.
	CustomValidators []string
}

// Field describes one config key.
type Field struct {
	// Key is the dotted local key; "[]" marks the items of a list, as in
	// "headers[].to".
	Key string
	// Type is string, boolean, integer, number, object, array<T> or map<K,T>.
	// connection_mode and consent_management are maps keyed by source token.
	Type string
	// Required is the bare `required` rule. On a nested key it applies only
	// once the enclosing block or list item is written.
	Required bool
	// RequiredIf makes the key required when every condition holds.
	RequiredIf []Condition
	// RequiredUnless makes the key required unless any condition holds.
	RequiredUnless []Condition
	// RequiredWith makes the key required when any of these keys is set.
	RequiredWith []string
	// ExcludedWith forbids the key when any of these keys is set.
	ExcludedWith []string
	// CheckedBy names the definition's custom validators on this key.
	CheckedBy []string
	// Enum holds the allowed values, typed like the field (or its items).
	Enum []any
	// OpenEnum is set when the key also takes a template or env. value in
	// place of an enum member.
	OpenEnum bool
	// Default applies when the key is omitted, and HasDefault tells a false or
	// empty default apart from none. On a nested key it applies only once the
	// enclosing block is written.
	Default    any
	HasDefault bool
	// Pattern names an entry of Description.Patterns. On a list it applies to
	// each item.
	Pattern string
	// AcceptsTemplate is set when the key also takes a `{{ path || fallback }}`
	// template, which the pattern does not have to match.
	AcceptsTemplate bool
	// Max is the `max` rule: a length for strings and lists, a value for numbers.
	Max *int
	// OnlyForSourceTypes lists the source tokens the key is valid for; empty
	// means every source type.
	OnlyForSourceTypes []string
	// Secret marks a SecretKeys path.
	Secret bool
}

// Nested reports whether the key sits inside a block or a list item.
func (f Field) Nested() bool {
	return strings.Contains(f.Key, ".")
}

// Condition is one `key value` pair of a required_if or required_unless rule.
type Condition struct {
	Key   string
	Value any
}

// Describe returns the definition's config description. Callers own the copy.
func (d *RegisteredDefinition) Describe() Description {
	fields := make([]Field, len(d.fields))
	for i, f := range d.fields {
		fields[i] = f.clone()
	}

	connectionRequiredKeys := make(map[string]map[string][]string, len(d.DestinationDefinition.ConnectionRequiredKeys))
	for sourceType, byMode := range d.DestinationDefinition.ConnectionRequiredKeys {
		connectionRequiredKeys[sourceType] = cloneStringSlices(byMode)
	}

	customValidators := make([]string, 0, len(d.ConfigValidateFuncs))
	for _, fn := range d.ConfigValidateFuncs {
		customValidators = append(customValidators, fn.Tag)
	}

	return Description{
		Type:                   d.Type,
		DisplayName:            d.DisplayName,
		APIType:                d.APIType,
		Version:                d.Version,
		SourceTypes:            d.SupportedSourceTypes(),
		ConnectionModes:        cloneStringSlices(d.DestinationDefinition.ConnectionModes),
		ConnectionRequiredKeys: connectionRequiredKeys,
		SecretKeys:             d.SecretKeys(),
		Fields:                 fields,
		Patterns:               maps.Clone(d.patterns),
		CustomValidators:       customValidators,
	}
}

func (f Field) clone() Field {
	f.RequiredIf = slices.Clone(f.RequiredIf)
	f.RequiredUnless = slices.Clone(f.RequiredUnless)
	f.RequiredWith = slices.Clone(f.RequiredWith)
	f.ExcludedWith = slices.Clone(f.ExcludedWith)
	f.CheckedBy = slices.Clone(f.CheckedBy)
	f.Enum = slices.Clone(f.Enum)
	f.OnlyForSourceTypes = slices.Clone(f.OnlyForSourceTypes)
	if f.Max != nil {
		limit := *f.Max
		f.Max = &limit
	}
	return f
}

func cloneStringSlices(in map[string][]string) map[string][]string {
	out := make(map[string][]string, len(in))
	for key, values := range in {
		out[key] = slices.Clone(values)
	}
	return out
}

var (
	connectionModeType    = reflect.TypeOf(common.ConnectionMode{})
	consentManagementType = reflect.TypeOf(common.ConsentManagement{})
)

// configWalker reads a config model's fields. It fails on any validate rule
// it cannot describe, so a rule added to a definition without teaching the
// walker about it fails registration instead of going undocumented.
type configWalker struct {
	customTags []string
	gated      map[string][]string
	defaults   map[string]any
	secretKeys []string
	fields     []Field
	patterns   map[string]funcs.PatternInfo
}

func describeConfig(def *DestinationDefinition, configType reflect.Type, gated map[string][]string, defaults map[string]any) ([]Field, map[string]funcs.PatternInfo, error) {
	w := &configWalker{
		gated:      gated,
		defaults:   defaults,
		secretKeys: def.SecretKeys,
		patterns:   map[string]funcs.PatternInfo{},
	}
	for _, fn := range def.ConfigValidateFuncs {
		w.customTags = append(w.customTags, fn.Tag)
	}

	if err := w.walkStruct(configType, "", nil, false); err != nil {
		return nil, nil, err
	}
	return w.fields, w.patterns, nil
}

// walkStruct describes typ's fields under prefix. gate is the source-type gate
// inherited from an enclosing gated block. alwaysChecked is set for a nested
// block that is not a pointer, which go-playground validates even when the
// spec leaves the block out.
func (w *configWalker) walkStruct(typ reflect.Type, prefix string, gate []string, alwaysChecked bool) error {
	typ = derefType(typ)
	for i := 0; i < typ.NumField(); i++ {
		structField := typ.Field(i)
		if !structField.IsExported() {
			continue
		}
		name, ok := mapstructureFieldTag(structField)
		if !ok {
			continue
		}

		key := name
		if prefix != "" {
			key = prefix + "." + name
		}
		if err := w.walkField(typ, structField, key, gate, alwaysChecked); err != nil {
			return fmt.Errorf("config key %q: %w", key, err)
		}
	}
	return nil
}

func (w *configWalker) walkField(owner reflect.Type, structField reflect.StructField, key string, gate []string, alwaysChecked bool) error {
	var (
		fieldType = derefType(structField.Type)
		dotted    = strings.ReplaceAll(key, "[]", "")
		field     = Field{Key: key, Type: describeType(structField.Type)}
	)

	if fieldGate, ok := w.gated[localKeyToPointer(dotted)]; ok {
		gate = fieldGate
	}
	field.OnlyForSourceTypes = slices.Clone(gate)
	field.Secret = slices.Contains(w.secretKeys, dotted)
	if value, ok := lookupDefault(w.defaults, dotted); ok {
		field.Default, field.HasDefault = value, true
	}

	ownRules, itemRules := splitDive(structField.Tag.Get("validate"))
	scope := ruleScope{owner: owner, prefix: parentKey(key), valueType: fieldType}
	for _, rule := range ownRules {
		if err := w.applyRule(&field, rule, scope); err != nil {
			return err
		}
	}
	if err := requiredMeansWritten(structField.Type, field, alwaysChecked); err != nil {
		return err
	}

	var itemStruct reflect.Type
	if fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Array {
		if elem := derefType(fieldType.Elem()); elem.Kind() == reflect.Struct {
			itemStruct = elem
		}
	}

	if len(itemRules) > 0 {
		if itemStruct != nil || (fieldType.Kind() != reflect.Slice && fieldType.Kind() != reflect.Array) {
			return fmt.Errorf("rules after dive apply only to a list of scalars")
		}
		scope.valueType = derefType(fieldType.Elem())
		for _, rule := range itemRules {
			if name, _, _ := strings.Cut(rule, "="); !describedPerItem(name) && !slices.Contains(w.customTags, name) {
				return fmt.Errorf("validate rule %q after dive would be described as a rule on the list; "+
					"only oneof, dynamic_or_oneof, pattern and dynamic_or_pattern apply to each item", name)
			}
			if err := w.applyRule(&field, rule, scope); err != nil {
				return err
			}
		}
	}

	switch {
	case fieldType.Kind() == reflect.Struct:
		if field.constrained() {
			w.fields = append(w.fields, field)
		}
		return w.walkStruct(fieldType, key, gate, structField.Type.Kind() != reflect.Pointer)
	case itemStruct != nil:
		if field.constrained() {
			w.fields = append(w.fields, field)
		}
		return w.walkStruct(itemStruct, key+"[]", gate, false)
	default:
		w.fields = append(w.fields, field)
		return nil
	}
}

// requiredMeansWritten fails a required rule the docs would misstate. The docs
// say a required key must be written, and a nested one once its parent is.
// go-playground instead checks a key inside a block that is not a pointer even
// when the block is left out, and rejects the zero value of a bool or number
// that is not a pointer, so `false` or `0` would fail a key the docs call
// written.
func requiredMeansWritten(goType reflect.Type, field Field, alwaysChecked bool) error {
	if field.Required && alwaysChecked {
		return fmt.Errorf("required inside a block that is not a pointer applies even when the spec leaves the block out; make the block a pointer")
	}
	requires := field.Required || len(field.RequiredIf) > 0 || len(field.RequiredUnless) > 0 || len(field.RequiredWith) > 0
	if requires && zeroIsAValue(goType) {
		return fmt.Errorf("a required rule on a %s that is not a pointer rejects %v, so it means non-zero, not written; make the field a pointer",
			goType.Kind(), reflect.Zero(goType).Interface())
	}
	return nil
}

func zeroIsAValue(typ reflect.Type) bool {
	switch typ.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// describedPerItem reports whether a rule after dive is one Describe can
// present as a rule on each item. Others, such as required or max, would read
// as rules on the list itself.
func describedPerItem(name string) bool {
	switch name {
	case "", "omitempty", "oneof", "dynamic_or_oneof", "pattern", "dynamic_or_pattern":
		return true
	default:
		return false
	}
}

// constrained reports whether a container carries a rule of its own, which is
// what earns it an entry next to the keys it holds.
func (f Field) constrained() bool {
	return f.Required || len(f.RequiredIf) > 0 || len(f.RequiredUnless) > 0 ||
		len(f.RequiredWith) > 0 || len(f.ExcludedWith) > 0 || len(f.CheckedBy) > 0 ||
		len(f.Enum) > 0 || f.Pattern != "" || f.Max != nil
}

// ruleScope is what a rule needs to resolve its parameters: go-playground
// resolves field names in cross-field rules against the struct holding the
// field, and an enum's values take the type of the value they constrain.
type ruleScope struct {
	owner     reflect.Type
	prefix    string
	valueType reflect.Type
}

func (w *configWalker) applyRule(field *Field, rule string, scope ruleScope) error {
	// go-playground reads | as "or", quoted values as one parameter, and 0x2C
	// and 0x7C as escaped commas and pipes. Describe splits parameters on
	// spaces only, so it refuses these rather than misread them.
	if strings.ContainsAny(rule, "|'") || strings.Contains(rule, "0x2C") || strings.Contains(rule, "0x7C") {
		return fmt.Errorf("validate rule %q uses |, a quote or an escaped character, which Describe cannot read", rule)
	}
	name, param, _ := strings.Cut(rule, "=")
	switch name {
	case "", "omitempty":
		return nil
	case "required":
		field.Required = true
	case "required_if", "required_unless":
		conditions, err := scope.conditions(param)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if name == "required_if" {
			field.RequiredIf = append(field.RequiredIf, conditions...)
		} else {
			field.RequiredUnless = append(field.RequiredUnless, conditions...)
		}
	case "required_with", "excluded_with":
		keys, err := scope.keys(strings.Fields(param))
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if name == "required_with" {
			field.RequiredWith = append(field.RequiredWith, keys...)
		} else {
			field.ExcludedWith = append(field.ExcludedWith, keys...)
		}
	case "oneof", "dynamic_or_oneof":
		for _, raw := range strings.Fields(param) {
			value, err := parseTypedValue(scope.valueType, raw)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			field.Enum = append(field.Enum, value)
		}
		if name == "dynamic_or_oneof" {
			field.OpenEnum, field.AcceptsTemplate = true, true
		}
	case "pattern", "dynamic_or_pattern":
		pattern, ok := funcs.LookupPattern(param)
		if !ok {
			return fmt.Errorf("%s: pattern %q is not registered", name, param)
		}
		w.patterns[param] = pattern
		field.Pattern = param
		if name == "dynamic_or_pattern" {
			field.AcceptsTemplate = true
		}
	case "max":
		limit, err := strconv.Atoi(param)
		if err != nil {
			return fmt.Errorf("max: invalid limit %q", param)
		}
		field.Max = &limit
	default:
		if !slices.Contains(w.customTags, name) {
			return fmt.Errorf("validate rule %q is neither described nor one of the definition's ConfigValidateFuncs", name)
		}
		field.CheckedBy = append(field.CheckedBy, name)
	}
	return nil
}

// conditions reads a required_if or required_unless parameter, a list of
// `Field value` pairs.
func (s ruleScope) conditions(param string) ([]Condition, error) {
	parts := strings.Fields(param)
	if len(parts) == 0 || len(parts)%2 != 0 {
		return nil, fmt.Errorf("want field and value pairs, got %q", param)
	}

	conditions := make([]Condition, 0, len(parts)/2)
	for i := 0; i < len(parts); i += 2 {
		key, fieldType, err := s.resolve(parts[i])
		if err != nil {
			return nil, err
		}
		value, err := parseTypedValue(fieldType, parts[i+1])
		if err != nil {
			return nil, fmt.Errorf("value for %s: %w", key, err)
		}
		conditions = append(conditions, Condition{Key: key, Value: value})
	}
	return conditions, nil
}

func (s ruleScope) keys(names []string) ([]string, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("names no field")
	}
	keys := make([]string, 0, len(names))
	for _, name := range names {
		key, _, err := s.resolve(name)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// resolve turns a Go field name, possibly dotted into nested structs, into
// the full local key and the field's type.
func (s ruleScope) resolve(goName string) (string, reflect.Type, error) {
	var (
		current = s.owner
		key     = s.prefix
	)
	for _, part := range strings.Split(goName, ".") {
		current = derefType(current)
		if current.Kind() != reflect.Struct {
			return "", nil, fmt.Errorf("field %q does not resolve to a config model field", goName)
		}
		structField, ok := current.FieldByName(part)
		if !ok {
			return "", nil, fmt.Errorf("field %q does not resolve to a config model field", goName)
		}
		name, ok := mapstructureFieldTag(structField)
		if !ok {
			return "", nil, fmt.Errorf("field %q has no mapstructure key", goName)
		}
		if key != "" {
			key += "."
		}
		key += name
		current = structField.Type
	}
	return key, derefType(current), nil
}

// parseTypedValue converts a tag value to the type of the value it applies
// to, so an integer enum stays integers in the output.
func parseTypedValue(typ reflect.Type, raw string) (any, error) {
	switch derefType(typ).Kind() {
	case reflect.Bool:
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid bool %q", raw)
		}
		return value, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid integer %q", raw)
		}
		return value, nil
	case reflect.Float32, reflect.Float64:
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number %q", raw)
		}
		return value, nil
	default:
		return raw, nil
	}
}

// splitDive separates the rules on a field from the rules after `dive`, which
// go-playground applies to each item of a list.
func splitDive(tag string) (own, items []string) {
	if tag == "" {
		return nil, nil
	}
	rules := strings.Split(tag, ",")
	if i := slices.Index(rules, "dive"); i >= 0 {
		return rules[:i], rules[i+1:]
	}
	return rules, nil
}

// parentKey is the key of the block holding key, which is where go-playground
// resolves a cross-field rule's field names.
func parentKey(key string) string {
	i := strings.LastIndex(key, ".")
	if i < 0 {
		return ""
	}
	return key[:i]
}

func describeType(typ reflect.Type) string {
	typ = derefType(typ)
	switch typ {
	case connectionModeType:
		return "map<sourceToken,string>"
	case consentManagementType:
		return "map<sourceToken,array<object>>"
	}

	switch typ.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Struct:
		return "object"
	case reflect.Slice, reflect.Array:
		return "array<" + describeType(typ.Elem()) + ">"
	case reflect.Map:
		return "map<string," + describeType(typ.Elem()) + ">"
	default:
		return "any"
	}
}

// lookupDefault finds a scalar default by dotted key in the nested defaults
// that buildConfigDefaults collects.
func lookupDefault(defaults map[string]any, dotted string) (any, bool) {
	var current any = defaults
	for _, segment := range strings.Split(dotted, ".") {
		block, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		if current, ok = block[segment]; !ok {
			return nil, false
		}
	}
	if _, isBlock := current.(map[string]any); isBlock {
		return nil, false
	}
	return current, true
}
