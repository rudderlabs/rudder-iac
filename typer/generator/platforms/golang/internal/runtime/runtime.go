package runtime

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"reflect"
	"slices"

	analytics "github.com/rudderlabs/analytics-go/v4"
)

// applyOptions applies opts and copies every caller-owned value they hold, so
// the SDK, which serializes later on another goroutine, never shares memory
// with the caller. The fields it does not copy are encoded once, so a value
// the SDK could not send is reported here rather than in Config.Callback.
func applyOptions(opts []Option) (callOptions, error) {
	var o callOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	var err error
	if o.integrations, err = snapshot(o.integrations); err != nil {
		return o, fmt.Errorf("integrations: %w", err)
	}
	if _, err := o.timestamp.MarshalJSON(); err != nil {
		return o, fmt.Errorf("%w: timestamp: %w", ErrInvalidValue, err)
	}
	if o.context == nil {
		return o, nil
	}
	ctx := *o.context
	if ctx.Extra, err = snapshot(ctx.Extra); err != nil {
		return o, fmt.Errorf("context.Extra: %w", err)
	}
	if ctx.Traits, err = snapshot(ctx.Traits); err != nil {
		return o, fmt.Errorf("context.Traits: %w", err)
	}
	ctx.IP = slices.Clone(ctx.IP)
	fields := ctx
	fields.Extra, fields.Traits = nil, nil
	if _, err := json.Marshal(fields); err != nil {
		return o, fmt.Errorf("%w: context: %w", ErrInvalidValue, err)
	}
	o.context = &ctx
	return o, nil
}

// analyticsContext builds a fresh context for the call, because the SDK
// writes into the context it is given. contextTraits (from a context.traits
// identify rule) win over caller-supplied Context.Traits keys.
func (o callOptions) analyticsContext(contextTraits analytics.Traits) *analytics.Context {
	var ctx analytics.Context
	if o.context != nil {
		ctx = *o.context
	}
	extra := make(map[string]any, len(ctx.Extra)+1)
	maps.Copy(extra, ctx.Extra)
	extra["ruddertyper"] = rudderTyperContext()
	ctx.Extra = extra
	if len(contextTraits) > 0 {
		traits := make(analytics.Traits, len(ctx.Traits)+len(contextTraits))
		maps.Copy(traits, ctx.Traits)
		maps.Copy(traits, contextTraits)
		ctx.Traits = traits
	}
	return &ctx
}

// wireObject is implemented by every generated struct, and wireValuer by the
// other generated types with a wire form of their own: unions, mixed enums,
// Nullable and Null.
type (
	wireObject interface{ toMap() map[string]any }
	wireValuer interface{ wireValue() (any, error) }
)

// maxDepth bounds how deeply a value may nest. Real events stay far below it;
// a value that contains itself reaches it and fails instead of overflowing
// the stack.
const maxDepth = 1000

// snapshot converts v into what the SDK receives: new maps and slices holding
// strings, bools, numbers and nil. The SDK serializes later on another
// goroutine, so nothing the caller owns may reach it, and converting now
// reports a value that cannot be sent before Enqueue.
func snapshot(v any) (map[string]any, error) {
	w, err := wire(reflect.ValueOf(v), 0)
	if err != nil {
		return nil, err
	}
	m, _ := w.(map[string]any)
	return m, nil
}

// marshal is the MarshalJSON of every generated type, so json.Marshal and the
// event methods produce the same JSON.
func marshal(v any) ([]byte, error) {
	w, err := wire(reflect.ValueOf(v), 0)
	if err != nil {
		return nil, err
	}
	return json.Marshal(w)
}

