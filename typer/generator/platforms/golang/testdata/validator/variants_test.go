package validator

import (
	"testing"

	analytics "github.com/rudderlabs/analytics-go/v4"
	"github.com/stretchr/testify/assert"

	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/golang/testdata/validator/examples"
	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/golang/testdata/validator/ruddertyper"
)

func TestVariantsWire(t *testing.T) {
	var (
		ex    *examples.RudderTyperAnalytics
		ref   *ruddertyper.RudderTyperAnalytics
		id    = examples.Identity{UserID: "user-123"}
		refID = ruddertyper.Identity{UserID: "user-123"}

		// Shared sub-values and one valid payload per event; each row changes
		// one field of a copy.
		profile     = ruddertyper.CustomTypeUserProfile{Email: "a@b.c", FirstName: "Ann"}
		mobile      = ruddertyper.TrackEventWithVariantsPropertiesCaseMobile{Profile: profile}
		signedUp    = ruddertyper.TrackUserSignedUpProperties{Profile: profile}
		address     = examples.PropertyAddress{"city": "Pune"}
		someVariant = examples.TrackSomeVariantEventProperties{Shipping: examples.CustomTypeShippingAbsent{Address: address}}
	)
	// mobileSent, signedUpSent and someVariantSent are the messages mobile,
	// signedUp and someVariant send, with extra, the changed field's key and
	// value, appended to their properties.
	var (
		mobileSent = func(extra string) string {
			return trackJSON(refMeta, "Event With Variants", `{"device_type": "mobile", "profile": {"email": "a@b.c", "first_name": "Ann"}`+extra+`}`)
		}
		signedUpSent = func(extra string) string {
			return trackJSON(refMeta, "User Signed Up", `{"active": false, "profile": {"email": "a@b.c", "first_name": "Ann"}`+extra+`}`)
		}
		someVariantSent = func(extra string) string {
			return trackJSON(exMeta, "Some Variant Event", `{"shipping": {"address": {"city": "Pune"}}`+extra+`}`)
		}
	)

	rows := []wireRow{
		{
			name: "variant: Event With Variants Mobile sends the enum's string with its base and case properties",
			call: func() error {
				p := mobile
				p.Tags = []string{"t"}
				return ref.TrackEventWithVariants(refID, p)
			},
			want: mobileSent(`, "tags": ["t"]`),
		},
		{
			name: "variant: Event With Variants Desktop sends the enum's string with its base and case properties",
			call: func() error {
				return ref.TrackEventWithVariants(refID, ruddertyper.TrackEventWithVariantsPropertiesCaseDesktop{
					FirstName: "Bo", LastName: ruddertyper.Ptr("Li"), Profile: profile,
				})
			},
			want: trackJSON(refMeta, "Event With Variants",
				`{"device_type": "desktop", "first_name": "Bo", "last_name": "Li", "profile": {"email": "a@b.c", "first_name": "Ann"}}`),
		},
		{
			name: "variant: page_context Search sends a string discriminator",
			call: func() error {
				p := mobile
				p.PageContext = ruddertyper.CustomTypePageContextCaseSearch{Query: "q"}
				return ref.TrackEventWithVariants(refID, p)
			},
			want: mobileSent(`, "page_context": {"page_type": "search", "query": "q"}`),
		},
		{
			name: "variant: page_context Product sends a string discriminator",
			call: func() error {
				p := mobile
				p.PageContext = ruddertyper.CustomTypePageContextCaseProduct{ProductID: "p1"}
				return ref.TrackEventWithVariants(refID, p)
			},
			want: mobileSent(`, "page_context": {"page_type": "product", "product_id": "p1"}`),
		},
		{
			name: "variant: page_context Home, an empty case, sends only its discriminator",
			call: func() error {
				p := mobile
				p.PageContext = ruddertyper.CustomTypePageContextCaseHome{}
				return ref.TrackEventWithVariants(refID, p)
			},
			want: mobileSent(`, "page_context": {"page_type": "home"}`),
		},
		{
			name: "variant: user_access True sends a boolean discriminator",
			call: func() error {
				p := signedUp
				p.UserAccess = ruddertyper.CustomTypeUserAccessCaseTrue{Email: "b@c.d"}
				return ref.TrackUserSignedUp(refID, p)
			},
			want: signedUpSent(`, "user_access": {"active": true, "email": "b@c.d"}`),
		},
		{
			name: "variant: user_access False sends a boolean discriminator, even false",
			call: func() error {
				p := signedUp
				p.UserAccess = ruddertyper.CustomTypeUserAccessCaseFalse{Status: ruddertyper.CustomTypeStatusSuspended}
				return ref.TrackUserSignedUp(refID, p)
			},
			want: signedUpSent(`, "user_access": {"active": false, "status": "suspended"}`),
		},
		{
			name: "variant: feature_config True sends the union's boolean member",
			call: func() error {
				p := signedUp
				p.FeatureConfig = ruddertyper.CustomTypeFeatureConfigCaseTrue{Age: ruddertyper.Ptr(30.5)}
				return ref.TrackUserSignedUp(refID, p)
			},
			want: signedUpSent(`, "feature_config": {"feature_flag": true, "age": 30.5}`),
		},
		{
			name: "variant: feature_config False sends the union's boolean member, even false",
			call: func() error {
				p := signedUp
				p.FeatureConfig = ruddertyper.CustomTypeFeatureConfigCaseFalse{FirstName: ruddertyper.Ptr("Bo")}
				return ref.TrackUserSignedUp(refID, p)
			},
			want: signedUpSent(`, "feature_config": {"feature_flag": false, "first_name": "Bo"}`),
		},
		{
			name: "variant: feature_config Beta sends the union's string member",
			call: func() error {
				p := signedUp
				p.FeatureConfig = ruddertyper.CustomTypeFeatureConfigCaseBeta{Tags: []string{"early"}}
				return ref.TrackUserSignedUp(refID, p)
			},
			want: signedUpSent(`, "feature_config": {"feature_flag": "beta", "tags": ["early"]}`),
		},
		{
			name: "variant: release Draft sends the nullable enum's value",
			call: func() error {
				p := someVariant
				p.Releases = []examples.CustomTypeRelease{examples.CustomTypeReleaseCaseDraft{}}
				return ex.TrackSomeVariantEvent(id, p)
			},
			want: someVariantSent(`, "releases": [{"stage": "draft"}]`),
		},
		{
			name: "variant: release Live sends the nullable enum's value",
			call: func() error {
				p := someVariant
				p.Releases = []examples.CustomTypeRelease{examples.CustomTypeReleaseCaseLive{URL: "https://x.io"}}
				return ex.TrackSomeVariantEvent(id, p)
			},
			want: someVariantSent(`, "releases": [{"stage": "live", "url": "https://x.io"}]`),
		},
		{
			name: "variant: shipping Express sends its optional discriminator",
			call: func() error {
				p := someVariant
				p.Shipping = examples.CustomTypeShippingCaseExpress{Address: address, Deadline: "today"}
				return ex.TrackSomeVariantEvent(id, p)
			},
			want: trackJSON(exMeta, "Some Variant Event", `{"shipping": {"method": "express", "address": {"city": "Pune"}, "deadline": "today"}}`),
		},
		{
			name: "variant: SomeVariantType Case2A sends its own match value",
			call: func() error {
				p := someVariant
				p.SomeVariantType = examples.CustomTypeSomeVariantTypeCaseCase2A{SomeNumber: 3.5}
				return ex.TrackSomeVariantEvent(id, p)
			},
			want: someVariantSent(`, "someVariantType": {"someString": "case_2_a", "someNumber": 3.5}`),
		},
		{
			name: "variant: SomeVariantType Case2B, the same case's other match value, sends case_2_b",
			call: func() error {
				p := someVariant
				p.SomeVariantType = examples.CustomTypeSomeVariantTypeCaseCase2B{SomeNumber: 1.5, SomeInteger: examples.Ptr[int64](2)}
				return ex.TrackSomeVariantEvent(id, p)
			},
			want: someVariantSent(`, "someVariantType": {"someString": "case_2_b", "someNumber": 1.5, "someInteger": 2}`),
		},
		{
			name: "variant: Coded 1 sends the mixed enum's integer",
			call: func() error {
				return ex.TrackCodedEvent(id, examples.TrackCodedEventProperties{Coded: examples.CustomTypeCodedCase1{}})
			},
			want: trackJSON(exMeta, "Coded Event", `{"coded": {"code": 1}}`),
		},
		{
			name: "variant: Coded A sends the mixed enum's string",
			call: func() error {
				return ex.TrackCodedEvent(id, examples.TrackCodedEventProperties{Coded: examples.CustomTypeCodedCaseA{}})
			},
			want: trackJSON(exMeta, "Coded Event", `{"coded": {"code": "a"}}`),
		},
		{
			name: "default: Event With Variants Default sends the enum value it holds",
			call: func() error {
				return ref.TrackEventWithVariants(refID, ruddertyper.TrackEventWithVariantsPropertiesDefault{
					DeviceType: ruddertyper.PropertyDeviceTypeTablet, Profile: profile, UntypedField: 1,
				})
			},
			want: trackJSON(refMeta, "Event With Variants",
				`{"device_type": "tablet", "profile": {"email": "a@b.c", "first_name": "Ann"}, "untyped_field": 1}`),
		},
		{
			name: "default: release Default holding null sends null",
			call: func() error {
				p := someVariant
				p.Releases = []examples.CustomTypeRelease{examples.CustomTypeReleaseDefault{}}
				return ex.TrackSomeVariantEvent(id, p)
			},
			want: someVariantSent(`, "releases": [{"stage": null}]`),
		},
		{
			name: "absent: shipping Absent sends no discriminator key",
			call: func() error { return ex.TrackSomeVariantEvent(id, someVariant) },
			want: someVariantSent(""),
		},
		{
			name:  "nil: a nil track payload returns ErrNilPayload",
			call:  func() error { return ref.TrackEventWithVariants(refID, nil) },
			errIs: ruddertyper.ErrNilPayload,
		},
		{
			name: "nil: a typed-nil track payload returns ErrNilPayload",
			call: func() error {
				return ref.TrackEventWithVariants(refID, (*ruddertyper.TrackEventWithVariantsPropertiesCaseMobile)(nil))
			},
			errIs: ruddertyper.ErrNilPayload,
		},
		{
			name: "nil: a nil optional variant field omits its key",
			call: func() error { return ref.TrackEventWithVariants(refID, mobile) },
			want: mobileSent(""),
		},
		{
			name: "nil: a typed-nil optional variant field omits its key",
			call: func() error {
				p := mobile
				p.PageContext = (*ruddertyper.CustomTypePageContextCaseHome)(nil)
				return ref.TrackEventWithVariants(refID, p)
			},
			want: mobileSent(""),
		},
		{
			name: "nil: a nil required variant field returns ErrInvalidValue",
			call: func() error {
				p := someVariant
				p.Shipping = nil
				return ex.TrackSomeVariantEvent(id, p)
			},
			errIs: examples.ErrInvalidValue,
		},
		{
			name: "nil: a typed-nil required variant field returns ErrInvalidValue",
			call: func() error {
				p := someVariant
				p.Shipping = (*examples.CustomTypeShippingCaseExpress)(nil)
				return ex.TrackSomeVariantEvent(id, p)
			},
			errIs: examples.ErrInvalidValue,
		},
		{
			name:  "nil: a nil required variant field of an open struct returns ErrInvalidValue",
			call:  func() error { return ex.TrackCodedEvent(id, examples.TrackCodedEventProperties{}) },
			errIs: examples.ErrInvalidValue,
		},
		{
			name: "nil: a nil variant array item returns ErrInvalidValue",
			call: func() error {
				p := someVariant
				p.Releases = []examples.CustomTypeRelease{nil}
				return ex.TrackSomeVariantEvent(id, p)
			},
			errIs: examples.ErrInvalidValue,
		},
		{
			name: "nil: a typed-nil variant array item returns ErrInvalidValue",
			call: func() error {
				p := someVariant
				p.Releases = []examples.CustomTypeRelease{(*examples.CustomTypeReleaseCaseLive)(nil)}
				return ex.TrackSomeVariantEvent(id, p)
			},
			errIs: examples.ErrInvalidValue,
		},
		{
			name: "pointer: a pointer track payload sends like the value",
			call: func() error { return ref.TrackEventWithVariants(refID, &mobile) },
			want: mobileSent(""),
		},
		{
			name: "pointer: a pointer in a variant field sends like the value",
			call: func() error {
				p := mobile
				p.PageContext = &ruddertyper.CustomTypePageContextCaseSearch{Query: "q"}
				return ref.TrackEventWithVariants(refID, p)
			},
			want: mobileSent(`, "page_context": {"page_type": "search", "query": "q"}`),
		},
		{
			name: "open: an open case passes undeclared AdditionalProperties keys, not declared ones or the discriminator",
			call: func() error {
				return ex.TrackSomeOpenVariantEvent(id, examples.TrackSomeOpenVariantEventPropertiesCaseOnLine{
					URL:                  "u",
					AdditionalProperties: map[string]any{"coupon": "SAVE", "discriminator": "override", "geo": "override", "meta": "override", "kind": "offline"},
				})
			},
			want: trackJSON(exMeta, "Some Open Variant Event", `{"kind": "on-line", "url": "u", "coupon": "SAVE"}`),
		},
		{
			name: "open: an open Default passes undeclared AdditionalProperties keys, not declared ones or the discriminator",
			call: func() error {
				return ex.TrackSomeOpenVariantEvent(id, examples.TrackSomeOpenVariantEventPropertiesDefault{
					Kind:                 "retail",
					AdditionalProperties: map[string]any{"coupon": "SAVE", "discriminator": "override", "meta": "override", "note": "override", "kind": "on-line"},
				})
			},
			want: trackJSON(exMeta, "Some Open Variant Event", `{"kind": "retail", "coupon": "SAVE"}`),
		},
		{
			name: "open: an open Absent's AdditionalProperties cannot supply the discriminator",
			call: func() error {
				p := someVariant
				p.Shipping = examples.CustomTypeShippingAbsent{
					Address:              address,
					AdditionalProperties: map[string]any{"coupon": "SAVE", "method": "express"},
				}
				return ex.TrackSomeVariantEvent(id, p)
			},
			want: trackJSON(exMeta, "Some Variant Event", `{"shipping": {"address": {"city": "Pune"}, "coupon": "SAVE"}}`),
		},
		{
			name: "empty: an empty optional slice sends []",
			call: func() error {
				p := mobile
				p.Tags = []string{}
				return ref.TrackEventWithVariants(refID, p)
			},
			want: mobileSent(`, "tags": []`),
		},
		{
			name: "empty: an empty optional map sends {}",
			call: func() error {
				p := mobile
				p.PageContext = ruddertyper.CustomTypePageContextDefault{PageType: "other", PageData: map[string]any{}}
				return ref.TrackEventWithVariants(refID, p)
			},
			want: mobileSent(`, "page_context": {"page_type": "other", "page_data": {}}`),
		},
	}

	assertWireRows(t, func(client analytics.Client) {
		ex, ref = examples.New(client), ruddertyper.New(client)
	}, rows)
}

