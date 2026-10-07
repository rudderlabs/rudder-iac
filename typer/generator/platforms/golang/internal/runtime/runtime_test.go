package runtime

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"net"
	"strings"
	"testing"
	"time"

	analytics "github.com/rudderlabs/analytics-go/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testObject stands in for a generated open struct. Its MarshalJSON differs
// from toMap, so output built from toMap shows that the hook wins.
type testObject struct {
	Name                 string
	Tags                 []string
	AdditionalProperties map[string]any
}

func (v testObject) toMap() map[string]any {
	m := make(map[string]any, len(v.AdditionalProperties)+2)
	maps.Copy(m, v.AdditionalProperties)
	m["name"] = v.Name
	m["tags"] = v.Tags
	return m
}

func (v testObject) MarshalJSON() ([]byte, error) { return []byte(`"from MarshalJSON"`), nil }

// testVariant stands in for a generated variant interface, which lists toMap.
type testVariant interface {
	toMap() map[string]any
	isTestVariant()
}

type testCase struct{ Kind string }

func (v testCase) toMap() map[string]any { return map[string]any{"kind": v.Kind} }
func (testCase) isTestVariant()          {}

// testPayload is a generated struct whose MarshalJSON is marshal, as in
// generated code, so encoding it starts a new walk.
type testPayload struct{ AdditionalProperties map[string]any }

func (v testPayload) toMap() map[string]any        { return maps.Clone(v.AdditionalProperties) }
func (v testPayload) MarshalJSON() ([]byte, error) { return marshal(v) }

// testUnion stands in for a union, mixed enum, Nullable or Null. Its
// MarshalJSON differs from wireValue, so output built from wireValue shows
// that the hook wins.
type testUnion struct {
	value any
	set   bool
}

func (u testUnion) wireValue() (any, error) {
	if !u.set {
		return nil, fmt.Errorf("%w: unset union", ErrInvalidValue)
	}
	return u.value, nil
}

func (testUnion) MarshalJSON() ([]byte, error) { return []byte(`"from MarshalJSON"`), nil }

type (
	testEnum  string
	testLevel int8
)

type callerStruct struct {
	Count  int
	Price  float64
	Skip   func() `json:"-"`
	hidden int
}

type (
	callerNode   struct{ Next *callerNode }
	callerHolder struct{ Payload testPayload }
)

// callerFields promotes Tags from an embedded pointer to an unexported type,
// as encoding/json does.
type (
	callerFields struct {
		Items  []string
		Labels map[string]int
		*embeddedFields
	}
	embeddedFields struct{ Tags []string }
	callerOmitZero struct {
		Items  []string       `json:",omitzero"`
		Labels map[string]int `json:",omitzero"`
		Done   func()         `json:",omitzero"`
	}
	// encoding/json skips the fields promoted from an embedded struct tagged
	// "-", so a function there is never encoded.
	callerIgnored struct {
		Name        string
		callerHooks `json:"-"`
	}
	callerHooks struct{ OnSend func() }
)

type (
	callerEmbedded    struct{ embeddedPayload }
	callerEmbeddedPtr struct{ *embeddedPayload }
	embeddedPayload   struct{ Payload testPayload }
)

// ptrText and ptrJSON marshal through pointer receivers, which encoding/json
// calls only on addressable values, such as slice elements and values behind
// a pointer.
type (
	ptrText       string
	ptrJSON       struct{ N int }
	callerPtrJSON struct{ J ptrJSON }
)

func (s *ptrText) MarshalText() ([]byte, error) { return []byte(strings.ToUpper(string(*s))), nil }
func (*ptrJSON) MarshalJSON() ([]byte, error)   { return []byte(`"from MarshalJSON"`), nil }

// upperText is encoded by its MarshalText, which differs from its value.
type upperText string

func (s upperText) MarshalText() ([]byte, error) { return []byte(strings.ToUpper(string(s))), nil }

type testMarshalError struct{}

func (testMarshalError) Error() string { return "marshal failed" }

type failingMarshaler struct{}

func (failingMarshaler) MarshalJSON() ([]byte, error) { return nil, testMarshalError{} }

// nest wraps leaf in n slices.
func nest(n int, leaf any) any {
	v := leaf
	for range n {
		v = []any{v}
	}
	return v
}

