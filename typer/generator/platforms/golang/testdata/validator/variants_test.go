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

		// One valid payload per event; each row changes one field of a copy.
		profile  = ruddertyper.CustomTypeUserProfile{Email: "a@b.c", FirstName: "Ann"}
		mobile   = ruddertyper.TrackEventWithVariantsPropertiesCaseMobile{Profile: profile}
		signedUp = ruddertyper.TrackUserSignedUpProperties{Profile: profile}
		address  = examples.PropertyAddress{"city": "Pune"}
		variant  = examples.TrackSomeVariantEventProperties{Shipping: examples.CustomTypeShippingAbsent{Address: address}}
	)
	const absentSent = `{"address": {"city": "Pune"}}`
	// mobileSent, signedUpSent and variantSent are the messages mobile,
	// signedUp and variant send, with extra, the changed field's key and value,
	// appended to their properties; variantSent also takes the shipping object.
	mobileSent := func(extra string) string {
		return trackJSON(refMeta, "Event With Variants", `{"device_type": "mobile", "profile": {"email": "a@b.c", "first_name": "Ann"}`+extra+`}`)
	}
	signedUpSent := func(extra string) string {
		return trackJSON(refMeta, "User Signed Up", `{"active": false, "profile": {"email": "a@b.c", "first_name": "Ann"}`+extra+`}`)
	}
	variantSent := func(shipping, extra string) string {
		return trackJSON(exMeta, "Some Variant Event", `{"shipping": `+shipping+extra+`}`)
	}

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
			name: "variant: user_access CaseTrue sends a boolean discriminator",
			call: func() error {
				p := signedUp
				p.UserAccess = ruddertyper.CustomTypeUserAccessCaseTrue{Email: "b@c.d"}
				return ref.TrackUserSignedUp(refID, p)
			},
			want: signedUpSent(`, "user_access": {"active": true, "email": "b@c.d"}`),
		},
		{
			name: "variant: user_access CaseFalse sends a boolean discriminator, even false",
			call: func() error {
				p := signedUp
				p.UserAccess = ruddertyper.CustomTypeUserAccessCaseFalse{Status: ruddertyper.CustomTypeStatusSuspended}
				return ref.TrackUserSignedUp(refID, p)
			},
			want: signedUpSent(`, "user_access": {"active": false, "status": "suspended"}`),
		},
		{
			name: "variant: feature_config CaseTrue sends the union's boolean member",
			call: func() error {
				p := signedUp
				p.FeatureConfig = ruddertyper.CustomTypeFeatureConfigCaseTrue{Age: ruddertyper.Ptr(30.5)}
				return ref.TrackUserSignedUp(refID, p)
			},
			want: signedUpSent(`, "feature_config": {"feature_flag": true, "age": 30.5}`),
		},
		{
			name: "variant: feature_config CaseFalse sends the union's boolean member, even false",
			call: func() error {
				p := signedUp
				p.FeatureConfig = ruddertyper.CustomTypeFeatureConfigCaseFalse{FirstName: ruddertyper.Ptr("Bo")}
				return ref.TrackUserSignedUp(refID, p)
			},
			want: signedUpSent(`, "feature_config": {"feature_flag": false, "first_name": "Bo"}`),
		},
		{
			name: "variant: feature_config CaseBeta sends the union's string member",
			call: func() error {
				p := signedUp
				p.FeatureConfig = ruddertyper.CustomTypeFeatureConfigCaseBeta{Tags: []string{"early"}}
				return ref.TrackUserSignedUp(refID, p)
			},
			want: signedUpSent(`, "feature_config": {"feature_flag": "beta", "tags": ["early"]}`),
		},
		{
			name: "variant: release CaseDraft sends the nullable enum's value",
			call: func() error {
				p := variant
				p.Releases = []examples.CustomTypeRelease{examples.CustomTypeReleaseCaseDraft{}}
				return ex.TrackSomeVariantEvent(id, p)
			},
			want: variantSent(absentSent, `, "releases": [{"stage": "draft"}]`),
		},
		{
			name: "variant: release CaseLive sends the nullable enum's value",
			call: func() error {
				p := variant
				p.Releases = []examples.CustomTypeRelease{examples.CustomTypeReleaseCaseLive{URL: "https://x.io"}}
				return ex.TrackSomeVariantEvent(id, p)
			},
			want: variantSent(absentSent, `, "releases": [{"stage": "live", "url": "https://x.io"}]`),
		},
		{
			name: "variant: shipping CaseExpress sends its optional discriminator",
			call: func() error {
				p := variant
				p.Shipping = examples.CustomTypeShippingCaseExpress{Address: address, Deadline: "today"}
				return ex.TrackSomeVariantEvent(id, p)
			},
			want: variantSent(`{"method": "express", "address": {"city": "Pune"}, "deadline": "today"}`, ""),
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
			name: "default: shipping Default sends the optional discriminator it holds",
			call: func() error {
				p := variant
				p.Shipping = examples.CustomTypeShippingDefault{Address: address, Carrier: "ups", Method: "ground"}
				return ex.TrackSomeVariantEvent(id, p)
			},
			want: variantSent(`{"address": {"city": "Pune"}, "carrier": "ups", "method": "ground"}`, ""),
		},
		{
			name: "default: release Default holding null sends null",
			call: func() error {
				p := variant
				p.Releases = []examples.CustomTypeRelease{examples.CustomTypeReleaseDefault{}}
				return ex.TrackSomeVariantEvent(id, p)
			},
			want: variantSent(absentSent, `, "releases": [{"stage": null}]`),
		},
		{
			name: "absent: shipping Absent sends no discriminator key",
			call: func() error { return ex.TrackSomeVariantEvent(id, variant) },
			want: variantSent(absentSent, ""),
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
				p := variant
				p.Shipping = nil
				return ex.TrackSomeVariantEvent(id, p)
			},
			errIs: examples.ErrInvalidValue,
		},
		{
			name: "nil: a typed-nil required variant field returns ErrInvalidValue",
			call: func() error {
				p := variant
				p.Shipping = (*examples.CustomTypeShippingCaseExpress)(nil)
				return ex.TrackSomeVariantEvent(id, p)
			},
			errIs: examples.ErrInvalidValue,
		},
		{
			name: "nil: a nil variant array item returns ErrInvalidValue",
			call: func() error {
				p := variant
				p.Releases = []examples.CustomTypeRelease{nil}
				return ex.TrackSomeVariantEvent(id, p)
			},
			errIs: examples.ErrInvalidValue,
		},
		{
			name: "nil: a typed-nil variant array item returns ErrInvalidValue",
			call: func() error {
				p := variant
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
					AdditionalProperties: map[string]any{"coupon": "SAVE", "meta": "override", "kind": "offline"},
				})
			},
			want: trackJSON(exMeta, "Some Open Variant Event", `{"kind": "on-line", "url": "u", "coupon": "SAVE"}`),
		},
		{
			name: "open: an open Default passes undeclared AdditionalProperties keys, not declared ones or the discriminator",
			call: func() error {
				return ex.TrackSomeOpenVariantEvent(id, examples.TrackSomeOpenVariantEventPropertiesDefault{
					Kind:                 "retail",
					AdditionalProperties: map[string]any{"coupon": "SAVE", "note": "override", "kind": "on-line"},
				})
			},
			want: trackJSON(exMeta, "Some Open Variant Event", `{"kind": "retail", "coupon": "SAVE"}`),
		},
		{
			name: "open: an open Absent's AdditionalProperties cannot supply the discriminator",
			call: func() error {
				p := variant
				p.Shipping = examples.CustomTypeShippingAbsent{
					Address:              address,
					AdditionalProperties: map[string]any{"coupon": "SAVE", "method": "express"},
				}
				return ex.TrackSomeVariantEvent(id, p)
			},
			want: variantSent(`{"address": {"city": "Pune"}, "coupon": "SAVE"}`, ""),
		},
	}

	assertWireRows(t, func(client analytics.Client) {
		ex, ref = examples.New(client), ruddertyper.New(client)
	}, rows)

	// Discriminator() is never sent on its own, so its values are compared
	// directly. methodOK pairs the two results of an optional discriminator.
	methodOK := func(v examples.PropertyMethod, ok bool) []any { return []any{v, ok} }
	discriminators := []struct {
		name      string
		got, want any
	}{
		{"Event With Variants Mobile returns its enum constant", ruddertyper.TrackEventWithVariantsPropertiesCaseMobile{}.Discriminator(), ruddertyper.PropertyDeviceTypeMobile},
		{"Event With Variants Desktop returns its enum constant", ruddertyper.TrackEventWithVariantsPropertiesCaseDesktop{}.Discriminator(), ruddertyper.PropertyDeviceTypeDesktop},
		{"page_context Search returns its string", ruddertyper.CustomTypePageContextCaseSearch{}.Discriminator(), "search"},
		{"page_context Product returns its string", ruddertyper.CustomTypePageContextCaseProduct{}.Discriminator(), "product"},
		{"page_context Home returns its string", ruddertyper.CustomTypePageContextCaseHome{}.Discriminator(), "home"},
		{"user_access CaseTrue returns true", ruddertyper.CustomTypeUserAccessCaseTrue{}.Discriminator(), true},
		{"user_access CaseFalse returns false", ruddertyper.CustomTypeUserAccessCaseFalse{}.Discriminator(), false},
		{"feature_config CaseTrue returns the boolean member true", ruddertyper.CustomTypeFeatureConfigCaseTrue{}.Discriminator(), ruddertyper.NewPropertyFeatureFlagBoolean(true)},
		{"feature_config CaseFalse returns the boolean member false", ruddertyper.CustomTypeFeatureConfigCaseFalse{}.Discriminator(), ruddertyper.NewPropertyFeatureFlagBoolean(false)},
		{"feature_config CaseBeta returns the string member beta", ruddertyper.CustomTypeFeatureConfigCaseBeta{}.Discriminator(), ruddertyper.NewPropertyFeatureFlagString("beta")},
		{"release CaseDraft returns a valid Nullable of draft", examples.CustomTypeReleaseCaseDraft{}.Discriminator(), examples.NewNullable(examples.PropertyStageValueDraft)},
		{"release CaseLive returns a valid Nullable of live", examples.CustomTypeReleaseCaseLive{}.Discriminator(), examples.NewNullable(examples.PropertyStageValueLive)},
		{"shipping CaseExpress returns express and ok", methodOK(examples.CustomTypeShippingCaseExpress{}.Discriminator()), methodOK("express", true)},
		{"shipping Default returns its Method and ok", methodOK(examples.CustomTypeShippingDefault{Method: "ground"}.Discriminator()), methodOK("ground", true)},
		{"shipping Absent returns the zero value and not ok", methodOK(examples.CustomTypeShippingAbsent{}.Discriminator()), methodOK("", false)},
	}
	for _, d := range discriminators {
		t.Run("discriminator: "+d.name, func(t *testing.T) { assert.Equal(t, d.want, d.got) })
	}
}
