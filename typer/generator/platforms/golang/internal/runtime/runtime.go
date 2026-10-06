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
		return nil, tooDeep()
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
	if addrMarshaler(rv) {
		return encode(rv.Addr().Interface())
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
// with encoding/json, whose rules for them this file does not repeat. It
// encodes a copy in which nil slices and maps are empty, so the nil rule
// holds inside them too. Building the copy visits every value encoding/json
// will reach: a generated value in there is encoded by its own MarshalJSON,
// which starts a new walk, so only this walk can stop a cycle that runs
// through it.
func foreign(rv reflect.Value, depth int) (any, error) {
	c, err := normalize(rv, depth-1) // wire has already counted rv
	if err != nil {
		return nil, err
	}
	// encoding/json calls pointer-receiver marshalers only on addressable
	// values, so the copy is encoded as addressable as rv is.
	if rv.Kind() == reflect.Struct && rv.CanAddr() {
		c = c.Addr()
	}
	return encode(c.Interface())
}

// normalize returns a copy of rv in which nil slices and maps are empty,
// except []byte and maps with non-string keys, which encoding/json sends as
// null. Generated values and marshalers are kept, since they encode
// themselves; generated values are walked first to catch cycles.
func normalize(rv reflect.Value, depth int) (reflect.Value, error) {
	k, t := rv.Kind(), rv.Type()
	if (k == reflect.Pointer || k == reflect.Interface) && rv.IsNil() {
		_, err := wire(rv, depth) // rejects a nil variant
		return rv, err
	}
	switch rv.Interface().(type) {
	case wireObject, wireValuer:
		_, err := wire(rv, depth)
		return rv, err
	case json.Marshaler, encoding.TextMarshaler:
		return rv, nil
	}
	if addrMarshaler(rv) {
		return rv, nil
	}
	if depth > maxDepth {
		return rv, tooDeep()
	}
	depth++
	switch k {
	case reflect.Interface:
		return normalize(rv.Elem(), depth)
	case reflect.Pointer:
		e, err := normalize(rv.Elem(), depth)
		if err != nil {
			return rv, err
		}
		c := reflect.New(t.Elem())
		c.Elem().Set(e)
		return c, nil
	case reflect.Map:
		if rv.IsNil() && t.Key().Kind() != reflect.String {
			return rv, nil
		}
		c := reflect.MakeMapWithSize(t, rv.Len())
		for it := rv.MapRange(); it.Next(); {
			e, err := normalize(it.Value(), depth)
			if err != nil {
				return rv, err
			}
			c.SetMapIndex(it.Key(), e)
		}
		return c, nil
	case reflect.Slice, reflect.Array:
		if k == reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
			return rv, nil
		}
		c := reflect.New(t).Elem()
		if k == reflect.Slice {
			c.Set(reflect.MakeSlice(t, rv.Len(), rv.Len()))
		}
		for i := 0; i < rv.Len(); i++ {
			e, err := normalize(rv.Index(i), depth)
			if err != nil {
				return rv, err
			}
			c.Index(i).Set(e)
		}
		return c, nil
	case reflect.Struct:
		c := reflect.New(t).Elem()
		c.Set(rv)
		return c, normalizeFields(c, rv, depth)
	}
	return rv, nil
}

// normalizeFields normalizes into dst the fields of struct src that
// encoding/json writes: the exported ones, including those it promotes from
// an embedded struct of an unexported type. Fields are read from src, so a
// field is as addressable as encoding/json will find it. dst is invalid
// behind an embedded pointer to such a struct: the pointer cannot be
// replaced and what it points to belongs to the caller, so the fields there
// are only checked.
func normalizeFields(dst, src reflect.Value, depth int) error {
	if depth > maxDepth {
		return tooDeep()
	}
	for i := 0; i < src.NumField(); i++ {
		var (
			f, fv = src.Type().Field(i), src.Field(i)
			d     reflect.Value
			err   error
		)
		if dst.IsValid() {
			d = dst.Field(i)
		}
		switch {
		case f.Tag.Get("json") == "-":
		case f.IsExported():
			var x reflect.Value
			if x, err = normalize(fv, depth); err == nil && d.IsValid() {
				d.Set(x)
			}
		case !f.Anonymous:
		case fv.Kind() == reflect.Struct:
			err = normalizeFields(d, fv, depth)
		case fv.Kind() == reflect.Pointer && !fv.IsNil() && fv.Elem().Kind() == reflect.Struct:
			err = normalizeFields(reflect.Value{}, fv.Elem(), depth+1)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// addrMarshaler reports whether encoding/json would call a pointer-receiver
// MarshalJSON or MarshalText on rv, which it does when rv is addressable,
// such as a slice element.
func addrMarshaler(rv reflect.Value) bool {
	if rv.Kind() == reflect.Pointer || !rv.CanAddr() {
		return false
	}
	switch rv.Addr().Interface().(type) {
	case json.Marshaler, encoding.TextMarshaler:
		return true
	}
	return false
}

// tooDeep reports a value that nests more than maxDepth levels deep, which a
// value that contains itself always does.
func tooDeep() error {
	return fmt.Errorf("%w: value nests more than %d levels deep or contains itself", ErrInvalidValue, maxDepth)
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
