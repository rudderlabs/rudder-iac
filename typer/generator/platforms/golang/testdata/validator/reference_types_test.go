package validator

import (
	"encoding/json"
	"testing"

	analytics "github.com/rudderlabs/analytics-go/v4"

	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/golang/testdata/validator/ruddertyper"
)

func TestReferenceTypesWire(t *testing.T) {
	type props = ruddertyper.TrackUserSignedUpProperties
	var (
		ref *ruddertyper.RudderTyperAnalytics
		id  = ruddertyper.Identity{UserID: "user-123"}

		// The one valid payload; each row changes one field of a copy.
		base = props{Profile: ruddertyper.CustomTypeUserProfile{Email: "a@b.c", FirstName: "Ann"}}
	)
	// signedUp is the row that sends base after set changes a field; extra,
	// that field's key and value, is appended to base's properties.
	signedUp := func(name, extra string, set func(*props)) wireRow {
		return wireRow{
			name: name,
			call: func() error {
				p := base
				set(&p)
				return ref.TrackUserSignedUp(id, p)
			},
			want: trackJSON(refMeta, "User Signed Up", `{"active": false, "profile": {"email": "a@b.c", "first_name": "Ann"}`+extra+`}`),
		}
	}

	rows := []wireRow{
		signedUp("any: an optional any left nil omits its key", "", func(*props) {}),
		signedUp("any: Null{} on an optional any sends null", `, "property_of_any": null`,
			func(p *props) { p.PropertyOfAny = ruddertyper.Null{} }),
		signedUp("enum: an integer constant sends a JSON number", `, "priority": 1`,
			func(p *props) { p.Priority = ruddertyper.Ptr(ruddertyper.PropertyPriority1) }),
		signedUp("enum: a number constant sends a JSON number", `, "rating": 1.5`,
			func(p *props) { p.Rating = ruddertyper.Ptr(ruddertyper.PropertyRating1_5) }),
		signedUp("enum: a boolean constant sends a JSON boolean", `, "enabled": true`,
			func(p *props) { p.Enabled = ruddertyper.Ptr(ruddertyper.PropertyEnabledTrue) }),
		signedUp("enum: a string constant sends its string", `, "device_type": "IoT-Device"`,
			func(p *props) { p.DeviceType = ruddertyper.Ptr(ruddertyper.PropertyDeviceTypeIoTDevice) }),
		signedUp("enum: the status custom type sends its underlying string", `, "status": "suspended"`,
			func(p *props) { p.Status = ruddertyper.Ptr(ruddertyper.CustomTypeStatusSuspended) }),
		signedUp("enum: the типы_данных custom type sends its underlying string", `, "unicode_custom_type": "неактивный"`,
			func(p *props) {
				p.UnicodeCustomType = ruddertyper.Ptr(ruddertyper.CustomTypeТипыДанныхНеактивный)
			}),
		signedUp("enum: a mixed enum's string value sends a JSON string", `, "mixed_value": "active"`,
			func(p *props) { p.MixedValue = ruddertyper.Ptr(ruddertyper.PropertyMixedValueActive()) }),
		signedUp("enum: a mixed enum's integer value sends a JSON number", `, "mixed_value": 1`,
			func(p *props) { p.MixedValue = ruddertyper.Ptr(ruddertyper.PropertyMixedValue1()) }),
		signedUp("enum: a mixed enum's boolean value sends a JSON boolean", `, "mixed_value": true`,
			func(p *props) { p.MixedValue = ruddertyper.Ptr(ruddertyper.PropertyMixedValueTrue()) }),
		signedUp("enum: a mixed enum's number value sends a JSON number", `, "mixed_value": 2.5`,
			func(p *props) { p.MixedValue = ruddertyper.Ptr(ruddertyper.PropertyMixedValue2_5()) }),
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
				_, err := json.Marshal(ruddertyper.PropertyMixedValue{})
				return err
			},
			errIs: ruddertyper.ErrInvalidValue,
		},
		signedUp("nil through a wrapper: an invalid []Nullable item sends null at its index", `, "array_with_null_items": ["a", null, "b"]`,
			func(p *props) {
				p.ArrayWithNullItems = ruddertyper.PropertyArrayWithNullItems{ruddertyper.NewNullable("a"), {}, ruddertyper.NewNullable("b")}
			}),
		signedUp("nullable: an invalid Nullable sends null", `, "string_or_null": null`,
			func(p *props) { p.StringOrNull = &ruddertyper.PropertyStringOrNull{} }),
		signedUp("nullable: a valid Nullable sends its value", `, "string_or_null": "x"`,
			func(p *props) { p.StringOrNull = ruddertyper.Ptr(ruddertyper.NewNullable("x")) }),
		signedUp("nullable: Null{} sends null", `, "null_field": null`,
			func(p *props) { p.NullField = &ruddertyper.Null{} }),
		signedUp("wire key: 用户名 is sent unchanged", `, "用户名": "张三"`,
			func(p *props) { p.X用户名 = ruddertyper.Ptr("张三") }),
		signedUp("wire key: unicode_enum_field is sent unchanged with its unicode value", `, "unicode_enum_field": "已完成"`,
			func(p *props) { p.UnicodeEnumField = ruddertyper.Ptr(ruddertyper.PropertyUnicodeEnumField已完成) }),
		signedUp("custom type: an object custom type inside nested objects sends plan keys at every level",
			`, "context": {"ip_address": "127.0.0.1", "nested_context": {"profile": {"email": "b@c.d", "first_name": "Bo", "last_name": "Li"}}}`,
			func(p *props) {
				p.Context = &ruddertyper.TrackUserSignedUpPropertiesContext{
					IPAddress: "127.0.0.1",
					NestedContext: ruddertyper.TrackUserSignedUpPropertiesContextNestedContext{
						Profile: &ruddertyper.CustomTypeUserProfile{Email: "b@c.d", FirstName: "Bo", LastName: ruddertyper.Ptr("Li")},
					},
				}
			}),
		signedUp("custom type: a closed empty object sends {}", `, "empty_object_no_additional_props": {}`,
			func(p *props) { p.EmptyObjectNoAdditionalProps = &ruddertyper.CustomTypeEmptyObjectNoAdditionalProps{} }),
	}

	assertWireRows(t, func(client analytics.Client) { ref = ruddertyper.New(client) }, rows)
}
