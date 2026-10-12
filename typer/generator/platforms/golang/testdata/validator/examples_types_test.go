package validator

import (
	"encoding/json"
	"fmt"
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
	wire, _ := capture(t, func(client analytics.Client) {
		start(client)
		for i, row := range rows {
			errs[i] = row.call()
			if row.want != "" {
				sent++
			}
		}
	})
	require.Len(t, wire, sent, "a failed call sent a message, or a call sent none; errors: %v", errs)

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
}

// trackJSON is the whole message of a track call for user-123 without options.
func trackJSON(meta, event, props string) string {
	return fmt.Sprintf(`{"type": "track", "channel": "server", "event": %q, "userId": "user-123",
		"properties": %s, "context": {"ruddertyper": %s}}`, event, props, meta)
}

func TestExamplesTypesWire(t *testing.T) {
	var (
		ex *examples.RudderTyperAnalytics
		id = examples.Identity{UserID: "user-123"}

		// One valid payload per event; each row changes one field of a copy.
		enum       = examples.TrackSomeEnumEventProperties{Mode: examples.PropertyModeActive, Status: examples.PropertyStatusActive1}
		multi      = examples.TrackSomeMultiTypeEventProperties{Matrix: [][]any{}, StringOrStrings: examples.NewPropertyStringOrStringsString("s")}
		primitives = examples.TrackEveryPrimitiveTypeProperties{Any: "x", Array: []any{}, Number: 1.5, String: "s"}
		custom     = examples.TrackSomeCustomTypeEventProperties{SomeObject: examples.CustomTypeSomeObjectType{SomeCustomString: "c"}}
		nested     = examples.TrackSomeNestedObjectEventProperties{Location: examples.NewNullable(examples.TrackSomeNestedObjectEventPropertiesLocation{Lat: 1.5})}
		someTrack  = examples.TrackSomeTrackEventProperties{SomeString: "s", SomeInteger: 1}
	)
	// multiSent is the message multi sends, with extra, the changed field's
	// key and value, appended to its properties.
	multiSent := func(extra string) string {
		return trackJSON(exMeta, "Some Multi Type Event", `{"matrix": [], "someObjectOrNull": null, "stringOrStrings": "s"`+extra+`}`)
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
				p := multi
				p.StringOrStrings = examples.PropertyStringOrStrings{}
				_, err := json.Marshal(p.StringOrStrings)
				return err
			},
			errIs: examples.ErrInvalidValue,
		},
		{
			name: "nil through a wrapper: a union Object member built from nil sends {}",
			call: func() error {
				p := multi
				p.SomeMultiType = examples.Ptr(examples.NewPropertySomeMultiTypeObject(nil))
				return ex.TrackSomeMultiTypeEvent(id, p)
			},
			want: multiSent(`, "someMultiType": {}`),
		},
		{
			name: "nil through a wrapper: a union Array member built from nil sends []",
			call: func() error {
				p := multi
				p.SomeMultiType = examples.Ptr(examples.NewPropertySomeMultiTypeArray(nil))
				return ex.TrackSomeMultiTypeEvent(id, p)
			},
			want: multiSent(`, "someMultiType": []`),
		},
		{
			name: "nil through a wrapper: a valid Nullable holding a nil slice sends []",
			call: func() error {
				p := multi
				p.SomeArrayOrNull = examples.Ptr(examples.NewNullable[[]string](nil))
				return ex.TrackSomeMultiTypeEvent(id, p)
			},
			want: multiSent(`, "someArrayOrNull": []`),
		},
		{
			name: "union: the String constructor sends a string",
			call: func() error {
				p := multi
				p.SomeMultiType = examples.Ptr(examples.NewPropertySomeMultiTypeString("v"))
				return ex.TrackSomeMultiTypeEvent(id, p)
			},
			want: multiSent(`, "someMultiType": "v"`),
		},
		{
			name: "union: the Integer constructor sends an integer",
			call: func() error {
				p := multi
				// 2^53+1 has no float64, so a constructor storing a float sends a different number.
				p.SomeMultiType = examples.Ptr(examples.NewPropertySomeMultiTypeInteger(9007199254740993))
				return ex.TrackSomeMultiTypeEvent(id, p)
			},
			want: multiSent(`, "someMultiType": 9007199254740993`),
		},
		{
			name: "union: the Number constructor sends a number",
			call: func() error {
				p := multi
				p.SomeMultiType = examples.Ptr(examples.NewPropertySomeMultiTypeNumber(2.5))
				return ex.TrackSomeMultiTypeEvent(id, p)
			},
			want: multiSent(`, "someMultiType": 2.5`),
		},
		{
			name: "union: the Boolean constructor sends a boolean, even false",
			call: func() error {
				p := multi
				p.SomeMultiType = examples.Ptr(examples.NewPropertySomeMultiTypeBoolean(false))
				return ex.TrackSomeMultiTypeEvent(id, p)
			},
			want: multiSent(`, "someMultiType": false`),
		},
		{
			name: "union: the Object constructor sends an object",
			call: func() error {
				p := multi
				p.SomeMultiType = examples.Ptr(examples.NewPropertySomeMultiTypeObject(map[string]any{"k": "v"}))
				return ex.TrackSomeMultiTypeEvent(id, p)
			},
			want: multiSent(`, "someMultiType": {"k": "v"}`),
		},
		{
			name: "union: the Array constructor sends an array",
			call: func() error {
				p := multi
				p.SomeMultiType = examples.Ptr(examples.NewPropertySomeMultiTypeArray([]any{"a", 1}))
				return ex.TrackSomeMultiTypeEvent(id, p)
			},
			want: multiSent(`, "someMultiType": ["a", 1]`),
		},
		{
			name: "union: the Null constructor sends null",
			call: func() error {
				p := multi
				p.SomeMultiType = examples.Ptr(examples.NewPropertySomeMultiTypeNull())
				return ex.TrackSomeMultiTypeEvent(id, p)
			},
			want: multiSent(`, "someMultiType": null`),
		},
		{
			name: "union: array items send one value per member",
			call: func() error {
				p := multi
				p.SomeArrayOfMultipleTypes = examples.PropertySomeArrayOfMultipleTypes{
					examples.NewPropertySomeArrayOfMultipleTypesItemString("a"),
					examples.NewPropertySomeArrayOfMultipleTypesItemInteger(1),
					examples.NewPropertySomeArrayOfMultipleTypesItemNumber(1.5),
					examples.NewPropertySomeArrayOfMultipleTypesItemBoolean(true),
				}
				return ex.TrackSomeMultiTypeEvent(id, p)
			},
			want: multiSent(`, "someArrayOfMultipleTypes": ["a", 1, 1.5, true]`),
		},
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
	}

	assertWireRows(t, func(client analytics.Client) { ex = examples.New(client) }, rows)
}
