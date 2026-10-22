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
	// "headers[].to", and "<sourceToken>" a key of a map keyed by source
	// token, as in "consent_management.<sourceToken>[].provider".
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
	// ExcludedIf forbids the key when every condition holds.
	ExcludedIf []Condition
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
	// Min and Max are the `min` and `max` rules: a length for strings and
	// lists, a value for numbers.
	Min *int
	Max *int
	// OnlyForSourceTypes lists the source tokens the key is valid for; empty
	// means every source type.
	OnlyForSourceTypes []string
	// Secret marks a SecretKeys path.
	Secret bool
	// Unique is set when no two items of the enclosing list may hold the same
	// value.
	Unique bool
}

// Nested reports whether the key sits inside a block or a list item.
func (f Field) Nested() bool {
	return strings.Contains(f.Key, ".")
}

// Condition is one `key value` pair of a required_if, required_unless or
// excluded_if rule.
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
	f.ExcludedIf = slices.Clone(f.ExcludedIf)
	f.CheckedBy = slices.Clone(f.CheckedBy)
	f.Enum = slices.Clone(f.Enum)
	f.OnlyForSourceTypes = slices.Clone(f.OnlyForSourceTypes)
	f.Min = cloneInt(f.Min)
	f.Max = cloneInt(f.Max)
	return f
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
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

// sourceTokenKey stands for a key of a map keyed by source token.
const sourceTokenKey = "<sourceToken>"

// configWalker reads a config model's fields. It fails on any validate rule
// it cannot describe, so a rule added to a definition without teaching the
// walker about it fails registration instead of going undocumented.
type configWalker struct {
	customTags       []string
	gated            map[string][]string
	defaults         map[string]any
	secretKeys       []string
	sourceTypes      []string
	consentOverrides map[string]common.ConsentValidator
	fields           []Field
	patterns         map[string]funcs.PatternInfo
}

func describeConfig(def *DestinationDefinition, configType reflect.Type, gated map[string][]string, defaults map[string]any) ([]Field, map[string]funcs.PatternInfo, error) {
	w := &configWalker{
		gated:            gated,
		defaults:         defaults,
		secretKeys:       def.SecretKeys,
		sourceTypes:      def.SourceTypes,
		consentOverrides: def.ConsentValidationOverrides,
		patterns:         map[string]funcs.PatternInfo{},
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
			return fmt.Errorf("reading config key %q: %w", key, err)
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
		field.Default, field.HasDefault = describedDefault(fieldType, value), true
	}

	ownRules, itemRules := splitDive(structField.Tag.Get("validate"))
	if err := requiredBeforeOmitempty(ownRules); err != nil {
		return err
	}
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
	case fieldType == consentManagementType:
		w.fields = append(w.fields, field)
		return w.describeConsentEntries(key, gate)
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

// describeConsentEntries describes the items of consent_management, which
// common.ValidateConsentEntries checks in Go code rather than through
// validate tags. A source token whose definition overrides that check is
// left out, since only the override's code says what it accepts.
func (w *configWalker) describeConsentEntries(key string, gate []string) error {
	gate, described := w.consentGate(gate)
	if !described {
		return nil
	}

	pattern, ok := funcs.LookupPattern(common.ConsentValuePattern)
	if !ok {
		return fmt.Errorf("pattern %q is not registered", common.ConsentValuePattern)
	}
	w.patterns[common.ConsentValuePattern] = pattern

	var (
		item     = key + "." + sourceTokenKey + "[]"
		provider = item + ".provider"
	)
	w.fields = append(w.fields,
		Field{
			Key:                provider,
			Type:               "string",
			Required:           true,
			Enum:               anyValues(common.ConsentProviders()),
			Unique:             true,
			OnlyForSourceTypes: slices.Clone(gate),
		},
		Field{
			Key:                item + ".resolution_strategy",
			Type:               "string",
			RequiredIf:         []Condition{{Key: provider, Value: common.ConsentProviderCustom}},
			Enum:               anyValues(common.ConsentResolutionStrategies()),
			OnlyForSourceTypes: slices.Clone(gate),
		},
		Field{
			Key:                item + ".consents",
			Type:               "array<string>",
			Pattern:            common.ConsentValuePattern,
			AcceptsTemplate:    true,
			OnlyForSourceTypes: slices.Clone(gate),
		},
	)
	return nil
}

// consentGate narrows gate to the source tokens the canonical consent check
// applies to. It reports false when every token has an override.
func (w *configWalker) consentGate(gate []string) ([]string, bool) {
	if len(w.consentOverrides) == 0 {
		return gate, true
	}
	tokens := gate
	if len(tokens) == 0 {
		tokens = w.sourceTypes
	}
	var canonical []string
	for _, token := range tokens {
		if _, overridden := w.consentOverrides[token]; !overridden {
			canonical = append(canonical, token)
		}
	}
	return canonical, len(canonical) > 0
}

func anyValues(values []string) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}

