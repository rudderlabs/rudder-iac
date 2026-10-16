package validator

import (
	"fmt"
	"testing"

	analytics "github.com/rudderlabs/analytics-go/v4"

	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/golang/testdata/validator/emptyidentity"
	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/golang/testdata/validator/examples"
	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/golang/testdata/validator/identitysections"
	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/golang/testdata/validator/ruddertyper"
)

// messageJSON is the whole typ message of a call for user-123 whose context
// holds only ruddertyper, with fields, the message's other keys, appended
// after userId.
func messageJSON(meta, typ, fields string) string {
	return fmt.Sprintf(`{"type": %q, "channel": "server", "userId": "user-123"%s, "context": {"ruddertyper": %s}}`, typ, fields, meta)
}

// Each row's name starts with the golden whose wrapper it calls.
func TestNonTrackWire(t *testing.T) {
	var (
		ex         *examples.RudderTyperAnalytics
		ref        *ruddertyper.RudderTyperAnalytics
		sections   *identitysections.RudderTyperAnalytics
		empty      *emptyidentity.RudderTyperAnalytics
		id         = examples.Identity{UserID: "user-123"}
		refID      = ruddertyper.Identity{UserID: "user-123"}
		sectionsID = identitysections.Identity{UserID: "user-123"}
		emptyID    = emptyidentity.Identity{UserID: "user-123"}
	)

	rows := []wireRow{
		{
			name: "examples: identify on traits sends the message's traits",
			call: func() error {
				return ex.Identify(id, &examples.IdentifyTraits{Email: "a@b.c", Plan: examples.Ptr("pro")})
			},
			want: messageJSON(exMeta, "identify", `, "traits": {"email": "a@b.c", "plan": "pro"}`),
		},
		{
			name: "examples: identify with only an anonymous ID sends anonymousId and no userId",
			call: func() error {
				return ex.Identify(examples.Identity{AnonymousID: "anon-42"}, &examples.IdentifyTraits{Email: "a@b.c"})
			},
			want: `{"type": "identify", "channel": "server", "anonymousId": "anon-42", "traits": {"email": "a@b.c"},
				"context": {"ruddertyper": ` + exMeta + `}}`,
		},
		{
			name: "identitysections: identify on context.traits sends context.traits, whose keys win over the caller's, and keeps the caller's other keys",
			call: func() error {
				return sections.Identify(sectionsID, &identitysections.IdentifyTraits{Email: "new@b.c"},
					identitysections.WithAnalyticsContext(analytics.Context{Traits: analytics.Traits{"email": "old@b.c", "plan": "free"}}))
			},
			want: `{"type": "identify", "channel": "server", "userId": "user-123",
				"context": {"traits": {"email": "new@b.c", "plan": "free"}, "ruddertyper": ` + sectionsMeta + `}}`,
		},
		{
			name: "identitysections: group on traits sends the message's traits",
			call: func() error {
				return sections.Group(sectionsID, "g1", &identitysections.GroupTraits{Active: true, Status: identitysections.Ptr(identitysections.CustomTypeStatusActive)})
			},
			want: messageJSON(sectionsMeta, "group", `, "groupId": "g1", "traits": {"active": true, "status": "active"}`),
		},
		{
			name: "ruddertyper: group on context.traits sends the message's traits and leaves the caller's context.traits untouched",
			call: func() error {
				return ref.Group(refID, "g1", &ruddertyper.GroupTraits{Active: true},
					ruddertyper.WithAnalyticsContext(analytics.Context{Traits: analytics.Traits{"active": false, "plan": "pro"}}))
			},
			want: `{"type": "group", "channel": "server", "userId": "user-123", "groupId": "g1", "traits": {"active": true},
				"context": {"traits": {"active": false, "plan": "pro"}, "ruddertyper": ` + refMeta + `}}`,
		},
		{
			name: "emptyidentity: the empty identify sends no traits and leaves the caller's context.traits untouched",
			call: func() error {
				return empty.Identify(emptyID, emptyidentity.WithAnalyticsContext(analytics.Context{Traits: analytics.Traits{"plan": "pro"}}))
			},
			want: `{"type": "identify", "channel": "server", "userId": "user-123",
				"context": {"traits": {"plan": "pro"}, "ruddertyper": ` + emptyIdentityMeta + `}}`,
		},
		{
			name: "emptyidentity: the empty group sends no traits and leaves the caller's context.traits untouched",
			call: func() error {
				return empty.Group(emptyID, "g1", emptyidentity.WithAnalyticsContext(analytics.Context{Traits: analytics.Traits{"plan": "pro"}}))
			},
			want: `{"type": "group", "channel": "server", "userId": "user-123", "groupId": "g1",
				"context": {"traits": {"plan": "pro"}, "ruddertyper": ` + emptyIdentityMeta + `}}`,
		},
		{
			name: "examples: page sends a non-empty name as the top-level name",
			call: func() error { return ex.Page(id, "Product Detail", &examples.PageProperties{ProductID: "p1"}) },
			want: messageJSON(exMeta, "page", `, "name": "Product Detail", "properties": {"productId": "p1"}`),
		},
		{
			name: "examples: screen sends a non-empty name as the top-level name",
			call: func() error { return ex.Screen(id, "Product Detail", &examples.ScreenProperties{ProductID: "p1"}) },
			want: messageJSON(exMeta, "screen", `, "name": "Product Detail", "properties": {"productId": "p1"}`),
		},
		{
			name: "examples: page with an empty name and a nil payload sends no name or properties",
			call: func() error { return ex.Page(id, "", nil) },
			want: messageJSON(exMeta, "page", ""),
		},
		{
			name: "examples: screen with an empty name and a nil payload sends no name or properties",
			call: func() error { return ex.Screen(id, "", nil) },
			want: messageJSON(exMeta, "screen", ""),
		},
		{
			name: "examples: identify on an empty open rule sends the map as traits",
			call: func() error { return ex.Identify1(id, examples.IdentifyOpenTraits{"plan": "pro"}) },
			want: messageJSON(exMeta, "identify", `, "traits": {"plan": "pro"}`),
		},
		{
			name: "examples: group on an empty open rule sends the map as traits",
			call: func() error { return ex.Group1(id, "g1", examples.GroupOpenTraits{"seats": 5}) },
			want: messageJSON(exMeta, "group", `, "groupId": "g1", "traits": {"seats": 5}`),
		},
		{
			name: "examples: page on an empty open rule sends the map as properties",
			call: func() error { return ex.PageOpen(id, "", examples.PageOpenProperties{"k": "v"}) },
			want: messageJSON(exMeta, "page", `, "properties": {"k": "v"}`),
		},
		{
			name: "examples: screen on an empty open rule sends the map as properties",
			call: func() error { return ex.ScreenOpen(id, "", examples.ScreenOpenProperties{"k": "v"}) },
			want: messageJSON(exMeta, "screen", `, "properties": {"k": "v"}`),
		},
		{
			name: "examples: page on an empty closed rule sends no properties",
			call: func() error { return ex.PageEmpty(id, "") },
			want: messageJSON(exMeta, "page", ""),
		},
		{
			name: "examples: screen on an empty closed rule sends no properties",
			call: func() error { return ex.ScreenEmpty(id, "") },
			want: messageJSON(exMeta, "screen", ""),
		},
		{
			name: "examples: a screen variant case sends its discriminator and properties",
			call: func() error {
				return ex.ScreenVariant(id, "", examples.ScreenVariantPropertiesCaseProduct{ProductID: "p1"})
			},
			want: messageJSON(exMeta, "screen", `, "properties": {"screenType": "product", "productId": "p1"}`),
		},
		{
			name: "examples: WithCategory equal to a typed category is accepted, on a closed schema too",
			call: func() error {
				return ex.PageCategory(id, "", &examples.PageCategoryProperties{Category: examples.Ptr("news")}, examples.WithCategory("news"))
			},
			want: messageJSON(exMeta, "page", `, "properties": {"category": "news"}`),
		},
		{
			name: "examples: WithCategory equal to the discriminator is accepted",
			call: func() error {
				return ex.PageCategoryVariant(id, "", examples.PageCategoryVariantPropertiesCaseNews{Headline: "h"}, examples.WithCategory("news"))
			},
			want: messageJSON(exMeta, "page", `, "properties": {"category": "news", "headline": "h"}`),
		},
		{
			name: "examples: WithCategory equal to an AdditionalProperties category is accepted",
			call: func() error {
				return ex.Page(id, "", &examples.PageProperties{ProductID: "p1", AdditionalProperties: map[string]any{"category": "deals"}},
					examples.WithCategory("deals"))
			},
			want: messageJSON(exMeta, "page", `, "properties": {"productId": "p1", "category": "deals"}`),
		},
		{
			name: "examples: WithCategory different from a typed category returns ErrCategoryConflict",
			call: func() error {
				return ex.PageCategory(id, "", &examples.PageCategoryProperties{Category: examples.Ptr("news")}, examples.WithCategory("sports"))
			},
			errIs: examples.ErrCategoryConflict,
		},
		{
			name: "examples: WithCategory never replaces a case's fixed discriminator, so a different value returns ErrCategoryConflict",
			call: func() error {
				return ex.PageCategoryVariant(id, "", examples.PageCategoryVariantPropertiesCaseNews{Headline: "h"}, examples.WithCategory("sports"))
			},
			errIs: examples.ErrCategoryConflict,
		},
		{
			name: "examples: WithCategory different from an AdditionalProperties category returns ErrCategoryConflict",
			call: func() error {
				return ex.Page(id, "", &examples.PageProperties{ProductID: "p1", AdditionalProperties: map[string]any{"category": "deals"}},
					examples.WithCategory("news"))
			},
			errIs: examples.ErrCategoryConflict,
		},
		{
			name: "examples: WithCategory on a declared category left unset returns ErrCategoryConflict",
			call: func() error {
				return ex.PageCategory(id, "", &examples.PageCategoryProperties{}, examples.WithCategory("news"))
			},
			errIs: examples.ErrCategoryConflict,
		},
		{
			name: "examples: WithCategory on a closed schema without category returns ErrCategoryConflict",
			call: func() error {
				return ex.Screen(id, "", &examples.ScreenProperties{ProductID: "p1"}, examples.WithCategory("Products"))
			},
			errIs: examples.ErrCategoryConflict,
		},
		{
			name:  `examples: WithCategory("") on a closed schema without category returns ErrCategoryConflict`,
			call:  func() error { return ex.PageEmpty(id, "", examples.WithCategory("")) },
			errIs: examples.ErrCategoryConflict,
		},
		{
			name: "examples: WithCategory on an open schema without category adds properties.category",
			call: func() error {
				return ex.Page(id, "", &examples.PageProperties{ProductID: "p1"}, examples.WithCategory("Products"))
			},
			want: messageJSON(exMeta, "page", `, "properties": {"productId": "p1", "category": "Products"}`),
		},
		{
			name: `examples: WithCategory("") on an open schema without category sends ""`,
			call: func() error {
				return ex.ScreenOpen(id, "", examples.ScreenOpenProperties{"k": "v"}, examples.WithCategory(""))
			},
			want: messageJSON(exMeta, "screen", `, "properties": {"k": "v", "category": ""}`),
		},
		{
			name: "examples: WithCategory with a nil payload on an open rule sends only properties.category",
			call: func() error { return ex.Page(id, "", nil, examples.WithCategory("Products")) },
			want: messageJSON(exMeta, "page", `, "properties": {"category": "Products"}`),
		},
		{
			name:  "examples: WithCategory with a nil payload on a closed rule returns ErrCategoryConflict",
			call:  func() error { return ex.Screen(id, "", nil, examples.WithCategory("Products")) },
			errIs: examples.ErrCategoryConflict,
		},
		{
			name: "examples: WithCategory on track is ignored",
			call: func() error { return ex.TrackSomeEmptyTrackEvent(id, examples.WithCategory("Products")) },
			want: messageJSON(exMeta, "track", `, "event": "Some Empty Track Event"`),
		},
		{
			name: "examples: identify with nil traits sends no traits, and ignores WithCategory",
			call: func() error { return ex.Identify(id, nil, examples.WithCategory("Products")) },
			want: messageJSON(exMeta, "identify", ""),
		},
		{
			name: "examples: group with nil traits sends no traits, and ignores WithCategory",
			call: func() error { return ex.Group(id, "g1", nil, examples.WithCategory("Products")) },
			want: messageJSON(exMeta, "group", `, "groupId": "g1"`),
		},
		{
			name:  "examples: group with an empty group ID returns the SDK's FieldError",
			call:  func() error { return ex.Group(id, "", &examples.GroupTraits{CompanyName: "Acme"}) },
			errIs: analytics.FieldError{Type: "analytics.Group", Name: "GroupId", Value: ""},
		},
		{
			name: "examples: WithCategory on an open case adds properties.category",
			call: func() error {
				return ex.PageKindVariant(id, "", examples.PageKindVariantPropertiesCaseArticle{Author: examples.Ptr("Ann")}, examples.WithCategory("news"))
			},
			want: messageJSON(exMeta, "page", `, "properties": {"kind": "article", "author": "Ann", "category": "news"}`),
		},
		{
			name: "examples: WithCategory on a closed case of an open base returns ErrCategoryConflict",
			call: func() error {
				return ex.PageKindVariant(id, "", examples.PageKindVariantPropertiesCaseGallery{}, examples.WithCategory("news"))
			},
			errIs: examples.ErrCategoryConflict,
		},
		{
			name: "examples: a nil page variant payload sends no properties",
			call: func() error { return ex.PageKindVariant(id, "", nil) },
			want: messageJSON(exMeta, "page", ""),
		},
		{
			name: "examples: a typed-nil closed case is checked against the open base, so WithCategory sends only properties.category",
			call: func() error {
				return ex.PageKindVariant(id, "", (*examples.PageKindVariantPropertiesCaseGallery)(nil), examples.WithCategory("news"))
			},
			want: messageJSON(exMeta, "page", `, "properties": {"category": "news"}`),
		},
		{
			name: "examples: a typed-nil screen variant payload sends no properties",
			call: func() error {
				return ex.ScreenVariant(id, "", (*examples.ScreenVariantPropertiesCaseProduct)(nil))
			},
			want: messageJSON(exMeta, "screen", ""),
		},
		{
			name:  "examples: a nil screen variant payload is checked against the closed base, so WithCategory returns ErrCategoryConflict",
			call:  func() error { return ex.ScreenVariant(id, "", nil, examples.WithCategory("news")) },
			errIs: examples.ErrCategoryConflict,
		},
		{
			name: "examples: an identify variant case on context.traits sends context.traits",
			call: func() error {
				return ex.Identify2(id, examples.IdentifyVariantTraitsCaseBusiness{CompanyName: "Acme"})
			},
			want: `{"type": "identify", "channel": "server", "userId": "user-123",
				"context": {"traits": {"accountType": "business", "companyName": "Acme"}, "ruddertyper": ` + exMeta + `}}`,
		},
		{
			name: "examples: a group variant case on context.traits sends the message's traits",
			call: func() error {
				return ex.Group2(id, "g1", examples.GroupVariantTraitsCaseEnterprise{EmployeeCount: 5})
			},
			want: messageJSON(exMeta, "group", `, "groupId": "g1", "traits": {"tier": "enterprise", "employeeCount": 5}`),
		},
		{
			name: "examples: a nil identify variant sends no traits",
			call: func() error { return ex.Identify2(id, nil) },
			want: messageJSON(exMeta, "identify", ""),
		},
		{
			name: "examples: a typed-nil group variant sends no traits",
			call: func() error {
				return ex.Group2(id, "g1", (*examples.GroupVariantTraitsCaseEnterprise)(nil))
			},
			want: messageJSON(exMeta, "group", `, "groupId": "g1"`),
		},
	}

	assertWireRows(t, func(client analytics.Client) {
		ex, ref = examples.New(client), ruddertyper.New(client)
		sections, empty = identitysections.New(client), emptyidentity.New(client)
	}, rows)
}