// wire returns the JSON value of rv as new plain Go values. Nil slices and
// maps become [] and {}, so a Go zero value is never sent as null by
// accident; null comes only from nil pointers and interfaces, Null and an
// invalid Nullable.
func wire(rv reflect.Value, depth int) (any, error) {
	if depth > maxDepth {
		return nil, fmt.Errorf("%w: value nests more than %d levels deep or contains itself", ErrInvalidValue, maxDepth)
	}
	depth++
	if !rv.IsValid() {
		return nil, nil
	}
	if k := rv.Kind(); (k == reflect.Pointer || k == reflect.Interface) && isNil(rv.Interface()) {
		// toMap already omits or rejects nil variants in fields, so this is a
		// nil variant array item, which has no valid wire form.
		if k == reflect.Interface && rv.Type().Implements(reflect.TypeOf((*wireObject)(nil)).Elem()) {
			return nil, fmt.Errorf("%w: nil %s item", ErrInvalidValue, rv.Type())
		}
		return nil, nil
	}
	switch v := rv.Interface().(type) {
	case wireObject:
		return wire(reflect.ValueOf(v.toMap()), depth)
	case wireValuer:
		x, err := v.wireValue()
		if err != nil {
			return nil, err
		}
		return wire(reflect.ValueOf(x), depth)
	case json.Number, json.Marshaler, encoding.TextMarshaler:
		return encode(v)
	}
	// encoding/json also calls pointer-receiver marshalers on addressable
	// values, such as slice elements.
	if rv.CanAddr() {
		switch p := rv.Addr().Interface().(type) {
		case json.Marshaler, encoding.TextMarshaler:
			return encode(p)
		}
	}
	switch rv.Kind() {
	case reflect.Interface, reflect.Pointer:
		return wire(rv.Elem(), depth)
	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String {
			return foreign(rv, depth)
		}
		m := make(map[string]any, rv.Len())
		for it := rv.MapRange(); it.Next(); {
			x, err := wire(it.Value(), depth)
			if err != nil {
				return nil, err
			}
			m[it.Key().String()] = x
		}
		return m, nil
	case reflect.Slice, reflect.Array:
		if rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8 {
			return encode(rv.Interface()) // base64, as encoding/json sends []byte
		}
		s := make([]any, rv.Len())
		for i := range s {
			x, err := wire(rv.Index(i), depth)
			if err != nil {
				return nil, err
			}
			s[i] = x
		}
		return s, nil
	case reflect.Struct:
		return foreign(rv, depth)
	case reflect.String:
		return rv.String(), nil
	case reflect.Bool:
		return rv.Bool(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return rv.Uint(), nil
	case reflect.Float32, reflect.Float64:
		f := rv.Float()
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return nil, fmt.Errorf("%w: %v is not a JSON number", ErrInvalidValue, f)
		}
		if rv.Kind() == reflect.Float32 {
			return float32(f), nil
		}
		return f, nil
	}
	// Channels, functions and complex numbers: encoding/json names the error.
	return encode(rv.Interface())
}

// foreign converts a caller-defined struct, or a map with non-string keys,
// with encoding/json, whose rules for them this file does not repeat. The
// values inside are walked first and the result dropped: a generated value
// in there is encoded by its own MarshalJSON, which starts a new walk, so
// only this walk can stop a cycle that runs through it.
func foreign(rv reflect.Value, depth int) (any, error) {
	if rv.Kind() == reflect.Map {
		for it := rv.MapRange(); it.Next(); {
			if _, err := wire(it.Value(), depth); err != nil {
				return nil, err
			}
		}
		return encode(rv.Interface())
	}
	// VisibleFields includes the fields encoding/json promotes from embedded
	// structs; those behind a nil embedded pointer have no value to walk.
	for _, f := range reflect.VisibleFields(rv.Type()) {
		fv, err := rv.FieldByIndexErr(f.Index)
		if err != nil || !f.IsExported() || f.Tag.Get("json") == "-" {
			continue
		}
		if _, err := wire(fv, depth); err != nil {
			return nil, err
		}
	}
	// encoding/json calls pointer-receiver marshalers only on addressable values.
	if rv.CanAddr() {
		rv = rv.Addr()
	}
	return encode(rv.Interface())
}

// encode converts v through encoding/json, for values whose JSON form only
// encoding/json defines. Numbers inside them come back as json.Number.
func encode(v any) (any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidValue, err)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var x any
	if err := dec.Decode(&x); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidValue, err)
	}
	return x, nil
}

// isNil reports whether v is nil or a nil pointer stored in an interface.
func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Pointer && rv.IsNil()
}
