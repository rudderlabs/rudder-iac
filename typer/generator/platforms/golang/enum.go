package golang

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"slices"
	"strconv"
	"strings"

	"github.com/rudderlabs/rudder-iac/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/typer/plan"
)

// effectiveEnum is the enum of a property as Go generates it: the values its
// declared types allow (Enum Resolution).
type effectiveEnum struct {
	// base is the Go type of the non-null values: string, int64, float64 or
	// bool; empty when they have different JSON types or there are none.
	base string
	// values are the non-null values in tracking plan order, each Name
	// holding only the value's part of its constant name.
	values []GoEnumValue
	// null is set when null is one of the values.
	null bool
	// signature lists every value's Text, and null, in brackets: the JSON
	// array of the values, except that Go quotes non-printable characters and
	// invalid UTF-8 differently.
	signature string
}

// resolveEnum returns the effective enum of p, or nil when p has no enum.
// Numbers are compared exactly, so 1 and 1.0 are one value; the first of
// equal values is kept, and the rest are dropped.
func resolveEnum(p plan.Property) (*effectiveEnum, error) {
	if p.Config == nil || p.Config.Enum == nil {
		return nil, nil
	}
	// An explicit `enum: []` allows no value at all, so generating the bare
	// declared type would accept values the tracking plan rejects.
	if len(p.Config.Enum) == 0 {
		return nil, errors.New("enum has no values")
	}

	var (
		allowed []any // normalized
		raw     []any // as in the plan, for errors
		seen    = map[string]bool{}
	)
	for _, v := range p.Config.Enum {
		n, err := normalizeEnumValue(v)
		if err != nil {
			return nil, err
		}
		// A *big.Rat prints exactly, in lowest terms, so equal numbers share a
		// key; the type keeps the string "1/1" apart from the number 1.
		key := fmt.Sprintf("%T %v", n, n)
		if seen[key] || !allows(p.Types, n) {
			continue
		}
		seen[key] = true
		allowed, raw = append(allowed, n), append(raw, v)
	}
	if len(allowed) == 0 {
		return nil, fmt.Errorf("no enum value has a declared type (%s)", declaredTypes(p))
	}

	var (
		e        = &effectiveEnum{base: enumBase(p.Types, allowed)}
		texts    = make([]string, 0, len(allowed))
		literals = map[string]any{}
	)
	for i, n := range allowed {
		var v GoEnumValue
		switch n := n.(type) {
		case nil:
			e.null = true
			texts = append(texts, "null")
			continue
		case string:
			v = GoEnumValue{Name: valueName(n), Text: strconv.Quote(n)}
		case bool:
			text := strconv.FormatBool(n)
			v = GoEnumValue{Name: valueName(text), Text: text}
		case *big.Rat:
			var err error
			if v, err = numberValue(n, raw[i], e.base, literals); err != nil {
				return nil, err
			}
		}
		v.Expr = cmp.Or(v.Expr, v.Text)
		e.values = append(e.values, v)
		texts = append(texts, v.Text)
	}
	e.signature = "[" + strings.Join(texts, ",") + "]"
	return e, nil
}

// numberValue writes r as an int64 for an int64 enum, or for an integral
// value of a mixed enum that fits, and as a float64 otherwise. literals holds
// the plan value behind each number already named, since two values named
// alike would become duplicate constants: one float64 can represent two
// values, and in a mixed enum an int64 and a float64 can be the same number.
func numberValue(r *big.Rat, raw any, base string, literals map[string]any) (GoEnumValue, error) {
	var (
		v       GoEnumValue
		isInt64 = r.IsInt() && r.Num().IsInt64()
	)
	switch {
	case base == "int64" && !isInt64:
		return v, fmt.Errorf("enum value %s does not fit in int64", jsonText(raw))
	case isInt64 && base != "float64":
		v.Name = r.Num().String()
		v.Text, v.Expr = v.Name, "int64("+v.Name+")"
	default:
		f, _ := r.Float64()
		if f == 0 {
			f = 0 // Go constants have no negative zero, so -0 is the same float64 as 0.
		}
		if math.IsInf(f, 0) {
			return v, fmt.Errorf("enum value %s does not fit in float64", jsonText(raw))
		}
		v.Name = strconv.FormatFloat(f, 'f', -1, 64)
		v.Text = v.Name
		if !strings.Contains(v.Text, ".") {
			// 'f' writes 1e300 as 301 digits, an integer constant too large for gc.
			v.Text = strconv.FormatFloat(f, 'g', -1, 64)
			// A mixed enum's value is untyped; an integer literal would default to int.
			v.Expr = "float64(" + v.Text + ")"
		}
	}
	if first, ok := literals[v.Name]; ok {
		return v, fmt.Errorf("enum values %s and %s are the same float64", jsonText(first), jsonText(raw))
	}
	literals[v.Name] = raw
	v.Name = strings.NewReplacer("-", "Neg", ".", "_").Replace(v.Name)
	return v, nil
}

