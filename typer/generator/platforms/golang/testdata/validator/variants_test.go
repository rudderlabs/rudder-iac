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

		// Shared sub-values and one valid payload per event; most rows change
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
			name: "default: Event With Variants Default sends the enum value it holds",
			call: func() error {
				return ref.TrackEventWithVariants(refID, ruddertyper.TrackEventWithVariantsPropertiesDefault{
					DeviceType: ruddertyper.PropertyDeviceTypeTablet, Profile: profile, UntypedField: map[string]any{"custom": "data", "count": 42},
				})
			},
			want: trackJSON(refMeta, "Event With Variants",
				`{"device_type": "tablet", "profile": {"email": "a@b.c", "first_name": "Ann"}, "untyped_field": {"custom": "data", "count": 42}}`),
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
			name: "default: feature_config Default sends the union's string member it holds",
			call: func() error {
				p := signedUp
				p.FeatureConfig = ruddertyper.CustomTypeFeatureConfigDefault{FeatureFlag: ruddertyper.NewPropertyFeatureFlagString("alpha")}
				return ref.TrackUserSignedUp(refID, p)
			},
			want: signedUpSent(`, "feature_config": {"feature_flag": "alpha"}`),
		},
		{
			name: "default: page_context Default inside the event Default sends the string it holds",
			call: func() error {
				return ref.TrackEventWithVariants(refID, ruddertyper.TrackEventWithVariantsPropertiesDefault{
					DeviceType: ruddertyper.PropertyDeviceTypeTablet, Profile: profile,
					PageContext: ruddertyper.CustomTypePageContextDefault{PageType: "settings", PageData: map[string]any{"theme": "dark", "language": "en"}},
				})
			},
			want: trackJSON(refMeta, "Event With Variants",
				`{"device_type": "tablet", "profile": {"email": "a@b.c", "first_name": "Ann"}, "page_context": {"page_type": "settings", "page_data": {"theme": "dark", "language": "en"}}}`),
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
			name: "open: an open case passes undeclared AdditionalProperties keys, not declared ones or its discriminator kind",
			call: func() error {
				return ex.TrackSomeOpenVariantEvent(id, examples.TrackSomeOpenVariantEventPropertiesCaseOnLine{
					URL:                  "u",
					AdditionalProperties: map[string]any{"coupon": "SAVE", "discriminator": "override", "geo": "override", "meta": "override", "kind": "offline"},
				})
			},
			want: trackJSON(exMeta, "Some Open Variant Event", `{"kind": "on-line", "url": "u", "coupon": "SAVE"}`),
		},
		{
			name: "open: an open Default passes undeclared AdditionalProperties keys, not declared ones or its discriminator kind",
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
	}

	assertWireRows(t, func(client analytics.Client) {
		ex, ref = examples.New(client), ruddertyper.New(client)
	}, rows)
}

func TestVariantsDiscriminator(t *testing.T) {
	// Only the values no wire row checks: Express's toMap writes a literal,
	// Absent sends none, and no row sends the other three. Every other case's
	// toMap writes Discriminator(), so its wire row checks it. methodOK packs
	// an optional discriminator's (value, ok) into one value, so each pair is
	// compared whole in a single assert.
	methodOK := func(v examples.PropertyMethod, ok bool) []any { return []any{v, ok} }
	assert.Equal(t, methodOK("express", true), methodOK(examples.CustomTypeShippingCaseExpress{}.Discriminator()))
	assert.Equal(t, methodOK("", false), methodOK(examples.CustomTypeShippingAbsent{}.Discriminator()))
	assert.Equal(t, methodOK("ground", true), methodOK(examples.CustomTypeShippingDefault{Method: "ground"}.Discriminator()))
	assert.Equal(t, "on_line", examples.TrackSomeOpenVariantEventPropertiesCaseOnLine1{}.Discriminator())
	assert.Equal(t, "case_2_b", examples.CustomTypeSomeVariantTypeCaseCase2B{}.Discriminator())
}