func TestVariantsDiscriminator(t *testing.T) {
	// Discriminator() is never sent on its own, so its values are compared
	// directly. methodOK packs an optional discriminator's (value, ok) results
	// into one value, so each pair is compared whole in a single assert.
	methodOK := func(v examples.PropertyMethod, ok bool) []any { return []any{v, ok} }
	assert.Equal(t, ruddertyper.PropertyDeviceTypeMobile, ruddertyper.TrackEventWithVariantsPropertiesCaseMobile{}.Discriminator())
	assert.Equal(t, ruddertyper.PropertyDeviceTypeDesktop, ruddertyper.TrackEventWithVariantsPropertiesCaseDesktop{}.Discriminator())
	assert.Equal(t, "search", ruddertyper.CustomTypePageContextCaseSearch{}.Discriminator())
	assert.Equal(t, "product", ruddertyper.CustomTypePageContextCaseProduct{}.Discriminator())
	assert.Equal(t, "home", ruddertyper.CustomTypePageContextCaseHome{}.Discriminator())
	assert.Equal(t, true, ruddertyper.CustomTypeUserAccessCaseTrue{}.Discriminator())
	assert.Equal(t, false, ruddertyper.CustomTypeUserAccessCaseFalse{}.Discriminator())
	assert.Equal(t, ruddertyper.NewPropertyFeatureFlagBoolean(true), ruddertyper.CustomTypeFeatureConfigCaseTrue{}.Discriminator())
	assert.Equal(t, ruddertyper.NewPropertyFeatureFlagBoolean(false), ruddertyper.CustomTypeFeatureConfigCaseFalse{}.Discriminator())
	assert.Equal(t, ruddertyper.NewPropertyFeatureFlagString("beta"), ruddertyper.CustomTypeFeatureConfigCaseBeta{}.Discriminator())
	assert.Equal(t, examples.NewNullable(examples.PropertyStageValueDraft), examples.CustomTypeReleaseCaseDraft{}.Discriminator())
	assert.Equal(t, examples.NewNullable(examples.PropertyStageValueLive), examples.CustomTypeReleaseCaseLive{}.Discriminator())
	assert.Equal(t, examples.PropertyCode1(), examples.CustomTypeCodedCase1{}.Discriminator())
	assert.Equal(t, examples.PropertyCodeA(), examples.CustomTypeCodedCaseA{}.Discriminator())
	assert.Equal(t, methodOK("express", true), methodOK(examples.CustomTypeShippingCaseExpress{}.Discriminator()))
	assert.Equal(t, methodOK("", false), methodOK(examples.CustomTypeShippingAbsent{}.Discriminator()))
}