// normalizeEnumValue returns a number as its exact value, so that equal
// numbers compare equal however the plan writes them: 1, 1.0 or
// json.Number("1e0").
func normalizeEnumValue(v any) (any, error) {
	var r *big.Rat
	switch v := v.(type) {
	case nil, string, bool:
		return v, nil
	case int:
		r = big.NewRat(int64(v), 1)
	case int64:
		r = big.NewRat(v, 1)
	case float64:
		r = new(big.Rat).SetFloat64(v) // nil for NaN and ±Inf
	case json.Number:
		r, _ = new(big.Rat).SetString(string(v))
	}
	if r == nil {
		return nil, fmt.Errorf("enum value %s is not a string, finite number, boolean or null", jsonText(v))
	}
	return r, nil
}

// jsonText writes an enum value for an error as JSON, as tracking plans write
// it, or as Go prints it when JSON cannot hold it (NaN and ±Inf).
func jsonText(v any) string {
	if b, err := json.Marshal(v); err == nil {
		return string(b)
	}
	return fmt.Sprint(v)
}

// allows reports whether the declared types allow the normalized value v.
// With no declared type every value is allowed.
func allows(types []plan.PropertyType, v any) bool {
	if len(types) == 0 {
		return true
	}
	switch v := v.(type) {
	case nil:
		return declares(types, plan.PrimitiveTypeNull)
	case string:
		return declares(types, plan.PrimitiveTypeString)
	case bool:
		return declares(types, plan.PrimitiveTypeBoolean)
	case *big.Rat:
		return declares(types, plan.PrimitiveTypeNumber) || declares(types, plan.PrimitiveTypeInteger) && v.IsInt()
	}
	return false
}

func declares(types []plan.PropertyType, t plan.PrimitiveType) bool {
	return slices.Contains(types, plan.PropertyType(t))
}

// enumBase picks the Go type of the non-null values from the base-type table
// (Enum Resolution). Numbers are float64 unless integer is declared and number
// is not, in which case allows has kept only integral ones.
func enumBase(types []plan.PropertyType, values []any) string {
	var base string
	for _, v := range values {
		var b string
		switch v.(type) {
		case nil:
			continue
		case string:
			b = "string"
		case bool:
			b = "bool"
		default:
			b = "float64"
		}
		if base != "" && base != b {
			return ""
		}
		base = b
	}
	if base == "float64" && declares(types, plan.PrimitiveTypeInteger) && !declares(types, plan.PrimitiveTypeNumber) {
		return "int64"
	}
	return base
}

// addEnum adds the type of prop, whose effective enum is e: the defined enum
// type, a Nullable of it when null is also a value, or Null when null is the
// only value.
func addEnum(ctx *GoContext, registry *core.NameRegistry, name, lead string, prop plan.Property, e *effectiveEnum) error {
	switch {
	case len(e.values) == 0:
		ctx.UsesNull = true
		ctx.PropertyTypes = append(ctx.PropertyTypes, GoPropertyType{Alias: &GoTypeAlias{
			Name: name,
			Doc:  withDescription(lead+".", prop.Description),
			Type: "Null",
		}})
		return nil
	case !e.null:
		ctx.PropertyTypes = append(ctx.PropertyTypes, GoPropertyType{Enum: newGoEnum(name, lead, prop.Description, e)})
		return nil
	}

	// Registered right after its owner, as an item union is.
	enumName, err := registry.RegisterName("enum:"+name, packageScope, name+"Value")
	if err != nil {
		return fmt.Errorf("registering the enum type of %s: %w", name, err)
	}
	texts := make([]string, len(e.values))
	for i, v := range e.values {
		texts[i] = v.Text
	}
	ctx.UsesNullable = true
	ctx.PropertyTypes = append(ctx.PropertyTypes,
		GoPropertyType{Enum: newGoEnum(enumName, fmt.Sprintf("%s is the enum type of the property %s", enumName, strconv.Quote(prop.Name)), "", e)},
		GoPropertyType{Alias: &GoTypeAlias{
			Name: name,
			Doc:  withDescription(fmt.Sprintf("%s: %s or null.", lead, strings.Join(texts, ", ")), prop.Description),
			Type: "Nullable[" + enumName + "]",
		}},
	)
	return nil
}

// newGoEnum lays out e as the enum type name. lead starts its doc comment.
func newGoEnum(name, lead, description string, e *effectiveEnum) *GoEnum {
	enum := &GoEnum{
		Name:       name,
		Doc:        withDescription(lead+".", description),
		Base:       e.base,
		ConstDoc:   "Allowed values of " + name + ".",
		ValuesFunc: name + "Values",
		Values:     make([]GoEnumValue, len(e.values)),
	}
	if e.null {
		enum.ConstDoc = "Allowed non-null values of " + name + "."
	}
	if e.base == "" {
		enum.Doc = withDescription(lead+", whose allowed values have different JSON types.\n"+
			"Values can only be obtained from the functions below; the zero value cannot be sent.", description)
	}
	for i, v := range e.values {
		v.Name = name + v.Name
		enum.Values[i] = v
	}
	return enum
}