// requiredBeforeOmitempty fails a required rule placed after omitempty. On an
// empty value go-playground stops at omitempty and skips every rule after it,
// so such a rule never rejects a missing key the docs would call required.
func requiredBeforeOmitempty(rules []string) error {
	omitted := false
	for _, rule := range rules {
		name, _, _ := strings.Cut(rule, "=")
		switch {
		case name == "omitempty":
			omitted = true
		case omitted && isRequiredRule(name):
			return fmt.Errorf("validate rule %q comes after omitempty, which skips it on an empty value; put it before omitempty", name)
		}
	}
	return nil
}

func isRequiredRule(name string) bool {
	switch name {
	case "required", "required_if", "required_unless", "required_with":
		return true
	default:
		return false
	}
}

// requiredMeansWritten fails a required rule the docs would misstate. The docs
// say a required key must be written, and a nested one once its parent is.
// go-playground instead checks a key inside a block that is not a pointer even
// when the block is left out, and rejects the zero value of a bool or number
// that is not a pointer, so `false` or `0` would fail a key the docs call
// written. On a block that is not a pointer it skips a bare required, because
// validate does not enable WithRequiredStructEnabled, and reads the other
// required rules as "not empty".
func requiredMeansWritten(goType reflect.Type, field Field, alwaysChecked bool) error {
	if field.Required && alwaysChecked {
		return fmt.Errorf("required inside a block that is not a pointer applies even when the spec leaves the block out; make the block a pointer")
	}
	requires := field.Required || len(field.RequiredIf) > 0 || len(field.RequiredUnless) > 0 || len(field.RequiredWith) > 0
	switch {
	case !requires:
		return nil
	case goType.Kind() == reflect.Struct:
		return fmt.Errorf("a required rule on a block that is not a pointer is skipped or rejects an empty block, so it does not mean written; make the block a pointer")
	case zeroIsAValue(goType):
		return fmt.Errorf("a required rule on a %s that is not a pointer rejects %v, so it means non-zero, not written; make the field a pointer",
			goType.Kind(), reflect.Zero(goType).Interface())
	default:
		return nil
	}
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
		len(f.RequiredWith) > 0 || len(f.ExcludedWith) > 0 || len(f.ExcludedIf) > 0 || len(f.CheckedBy) > 0 ||
		len(f.Enum) > 0 || f.Pattern != "" || f.Min != nil || f.Max != nil
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
	if err := ruleFitsValue(name, scope.valueType); err != nil {
		return err
	}
	switch name {
	case "", "omitempty":
		return nil
	case "required":
		field.Required = true
	case "required_if", "required_unless", "excluded_if":
		// go-playground checks each such rule on its own, so two of them combine
		// differently from the conditions of one, and Field has room for one.
		target := field.conditionsOf(name)
		if len(*target) > 0 {
			return fmt.Errorf("validate rule %q appears twice on one key; go-playground checks each on its own, "+
				"which the docs would read as one rule, so keep one", name)
		}
		conditions, err := scope.conditions(param)
		if err != nil {
			return fmt.Errorf("reading %s rule: %w", name, err)
		}
		*target = conditions
	case "required_with", "excluded_with":
		keys, err := scope.keys(strings.Fields(param))
		if err != nil {
			return fmt.Errorf("reading %s rule: %w", name, err)
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
				return fmt.Errorf("reading %s rule: %w", name, err)
			}
			field.Enum = append(field.Enum, value)
		}
		if name == "dynamic_or_oneof" {
			field.OpenEnum, field.AcceptsTemplate = true, true
		}
	case "pattern", "dynamic_or_pattern":
		pattern, ok := funcs.LookupPattern(param)
		if !ok {
			return fmt.Errorf("reading %s rule: pattern %q is not registered", name, param)
		}
		w.patterns[param] = pattern
		field.Pattern = param
		if name == "dynamic_or_pattern" {
			field.AcceptsTemplate = true
		}
	case "min", "max":
		limit, err := parseLimit(scope.valueType, param)
		if err != nil {
			return fmt.Errorf("reading %s rule: %w", name, err)
		}
		if name == "min" {
			field.Min = &limit
		} else {
			field.Max = &limit
		}
	default:
		if !slices.Contains(w.customTags, name) {
			return fmt.Errorf("validate rule %q is neither described nor one of the definition's ConfigValidateFuncs", name)
		}
		field.CheckedBy = append(field.CheckedBy, name)
	}
	return nil
}