func TestSnapshotAndMarshal(t *testing.T) {
	var (
		five         = 5
		leaf         = "leaf"
		cyclicMap    = map[string]any{}
		cyclicObj    = testObject{AdditionalProperties: map[string]any{}}
		cyclicNode   = &callerNode{}
		cyclicHolder = testPayload{AdditionalProperties: map[string]any{}}
		cyclicKeyed  = testPayload{AdditionalProperties: map[string]any{}}
		cyclicEmbed  = testPayload{AdditionalProperties: map[string]any{}}
		cyclicEmbedP = testPayload{AdditionalProperties: map[string]any{}}
	)
	cyclicMap["self"] = cyclicMap
	cyclicObj.AdditionalProperties["self"] = cyclicObj
	cyclicNode.Next = cyclicNode
	cyclicHolder.AdditionalProperties["holder"] = callerHolder{Payload: cyclicHolder}
	cyclicKeyed.AdditionalProperties["byID"] = map[int]testPayload{1: cyclicKeyed}
	cyclicEmbed.AdditionalProperties["embedded"] = callerEmbedded{embeddedPayload{Payload: cyclicEmbed}}
	cyclicEmbedP.AdditionalProperties["embedded"] = callerEmbeddedPtr{&embeddedPayload{Payload: cyclicEmbedP}}

	tests := []struct {
		name    string
		in      any
		want    map[string]any
		wantErr bool
		wantAs  any // a pointer errors.As must fill from the error chain
	}{
		{
			name: "integers become int64, unsigned integers uint64",
			in:   map[string]any{"i": int8(-3), "n": 42, "u": uint16(7), "ptr": uintptr(1)},
			want: map[string]any{"i": int64(-3), "n": int64(42), "u": uint64(7), "ptr": uint64(1)},
		},
		{
			name: "float32 stays float32",
			in:   map[string]any{"f": 1.5, "f32": float32(2.5)},
			want: map[string]any{"f": 1.5, "f32": float32(2.5)},
		},
		{
			name: "enums become their underlying type",
			in:   map[string]any{"kind": testEnum("a"), "level": testLevel(2)},
			want: map[string]any{"kind": "a", "level": int64(2)},
		},
		{
			name: "nil pointers and interfaces become null",
			in:   map[string]any{"field": map[string]*int{"p": nil}, "typedNil": (*int)(nil), "nil": nil, "set": &five},
			want: map[string]any{"field": map[string]any{"p": nil}, "typedNil": nil, "nil": nil, "set": int64(5)},
		},
		{
			name: "nil open payload becomes {}",
			in:   map[string]any(nil),
			want: map[string]any{},
		},
		{
			name: "nil slices and maps become [] and {} at any depth",
			in: map[string]any{
				"slice":  []string(nil),
				"map":    map[string]int(nil),
				"nested": []any{map[string]any{"inner": []int(nil)}, [][]string{nil}},
			},
			want: map[string]any{
				"slice":  []any{},
				"map":    map[string]any{},
				"nested": []any{map[string]any{"inner": []any{}}, []any{[]any{}}},
			},
		},
		{
			name: "nil slices and maps in AdditionalProperties become [] and {}",
			in:   testObject{Name: "a", AdditionalProperties: map[string]any{"list": []string(nil), "deep": map[string]any{"m": map[string]bool(nil)}}},
			want: map[string]any{"name": "a", "tags": []any{}, "list": []any{}, "deep": map[string]any{"m": map[string]any{}}},
		},
		{
			name: "generated objects use toMap, not MarshalJSON, at any depth",
			in:   map[string]any{"obj": testObject{Name: "a"}, "list": []testObject{{Name: "b", Tags: []string{"t"}}}},
			want: map[string]any{
				"obj":  map[string]any{"name": "a", "tags": []any{}},
				"list": []any{map[string]any{"name": "b", "tags": []any{"t"}}},
			},
		},
		{
			name: "wireValue results are walked",
			in:   map[string]any{"enum": testUnion{value: testEnum("a"), set: true}, "null": testUnion{set: true}, "empty": testUnion{value: []string(nil), set: true}},
			want: map[string]any{"enum": "a", "null": nil, "empty": []any{}},
		},
		{
			name:    "wireValue error",
			in:      map[string]any{"union": testUnion{}},
			wantErr: true,
		},
		{
			name: "variant items",
			in:   map[string]any{"items": []testVariant{testCase{Kind: "a"}}},
			want: map[string]any{"items": []any{map[string]any{"kind": "a"}}},
		},
		{
			name:    "nil variant item",
			in:      map[string]any{"items": []testVariant{nil}},
			wantErr: true,
		},
		{
			name:    "typed-nil variant item",
			in:      map[string]any{"items": []testVariant{(*testCase)(nil)}},
			wantErr: true,
		},
		{
			name: "[]byte becomes base64, a nil one null",
			in:   map[string]any{"bytes": []byte("hi"), "nil": []byte(nil)},
			want: map[string]any{"bytes": "aGk=", "nil": nil},
		},
		{
			name: "maps with non-string keys go through encoding/json, a nil one as null",
			in:   map[string]any{"map": map[int]float64{1: 2.5}, "nil": map[int]string(nil)},
			want: map[string]any{"map": map[string]any{"1": json.Number("2.5")}, "nil": nil},
		},
		{
			name: "json.Number",
			in:   map[string]any{"n": json.Number("12.50")},
			want: map[string]any{"n": json.Number("12.50")},
		},
		{
			name: "caller marshalers go through encoding/json",
			in:   map[string]any{"time": time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), "raw": json.RawMessage(`{"a":1}`), "text": upperText("a")},
			want: map[string]any{"time": "2026-01-02T03:04:05Z", "raw": map[string]any{"a": json.Number("1")}, "text": "A"},
		},
		{
			name: "caller structs go through encoding/json with json.Number numbers",
			in:   map[string]any{"s": callerStruct{Count: 3, Price: 1.5, Skip: func() {}, hidden: 1}},
			want: map[string]any{"s": map[string]any{"Count": json.Number("3"), "Price": json.Number("1.5")}},
		},
		{
			name: "nil slices and maps inside caller structs and non-string-key maps are null, as in encoding/json",
			in:   map[string]any{"s": callerFields{embeddedFields: &embeddedFields{}}, "byID": map[int][]string{1: nil}},
			want: map[string]any{
				"s":    map[string]any{"Items": nil, "Labels": nil, "Tags": nil},
				"byID": map[string]any{"1": nil},
			},
		},
		{
			name: "caller struct fields tagged omitzero are omitted when nil, as in encoding/json",
			in:   map[string]any{"s": callerOmitZero{}},
			want: map[string]any{"s": map[string]any{}},
		},
		{
			name: "fields promoted from an embedded struct tagged \"-\" are skipped, as in encoding/json",
			in:   map[string]any{"s": callerIgnored{Name: "a", callerHooks: callerHooks{OnSend: func() {}}}},
			want: map[string]any{"s": map[string]any{"Name": "a"}},
		},
		{
			name: "pointer-receiver marshalers apply only to addressable values, as in encoding/json",
			in:   map[string]any{"slice": []ptrText{"a"}, "map": map[string]ptrText{"k": "a"}, "pointer": &callerPtrJSON{}, "value": callerPtrJSON{}},
			want: map[string]any{
				"slice":   []any{"A"},
				"map":     map[string]any{"k": "a"},
				"pointer": map[string]any{"J": "from MarshalJSON"},
				"value":   map[string]any{"J": map[string]any{"N": json.Number("0")}},
			},
		},
		{
			name:    "caller MarshalJSON error",
			in:      map[string]any{"m": failingMarshaler{}},
			wantErr: true,
			wantAs:  new(testMarshalError),
		},
		{
			name:    "map that contains itself",
			in:      cyclicMap,
			wantErr: true,
		},
		{
			name:    "AdditionalProperties that contain their object",
			in:      cyclicObj,
			wantErr: true,
		},
		{
			name:    "caller struct that contains itself",
			in:      map[string]any{"node": cyclicNode},
			wantErr: true,
		},
		{
			// encoding/json cannot see these cycles, because each MarshalJSON
			// call starts a new walk; only the walk of the caller struct or
			// map can.
			name:    "generated value that contains itself through a caller struct",
			in:      cyclicHolder,
			wantErr: true,
		},
		{
			name:    "generated value that contains itself through a non-string-key map",
			in:      cyclicKeyed,
			wantErr: true,
		},
		{
			name:    "generated value that contains itself through an embedded struct of an unexported type",
			in:      cyclicEmbed,
			wantErr: true,
		},
		{
			name:    "generated value that contains itself through an embedded pointer to an unexported type",
			in:      cyclicEmbedP,
			wantErr: true,
		},
		{
			// Each level is a slice and the interface holding it: 499 levels
			// reach depth 1000 exactly.
			name: "nesting at the depth bound",
			in:   map[string]any{"v": nest(499, "leaf")},
			want: map[string]any{"v": nest(499, "leaf")},
		},
		{
			// The pointer to the leaf is one level more.
			name:    "nesting past the depth bound",
			in:      map[string]any{"v": nest(499, &leaf)},
			wantErr: true,
		},
		{name: "NaN", in: map[string]any{"f": math.NaN()}, wantErr: true},
		{name: "+Inf", in: map[string]any{"f": math.Inf(1)}, wantErr: true},
		{name: "-Inf float32", in: map[string]any{"f": float32(math.Inf(-1))}, wantErr: true},
		{name: "channel", in: map[string]any{"c": make(chan int)}, wantErr: true},
		{name: "function", in: map[string]any{"f": func() {}}, wantErr: true},
		{name: "complex number", in: map[string]any{"c": complex(1, 2)}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := snapshot(tt.in)
			b, marshalErr := marshal(tt.in)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidValue)
				assert.ErrorIs(t, marshalErr, ErrInvalidValue)
				if tt.wantAs != nil {
					assert.ErrorAs(t, err, tt.wantAs)
					assert.ErrorAs(t, marshalErr, tt.wantAs)
				}
				return
			}
			require.NoError(t, err)
			require.NoError(t, marshalErr)
			assert.Equal(t, tt.want, got)

			want, err := json.Marshal(got)
			require.NoError(t, err)
			assert.Equal(t, string(want), string(b))
		})
	}
}

