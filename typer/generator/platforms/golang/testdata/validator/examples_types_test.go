package validator

import (
	"encoding/json"
	"fmt"
	"maps"
	"testing"

	analytics "github.com/rudderlabs/analytics-go/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/golang/testdata/validator/examples"
)

// wireRow is one call of a type-system wire test. A call that succeeds must
// send want, a whole message; one that fails must return an error wrapping
// errIs and send nothing.
type wireRow struct {
	name  string
	call  func() error
	want  string
	errIs error
}

// assertWireRows makes every call in one capture, after start has built the
// wrapper the calls use, and checks each row against the message it sent.
func assertWireRows(t *testing.T, start func(analytics.Client), rows []wireRow) {
	t.Helper()
	var (
		errs = make([]error, len(rows))
		sent int
	)
	wire, succeeded := capture(t, func(client analytics.Client) {
		start(client)
		for i, row := range rows {
			errs[i] = row.call()
			if row.want != "" {
				sent++
			}
		}
	})
	require.Len(t, wire, sent, "a failed call sent a message, or a call sent none; errors: %v", errs)
	require.Len(t, succeeded, sent)

	for i, row := range rows {
		// Taking the row's message here, not in the subtest, keeps rows and
		// messages paired when -run skips a subtest.
		var msg map[string]any
		if row.want != "" {
			msg, wire = wire[0], wire[1:]
		}
		t.Run(row.name, func(t *testing.T) {
			if row.errIs != nil {
				assert.ErrorIs(t, errs[i], row.errIs)
				return
			}
			require.NoError(t, errs[i])
			assertMessage(t, row.want, msg)
		})
	}

	// Config.Callback receives the snapshot itself, so its Go value types show
	// what the wire JSON cannot, such as an enum left as its generated type.
	// context.ruddertyper is skipped: it is generated constant metadata, and
	// its trackingPlanVersion is a Go int.
	for _, msg := range succeeded {
		var (
			ctx  *analytics.Context
			data []map[string]any
		)
		switch m := msg.(type) {
		case analytics.Track:
			ctx, data = m.Context, []map[string]any{m.Properties, m.Integrations}
		case analytics.Page:
			ctx, data = m.Context, []map[string]any{m.Properties, m.Integrations}
		case analytics.Screen:
			ctx, data = m.Context, []map[string]any{m.Properties, m.Integrations}
		case analytics.Identify:
			ctx, data = m.Context, []map[string]any{m.Traits, m.Integrations}
		case analytics.Group:
			ctx, data = m.Context, []map[string]any{m.Traits, m.Integrations}
		default:
			require.Failf(t, "Config.Callback received an unexpected message", "%T", msg)
		}
		require.NotNil(t, ctx)
		extra := maps.Clone(ctx.Extra)
		delete(extra, "ruddertyper")
		for _, v := range append(data, ctx.Traits, extra) {
			assertPlain(t, v)
		}
	}
}

// trackJSON is the whole message of a track call for user-123 without options.
func trackJSON(meta, event, props string) string {
	return fmt.Sprintf(`{"type": "track", "channel": "server", "event": %q, "userId": "user-123",
		"properties": %s, "context": {"ruddertyper": %s}}`, event, props, meta)
}