func (f *Field) conditionsOf(rule string) *[]Condition {
	switch rule {
	case "required_if":
		return &f.RequiredIf
	case "required_unless":
		return &f.RequiredUnless
	default:
		return &f.ExcludedIf
	}
}

// ruleFitsValue fails a rule on a kind of value it does not check the way the
// docs state it. go-playground's oneof panics on anything but a string or an
// integer, and its min and max on a bool or a block. pattern and the
// dynamic_or_ rules read only strings: on other values pattern and
// dynamic_or_pattern always fail and dynamic_or_oneof always passes.
func ruleFitsValue(name string, valueType reflect.Type) error {
	kind := derefType(valueType).Kind()
	switch name {
	case "oneof":
		if kind == reflect.String || isInteger(kind) {
			return nil
		}
		return fmt.Errorf("validate rule %q on a %s makes go-playground panic; it takes only strings and integers", name, kind)
	case "pattern", "dynamic_or_pattern", "dynamic_or_oneof":
		if kind == reflect.String {
			return nil
		}
		return fmt.Errorf("validate rule %q on a %s does not check the value, since it reads only strings", name, kind)
	case "min", "max":
		switch {
		case kind == reflect.String, isInteger(kind), kind == reflect.Float32, kind == reflect.Float64,
			kind == reflect.Slice, kind == reflect.Array, kind == reflect.Map:
			return nil
		default:
			return fmt.Errorf("validate rule %q on a %s makes go-playground panic", name, kind)
		}
	default:
		return nil
	}
}

func isInteger(kind reflect.Kind) bool {
	return isSigned(kind) || isUnsigned(kind)
}

func isSigned(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	default:
		return false
	}
}

func isUnsigned(kind reflect.Kind) bool {
	switch kind {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

// parseLimit reads a min or max limit. go-playground reads it with base
// prefixes and panics on a negative limit for an unsigned value, so the limit
// must be plain base 10, and 0 or more on an unsigned value.
func parseLimit(valueType reflect.Type, raw string) (int, error) {
	limit, err := strconv.Atoi(raw)
	if err != nil || strconv.Itoa(limit) != raw {
		return 0, fmt.Errorf("want a plain base-10 integer, got %q", raw)
	}
	if kind := derefType(valueType).Kind(); limit < 0 && isUnsigned(kind) {
		return 0, fmt.Errorf("want 0 or more for a %s, got %q", kind, raw)
	}
	return limit, nil
}

// conditions reads a required_if, required_unless or excluded_if parameter, a
// list of `Field value` pairs.
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
			return nil, fmt.Errorf("parsing the value for %s: %w", key, err)
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
// to, so an integer enum stays integers in the output. It accepts only the
// spellings go-playground reads the same way. It compares a bool condition
// with the text "true", so a bool condition takes only true or false. It
// parses an integer condition or limit with base prefixes and compares a
// oneof value with the integer's base-10 text, so an integer must be plain
// base 10, and it panics on a negative value for an unsigned integer. A value
// of any other kind, such as a list compared by length, is refused.
func parseTypedValue(typ reflect.Type, raw string) (any, error) {
	switch kind := derefType(typ).Kind(); {
	case kind == reflect.String:
		return raw, nil
	case kind == reflect.Bool:
		if raw != "true" && raw != "false" {
			return nil, fmt.Errorf("want true or false, got %q", raw)
		}
		return raw == "true", nil
	case isInteger(kind):
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || strconv.FormatInt(value, 10) != raw {
			return nil, fmt.Errorf("want a plain base-10 integer, got %q", raw)
		}
		if value < 0 && isUnsigned(kind) {
			return nil, fmt.Errorf("want 0 or more for a %s, got %q", kind, raw)
		}
		return value, nil
	case kind == reflect.Float32 || kind == reflect.Float64:
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number %q", raw)
		}
		return value, nil
	default:
		return nil, fmt.Errorf("cannot compare a %s with %q", kind, raw)
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

// describedDefault turns an integer default back into int64. buildConfigDefaults
// keeps integers as float64 to compare with decoded config, while Enum and
// condition values hold int64, and a Go caller should be able to compare them.
func describedDefault(typ reflect.Type, value any) any {
	if number, ok := value.(float64); ok && describeType(typ) == "integer" {
		return int64(number)
	}
	return value
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
