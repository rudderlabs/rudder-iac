package validator

import (
	"encoding/json"
	"testing"

	analytics "github.com/rudderlabs/analytics-go/v4"

	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/golang/testdata/validator/ruddertyper"
)

func TestReferenceTypesWire(t *testing.T) {
	var (
		ref *ruddertyper.RudderTyperAnalytics
		id  = ruddertyper.Identity{UserID: "user-123"}

		// The one valid payload; each row changes one field of a copy.
		base = ruddertyper.TrackUserSignedUpProperties{Profile: ruddertyper.CustomTypeUserProfile{Email: "a@b.c", FirstName: "Ann"}}
	)
	// signedUp is the message base sends, with extra, the changed field's
	// key and value, appended to its properties.
	signedUp := func(extra string) string {
		return trackJSON(refMeta, "User Signed Up", `{"active": false, "profile": {"email": "a@b.c", "first_name": "Ann"}`+extra+`}`)
	}

	rows := []wireRow{
		{
			name: "any: an optional any left nil omits its key",
			call: func() error { return ref.TrackUserSignedUp(id, base) },
			want: signedUp(""),
		},
		{
			name: "any: Null{} on an optional any sends null",
			call: func() error {
				p := base
				p.PropertyOfAny = ruddertyper.Null{}
				return ref.TrackUserSignedUp(id, p)
			},
			want: signedUp(`, "property_of_any": null`),
		},
		{
			name: "enum: an integer constant sends a JSON number",
			call: func() error {
				p := base
				p.Priority = ruddertyper.Ptr(ruddertyper.PropertyPriority1)
				return ref.TrackUserSignedUp(id, p)
			},
			want: signedUp(`, "priority": 1`),
		},
		{
			name: "enum: a number constant sends a JSON number",
			call: func() error {
				p := base
				p.Rating = ruddertyper.Ptr(ruddertyper.PropertyRating1_5)
				return ref.TrackUserSignedUp(id, p)
			},
			want: signedUp(`, "rating": 1.5`),
		},
		{
			name: "enum: a boolean constant sends a JSON boolean",
			call: func() error {
				p := base
				p.Enabled = ruddertyper.Ptr(ruddertyper.PropertyEnabledTrue)
				return ref.TrackUserSignedUp(id, p)
			},
			want: signedUp(`, "enabled": true`),
		},
		{
			name: "enum: a string constant sends its string",
			call: func() error {
				p := base
				p.DeviceType = ruddertyper.Ptr(ruddertyper.PropertyDeviceTypeIoTDevice)
				return ref.TrackUserSignedUp(id, p)
			},
			want: signedUp(`, "device_type": "IoT-Device"`),
		},
		{
			name: "enum: the status custom type sends its underlying string",
			call: func() error {
				p := base
				p.Status = ruddertyper.Ptr(ruddertyper.CustomTypeStatusSuspended)
				return ref.TrackUserSignedUp(id, p)
			},
			want: signedUp(`, "status": "suspended"`),
		},
		{
			name: "enum: the типы_данных custom type sends its underlying string",
			call: func() error {
				p := base
				p.UnicodeCustomType = ruddertyper.Ptr(ruddertyper.CustomTypeТипыДанныхНеактивный)
				return ref.TrackUserSignedUp(id, p)
			},
			want: signedUp(`, "unicode_custom_type": "неактивный"`),
		},
		{
			name: "enum: a mixed enum's string value sends a JSON string",
			call: func() error {
				p := base
				p.MixedValue = ruddertyper.Ptr(ruddertyper.PropertyMixedValueActive())
				return ref.TrackUserSignedUp(id, p)
			},
			want: signedUp(`, "mixed_value": "active"`),
		},
		{
			name: "enum: a mixed enum's integer value sends a JSON number",
			call: func() error {
				p := base
				p.MixedValue = ruddertyper.Ptr(ruddertyper.PropertyMixedValue1())
				return ref.TrackUserSignedUp(id, p)
			},
			want: signedUp(`, "mixed_value": 1`),
		},
		{
			name: "enum: a mixed enum's boolean value sends a JSON boolean",
			call: func() error {
				p := base
				p.MixedValue = ruddertyper.Ptr(ruddertyper.PropertyMixedValueTrue())
				return ref.TrackUserSignedUp(id, p)
			},
			want: signedUp(`, "mixed_value": true`),
		},
		{
			name: "enum: a mixed enum's number value sends a JSON number",
			call: func() error {
				p := base
				p.MixedValue = ruddertyper.Ptr(ruddertyper.PropertyMixedValue2_5())
				return ref.TrackUserSignedUp(id, p)
			},
			want: signedUp(`, "mixed_value": 2.5`),
		},
		{
			name: "zero value: a mixed enum fails the event method",
			call: func() error {
				p := base
				p.MixedValue = &ruddertyper.PropertyMixedValue{}
				return ref.TrackUserSignedUp(id, p)
			},
			errIs: ruddertyper.ErrInvalidValue,
		},
		{
			name: "zero value: a mixed enum fails json.Marshal",
			call: func() error {
				p := base
				p.MixedValue = &ruddertyper.PropertyMixedValue{}
				_, err := json.Marshal(p.MixedValue)
				return err
			},
			errIs: ruddertyper.ErrInvalidValue,
		},
		{
			name: "nil through a wrapper: an invalid []Nullable item sends null at its index",
			call: func() error {
				p := base
				p.ArrayWithNullItems = ruddertyper.PropertyArrayWithNullItems{ruddertyper.NewNullable("a"), {}, ruddertyper.NewNullable("b")}
				return ref.TrackUserSignedUp(id, p)
			},
			want: signedUp(`, "array_with_null_items": ["a", null, "b"]`),
		},
		{
			name: "nullable: an invalid Nullable sends null",
			call: func() error {
				p := base
				p.StringOrNull = &ruddertyper.PropertyStringOrNull{}
				return ref.TrackUserSignedUp(id, p)
			},
			want: signedUp(`, "string_or_null": null`),
		},
		{
			name: "nullable: a valid Nullable sends its value",
			call: func() error {
				p := base
				p.StringOrNull = ruddertyper.Ptr(ruddertyper.NewNullable("x"))
				return ref.TrackUserSignedUp(id, p)
			},
			want: signedUp(`, "string_or_null": "x"`),
		},
		{
			name: "nullable: Null{} sends null",
			call: func() error {
				p := base
				p.NullField = &ruddertyper.Null{}
				return ref.TrackUserSignedUp(id, p)
			},
			want: signedUp(`, "null_field": null`),
		},
		{
			name: "wire key: 用户名 is sent unchanged",
			call: func() error {
				p := base
				p.X用户名 = ruddertyper.Ptr("张三")
				return ref.TrackUserSignedUp(id, p)
			},
			want: signedUp(`, "用户名": "张三"`),
		},
		{
			name: "wire key: unicode_enum_field is sent unchanged with its unicode value",
			call: func() error {
				p := base
				p.UnicodeEnumField = ruddertyper.Ptr(ruddertyper.PropertyUnicodeEnumField已完成)
				return ref.TrackUserSignedUp(id, p)
			},
			want: signedUp(`, "unicode_enum_field": "已完成"`),
		},
		{
			name: "custom type: an object custom type inside nested objects sends plan keys at every level",
			call: func() error {
				p := base
				p.Context = &ruddertyper.TrackUserSignedUpPropertiesContext{
					IPAddress: "127.0.0.1",
					NestedContext: ruddertyper.TrackUserSignedUpPropertiesContextNestedContext{
						Profile: &ruddertyper.CustomTypeUserProfile{Email: "b@c.d", FirstName: "Bo", LastName: ruddertyper.Ptr("Li")},
					},
				}
				return ref.TrackUserSignedUp(id, p)
			},
			want: signedUp(`, "context": {"ip_address": "127.0.0.1",
				"nested_context": {"profile": {"email": "b@c.d", "first_name": "Bo", "last_name": "Li"}}}`),
		},
		{
			name: "custom type: a closed empty object sends {}",
			call: func() error {
				p := base
				p.EmptyObjectNoAdditionalProps = &ruddertyper.CustomTypeEmptyObjectNoAdditionalProps{}
				return ref.TrackUserSignedUp(id, p)
			},
			want: signedUp(`, "empty_object_no_additional_props": {}`),
		},
	}

	assertWireRows(t, func(client analytics.Client) { ref = ruddertyper.New(client) }, rows)
}