// These mirror the With* options of the generated client.
func withContext(ctx analytics.Context) Option { return func(o *callOptions) { o.context = &ctx } }
func withIntegrations(i analytics.Integrations) Option {
	return func(o *callOptions) { o.integrations = i }
}
func withTimestamp(t time.Time) Option { return func(o *callOptions) { o.timestamp = t } }

func TestApplyOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		want    callOptions
		wantErr bool
	}{
		{
			name: "no options",
			want: callOptions{integrations: analytics.Integrations{}},
		},
		{
			name:    "NaN location field",
			opts:    []Option{withContext(analytics.Context{Location: analytics.LocationInfo{Latitude: math.NaN()}})},
			wantErr: true,
		},
		{
			name:    "NaN in Extra",
			opts:    []Option{withContext(analytics.Context{Extra: map[string]any{"score": math.NaN()}})},
			wantErr: true,
		},
		{
			name:    "Inf in Traits",
			opts:    []Option{withContext(analytics.Context{Traits: analytics.Traits{"score": math.Inf(-1)}})},
			wantErr: true,
		},
		{
			name:    "invalid IP",
			opts:    []Option{withContext(analytics.Context{IP: net.IP{1, 2, 3}})},
			wantErr: true,
		},
		{
			name:    "NaN in integrations",
			opts:    []Option{withIntegrations(analytics.Integrations{"Amplitude": map[string]any{"rate": math.NaN()}})},
			wantErr: true,
		},
		{
			name:    "timestamp after year 9999",
			opts:    []Option{withTimestamp(time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC))},
			wantErr: true,
		},
		{
			name:    "timestamp before year 0",
			opts:    []Option{withTimestamp(time.Date(-1, 1, 1, 0, 0, 0, 0, time.UTC))},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := applyOptions(tt.opts)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidValue)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAnalyticsContext(t *testing.T) {
	caller := analytics.Context{
		Locale: "en-US",
		Extra:  map[string]any{"ruddertyper": "overwritten", "custom": 1},
		Traits: analytics.Traits{"email": "caller@example.com", "name": "Caller"},
	}

	got := callOptions{context: &caller}.analyticsContext()

	assert.Equal(t, &analytics.Context{
		Locale: "en-US",
		Extra:  map[string]any{"custom": 1, "ruddertyper": rudderTyperContext()},
		Traits: analytics.Traits{"email": "caller@example.com", "name": "Caller"},
	}, got)
	assert.Equal(t, analytics.Context{
		Locale: "en-US",
		Extra:  map[string]any{"ruddertyper": "overwritten", "custom": 1},
		Traits: analytics.Traits{"email": "caller@example.com", "name": "Caller"},
	}, caller)
	assert.Equal(t, &analytics.Context{Extra: map[string]any{"ruddertyper": rudderTyperContext()}}, callOptions{}.analyticsContext())
}
