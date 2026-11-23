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
		return o, fmt.Errorf("copying integrations: %w", err)
	}
	if _, err := o.timestamp.MarshalJSON(); err != nil {
		return o, fmt.Errorf("checking timestamp: %w: %w", ErrInvalidValue, err)
	}
	if o.context == nil {
		return o, nil
	}
	ctx := *o.context
	if ctx.Extra, err = snapshot(ctx.Extra); err != nil {
		return o, fmt.Errorf("copying context.Extra: %w", err)
	}
	if ctx.Traits, err = snapshot(ctx.Traits); err != nil {
		return o, fmt.Errorf("copying context.Traits: %w", err)
	}
	ctx.IP = slices.Clone(ctx.IP)
	fields := ctx
	fields.Extra, fields.Traits = nil, nil
	if _, err := json.Marshal(fields); err != nil {
		return o, fmt.Errorf("checking context: %w: %w", ErrInvalidValue, err)
	}
	o.context = &ctx
	return o, nil
}

// analyticsContext builds a fresh context for the call, because the SDK
// writes into the context it is given.
func (o callOptions) analyticsContext() *analytics.Context {
	var ctx analytics.Context
	if o.context != nil {
		ctx = *o.context
	}
	extra := make(map[string]any, len(ctx.Extra)+1)
	maps.Copy(extra, ctx.Extra)
	extra["ruddertyper"] = rudderTyperContext()
	ctx.Extra = extra
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

// errTooDeep is a function because the three sentinels are the only package
// variables a generated file may declare.
func errTooDeep() error {
	return fmt.Errorf("%w: value nests more than %d levels deep or contains itself", ErrInvalidValue, maxDepth)
}

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
		return nil, errTooDeep()
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
	v := rv.Interface()
	if _, ok := v.(json.Number); !ok && rv.CanAddr() {
		// encoding/json calls the marshalers of *T on an addressable T, such
		// as a slice element, and they include T's own.
		v = rv.Addr().Interface()
	}
	// A caller's struct that embeds a generated type has its toMap or
	// wireValue too, but encoding/json encodes it by the caller's own
	// MarshalJSON, if it has one. Walking the hook's result first still stops
	// a cycle through the generated part.
	switch v := v.(type) {
	case wireObject:
		w, err := wire(reflect.ValueOf(v.toMap()), depth)
		if err != nil || !embeds(v) {
			return w, err
		}
		return encode(v)
	case wireValuer:
		x, err := v.wireValue()
		if err != nil {
			return nil, err
		}
		w, err := wire(reflect.ValueOf(x), depth)
		if err != nil || !embeds(v) {
			return w, err
		}
		return encode(v)
	case json.Number, json.Marshaler, encoding.TextMarshaler:
		return encode(v)
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
	if err := walkFields(rv, depth); err != nil {
		return nil, err
	}
	// encoding/json calls pointer-receiver marshalers only on addressable values.
	if rv.CanAddr() {
		rv = rv.Addr()
	}
	return encode(rv.Interface())
}

// walkFields walks every field of struct rv that encoding/json may encode,
// including those of the structs embedded in it, exported or not. Unlike
// reflect.VisibleFields, it hides no field for sharing a Go name with
// another, because encoding/json compares JSON names, so it can walk more
// than encoding/json encodes but never less. Zero and empty fields hold
// nothing to walk, and walking them could reject a value that omitzero or
// omitempty leaves out.
func walkFields(rv reflect.Value, depth int) error {
	if depth > maxDepth {
		return errTooDeep()
	}
	for i := 0; i < rv.NumField(); i++ {
		f, fv := rv.Type().Field(i), rv.Field(i)
		tag := f.Tag.Get("json")
		if tag == "-" || fv.IsZero() || (fv.Kind() == reflect.Slice || fv.Kind() == reflect.Map) && fv.Len() == 0 {
			continue
		}
		var err error
		switch s := reflect.Indirect(fv); {
		case f.Anonymous && s.Kind() == reflect.Struct && (!f.IsExported() || tag == "" || tag[0] == ','):
			// encoding/json promotes the fields of an embedded struct with no
			// JSON name, and an unexported one cannot reach wire. Counted as
			// a level: an embedded pointer can lead back to its own struct.
			err = walkFields(s, depth+1)
		case f.IsExported():
			_, err = wire(fv, depth)
		}
		if err != nil {
			return err
		}
	}
	return nil
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

// embeds reports whether v is a struct, or points to one, with an embedded
// field. Generated types embed nothing, so a v with toMap or wireValue that
// embeds is a caller's struct.
func embeds(v any) bool {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	for i := 0; t.Kind() == reflect.Struct && i < t.NumField(); i++ {
		if t.Field(i).Anonymous {
			return true
		}
	}
	return false
}

// isNil reports whether v is nil or a nil pointer stored in an interface.
func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Pointer && rv.IsNil()
}