func TestExamplesTypesWire(t *testing.T) {
	type multiProps = examples.TrackSomeMultiTypeEventProperties
	var (
		ex *examples.RudderTyperAnalytics
		id = examples.Identity{UserID: "user-123"}

		// One valid payload per event; each row changes one field of a copy.
		enum       = examples.TrackSomeEnumEventProperties{Mode: examples.PropertyModeActive, Status: examples.PropertyStatusActive1}
		multi      = multiProps{Matrix: [][]any{}, StringOrStrings: examples.NewPropertyStringOrStringsString("s")}
		primitives = examples.TrackEveryPrimitiveTypeProperties{Any: "x", Array: []any{}, Number: 1.5, String: "s"}
		custom     = examples.TrackSomeCustomTypeEventProperties{SomeObject: examples.CustomTypeSomeObjectType{SomeCustomString: "c"}}
		nested     = examples.TrackSomeNestedObjectEventProperties{Location: examples.NewNullable(examples.TrackSomeNestedObjectEventPropertiesLocation{Lat: 1.5})}
		someTrack  = examples.TrackSomeTrackEventProperties{SomeString: "s", SomeInteger: 1}
	)
	// multiSent is the row that sends multi after set changes a field; extra,
	// that field's key and value, is appended to multi's properties.
	multiSent := func(name, extra string, set func(*multiProps)) wireRow {
		return wireRow{
			name: name,
			call: func() error {
				p := multi
				set(&p)
				return ex.TrackSomeMultiTypeEvent(id, p)
			},
			want: trackJSON(exMeta, "Some Multi Type Event", `{"matrix": [], "someObjectOrNull": null, "stringOrStrings": "s"`+extra+`}`),
		}
	}

	rows := []wireRow{
		{
			name: "enum: a nullable enum sends its constant's value",
			call: func() error {
				p := enum
				p.Stage = examples.Ptr(examples.NewNullable(examples.PropertyStageValueDraft))
				return ex.TrackSomeEnumEvent(id, p)
			},
			want: trackJSON(exMeta, "Some Enum Event", `{"mode": "Active", "status": "active", "stage": "draft"}`),
		},
		{
			name: "enum: an invalid nullable enum sends null",
			call: func() error {
				p := enum
				p.Stage = &examples.PropertyStage{}
				return ex.TrackSomeEnumEvent(id, p)
			},
			want: trackJSON(exMeta, "Some Enum Event", `{"mode": "Active", "status": "active", "stage": null}`),
		},
		{
			name: "zero value: a union without a null member fails the event method",
			call: func() error {
				p := multi
				p.StringOrStrings = examples.PropertyStringOrStrings{}
				return ex.TrackSomeMultiTypeEvent(id, p)
			},
			errIs: examples.ErrInvalidValue,
		},
		{
			name: "zero value: a union without a null member fails json.Marshal",
			call: func() error {
				_, err := json.Marshal(examples.PropertyStringOrStrings{})
				return err
			},
			errIs: examples.ErrInvalidValue,
		},
		multiSent("nil through a wrapper: a union Object member built from nil sends {}", `, "someMultiType": {}`,
			func(p *multiProps) { p.SomeMultiType = examples.Ptr(examples.NewPropertySomeMultiTypeObject(nil)) }),
		multiSent("nil through a wrapper: a union Array member built from nil sends []", `, "someMultiType": []`,
			func(p *multiProps) { p.SomeMultiType = examples.Ptr(examples.NewPropertySomeMultiTypeArray(nil)) }),
		multiSent("nil through a wrapper: a valid Nullable holding a nil slice sends []", `, "someArrayOrNull": []`,
			func(p *multiProps) { p.SomeArrayOrNull = examples.Ptr(examples.NewNullable[[]string](nil)) }),
		{
			name: "nil through a wrapper: a valid Nullable holding a nil map sends {}",
			call: func() error {
				p := multi
				p.SomeObjectOrNull = examples.NewNullable[map[string]any](nil)
				return ex.TrackSomeMultiTypeEvent(id, p)
			},
			want: trackJSON(exMeta, "Some Multi Type Event", `{"matrix": [], "someObjectOrNull": {}, "stringOrStrings": "s"}`),
		},
		multiSent("union: the String constructor sends a string", `, "someMultiType": "v"`,
			func(p *multiProps) { p.SomeMultiType = examples.Ptr(examples.NewPropertySomeMultiTypeString("v")) }),
		// 2^53+1 has no float64, so a constructor storing a float sends a different number.
		multiSent("union: the Integer constructor sends an integer", `, "someMultiType": 9007199254740993`,
			func(p *multiProps) {
				p.SomeMultiType = examples.Ptr(examples.NewPropertySomeMultiTypeInteger(9007199254740993))
			}),
		multiSent("union: the Number constructor sends a number", `, "someMultiType": 2.5`,
			func(p *multiProps) { p.SomeMultiType = examples.Ptr(examples.NewPropertySomeMultiTypeNumber(2.5)) }),
		multiSent("union: the Boolean constructor sends a boolean, even false", `, "someMultiType": false`,
			func(p *multiProps) { p.SomeMultiType = examples.Ptr(examples.NewPropertySomeMultiTypeBoolean(false)) }),
		multiSent("union: the Object constructor sends an object", `, "someMultiType": {"k": "v"}`,
			func(p *multiProps) {
				p.SomeMultiType = examples.Ptr(examples.NewPropertySomeMultiTypeObject(map[string]any{"k": "v"}))
			}),
		multiSent("union: the Array constructor sends an array", `, "someMultiType": ["a", 1]`,
			func(p *multiProps) {
				p.SomeMultiType = examples.Ptr(examples.NewPropertySomeMultiTypeArray([]any{"a", 1}))
			}),
		multiSent("union: the Null constructor sends null", `, "someMultiType": null`,
			func(p *multiProps) { p.SomeMultiType = examples.Ptr(examples.NewPropertySomeMultiTypeNull()) }),
		multiSent("union: array items send one value per member", `, "someArrayOfMultipleTypes": ["a", 9007199254740993, 1.5, true]`,
			func(p *multiProps) {
				p.SomeArrayOfMultipleTypes = examples.PropertySomeArrayOfMultipleTypes{
					examples.NewPropertySomeArrayOfMultipleTypesItemString("a"),
					examples.NewPropertySomeArrayOfMultipleTypesItemInteger(9007199254740993),
					examples.NewPropertySomeArrayOfMultipleTypesItemNumber(1.5),
					examples.NewPropertySomeArrayOfMultipleTypesItemBoolean(true),
				}
			}),
		{
			name: "union: a hoisted Object member sends plan keys",
			call: func() error {
				p := nested
				p.Contact = examples.Ptr(examples.NewTrackSomeNestedObjectEventPropertiesContactObject(
					examples.TrackSomeNestedObjectEventPropertiesContactObject{Email: "a@b.c"},
				))
				return ex.TrackSomeNestedObjectEvent(id, p)
			},
			want: trackJSON(exMeta, "Some Nested Object Event", `{"location": {"lat": 1.5}, "contact": {"email": "a@b.c"}}`),
		},
		{
			name: "any: a required nil sends null",
			call: func() error {
				p := primitives
				p.Any = nil
				return ex.TrackEveryPrimitiveType(id, p)
			},
			want: trackJSON(exMeta, "Every Primitive Type", `{"any": null, "array": [], "number": 1.5, "string": "s"}`),
		},
		{
			name: "any: Null{} sends null",
			call: func() error {
				p := primitives
				p.Any = examples.Null{}
				return ex.TrackEveryPrimitiveType(id, p)
			},
			want: trackJSON(exMeta, "Every Primitive Type", `{"any": null, "array": [], "number": 1.5, "string": "s"}`),
		},
		{
			name: "custom type: an object custom type sends plan keys at every level",
			call: func() error {
				p := custom
				p.UserProfile = &examples.CustomTypeUserProfile{Address: &examples.CustomTypeUserProfileAddress{Street: "Main St"}}
				return ex.TrackSomeCustomTypeEvent(id, p)
			},
			want: trackJSON(exMeta, "Some Custom Type Event", `{"someObject": {"someCustomString": "c"}, "userProfile": {"address": {"street": "Main St"}}}`),
		},
		{
			name: "custom type: an open custom type sends its undeclared AdditionalProperties keys only",
			call: func() error {
				p := custom
				p.SomeObject.AdditionalProperties = map[string]any{"coupon": "SAVE", "someInteger": 9}
				return ex.TrackSomeCustomTypeEvent(id, p)
			},
			want: trackJSON(exMeta, "Some Custom Type Event", `{"someObject": {"someCustomString": "c", "coupon": "SAVE"}}`),
		},
		{
			name: "custom type: an array of custom objects converts each element",
			call: func() error {
				p := custom
				p.Items = examples.PropertyItems{{SomeCustomString: "a"}, {SomeCustomString: "b", SomeInteger: examples.Ptr[int64](2)}}
				return ex.TrackSomeCustomTypeEvent(id, p)
			},
			want: trackJSON(exMeta, "Some Custom Type Event",
				`{"someObject": {"someCustomString": "c"}, "items": [{"someCustomString": "a"}, {"someCustomString": "b", "someInteger": 2}]}`),
		},
		{
			name: "nested object: three levels deep send plan keys at every level",
			call: func() error {
				p := someTrack
				p.SomeNestedObject = &examples.TrackSomeTrackEventPropertiesSomeNestedObject{
					SomeNestedLevel1: &examples.TrackSomeTrackEventPropertiesSomeNestedObjectSomeNestedLevel1{
						SomeNestedLevel2: &examples.TrackSomeTrackEventPropertiesSomeNestedObjectSomeNestedLevel1SomeNestedLevel2{
							SomeInteger: examples.Ptr[int64](2),
							SomeString:  examples.Ptr("deep"),
						},
					},
				}
				return ex.TrackSomeTrackEvent(id, p)
			},
			want: trackJSON(exMeta, "Some Track Event",
				`{"someString": "s", "someInteger": 1, "someNestedObject": {"someNestedLevel1": {"someNestedLevel2": {"someInteger": 2, "someString": "deep"}}}}`),
		},
		{
			name: "nested object: a valid Nullable of a hoisted struct sends the object",
			call: func() error { return ex.TrackSomeNestedObjectEvent(id, nested) },
			want: trackJSON(exMeta, "Some Nested Object Event", `{"location": {"lat": 1.5}}`),
		},
		{
			name: "nested object: an invalid Nullable of a hoisted struct sends null",
			call: func() error {
				p := nested
				p.Location = examples.Nullable[examples.TrackSomeNestedObjectEventPropertiesLocation]{}
				return ex.TrackSomeNestedObjectEvent(id, p)
			},
			want: trackJSON(exMeta, "Some Nested Object Event", `{"location": null}`),
		},
		{
			name: "nested object: hoisted closed empty objects at depth send {}",
			call: func() error {
				p := nested
				p.Foo = &examples.TrackSomeNestedObjectEventPropertiesFoo{BarBaz: &examples.TrackSomeNestedObjectEventPropertiesFooBarBaz{}}
				p.FooBar = &examples.TrackSomeNestedObjectEventPropertiesFooBar{Baz: &examples.TrackSomeNestedObjectEventPropertiesFooBarBaz1{}}
				return ex.TrackSomeNestedObjectEvent(id, p)
			},
			want: trackJSON(exMeta, "Some Nested Object Event", `{"location": {"lat": 1.5}, "foo": {"barBaz": {}}, "fooBar": {"baz": {}}}`),
		},
	}

	assertWireRows(t, func(client analytics.Client) { ex = examples.New(client) }, rows)
}
