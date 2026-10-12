package validator

import (
	"fmt"
	"time"

	analytics "github.com/rudderlabs/analytics-go/v4"

	ruddertyper "github.com/rudderlabs/rudder-iac/typer/generator/platforms/golang/testdata/validator/examples"
)

// Each Example is one fenced caller example of the Go Generation
// Specification, named after its section, compiled against the examples
// golden. None has an Output comment, so go test compiles them but never runs
// them: they exist so that every sample the Go README quotes compiles. Lines
// before a spec example declare what it leaves undeclared.

func Example_requiredAndOptionalFields() {
	props := ruddertyper.TrackSomeTrackEventProperties{
		SomeString:  "hello",
		SomeInteger: 42,
		SomeBoolean: ruddertyper.Ptr(true),
	}
	_ = props
}

func Example_events() {
	occurredAt := time.Now()
	client := analytics.New("<WRITE_KEY>", "<DATA_PLANE_URL>")
	defer client.Close()

	rt := ruddertyper.New(client)
	user := ruddertyper.Identity{UserID: "user-123"}

	// With properties
	err := rt.TrackSomeTrackEvent(user, ruddertyper.TrackSomeTrackEventProperties{SomeString: "hello", SomeInteger: 1})

	// With properties and options
	err = rt.TrackSomeTrackEvent(
		user,
		ruddertyper.TrackSomeTrackEventProperties{SomeString: "hello", SomeInteger: 1},
		ruddertyper.WithIntegrations(analytics.Integrations{"All": false}),
		ruddertyper.WithTimestamp(occurredAt),
	)
	_ = err
}

func Example_nestedObjects() {
	props := ruddertyper.TrackSomeTrackEventProperties{
		SomeNestedObject: &ruddertyper.TrackSomeTrackEventPropertiesSomeNestedObject{
			SomeNestedLevel1: &ruddertyper.TrackSomeTrackEventPropertiesSomeNestedObjectSomeNestedLevel1{
				SomeNestedLevel2: &ruddertyper.TrackSomeTrackEventPropertiesSomeNestedObjectSomeNestedLevel1SomeNestedLevel2{
					SomeString: ruddertyper.Ptr("deep"),
				},
			},
		},
	}
	_ = props
}

func Example_customTypeWithVariants() {
	var props ruddertyper.TrackSomeVariantEventProperties

	props.SomeVariantType = ruddertyper.CustomTypeSomeVariantTypeCaseCase2A{SomeNumber: 3.5}
	// sent as: "someVariantType": {"someNumber": 3.5, "someString": "case_2_a"}
}

func Example_eventRuleWithVariants() {
	client := analytics.New("<WRITE_KEY>", "<DATA_PLANE_URL>")
	defer client.Close()
	rt := ruddertyper.New(client)
	user := ruddertyper.Identity{UserID: "user-123"}

	err := rt.TrackDeviceEvent(user, ruddertyper.TrackDeviceEventPropertiesCaseMobile{OsVersion: "17.1", AppVersion: "2.0"})
	// properties: {"appVersion": "2.0", "deviceType": "mobile", "osVersion": "17.1"}

	err = rt.TrackDeviceEvent(user, ruddertyper.TrackDeviceEventPropertiesDefault{DeviceType: "tv"})
	// properties: {"deviceType": "tv"}
	_ = err
}

// handleVariant is the Type Narrowing Usage example, which is a function.
func handleVariant(data ruddertyper.CustomTypeSomeVariantType) {
	switch v := data.(type) {
	case ruddertyper.CustomTypeSomeVariantTypeCaseCase1:
		fmt.Println(v.SomeInteger) // required in this case
	case ruddertyper.CustomTypeSomeVariantTypeCaseCase2A:
		fmt.Println(v.SomeNumber) // required in this case
	case ruddertyper.CustomTypeSomeVariantTypeCaseCase2B:
		fmt.Println(v.SomeNumber)
	case ruddertyper.CustomTypeSomeVariantTypeDefault:
		fmt.Println(v.SomeString, v.SomeBoolean) // SomeBoolean is *bool
	}
}

func Example_typeNarrowingUsage() {
	handleVariant(ruddertyper.CustomTypeSomeVariantTypeCaseCase1{SomeInteger: 1})
}

func Example_identifyEvents() {
	client := analytics.New("<WRITE_KEY>", "<DATA_PLANE_URL>")
	defer client.Close()
	rt := ruddertyper.New(client)

	user := ruddertyper.Identity{UserID: "user-123"}

	// Just the user
	err := rt.Identify(user, nil)

	// User + traits
	err = rt.Identify(user, &ruddertyper.IdentifyTraits{Email: "user@example.com"})

	// Full: user + traits + options
	err = rt.Identify(
		user,
		&ruddertyper.IdentifyTraits{Email: "user@example.com", UserName: ruddertyper.Ptr("John")},
		ruddertyper.WithIntegrations(analytics.Integrations{"All": false}),
	)

	// Anonymous identify
	err = rt.Identify(ruddertyper.Identity{AnonymousID: "anon-42"}, &ruddertyper.IdentifyTraits{Email: "user@example.com"})
	_ = err
}

func Example_identifyEventsNoTraitCarryOver() {
	client := analytics.New("<WRITE_KEY>", "<DATA_PLANE_URL>")
	defer client.Close()
	rt := ruddertyper.New(client)
	user := ruddertyper.Identity{UserID: "user-123"}

	traits, err := ruddertyper.IdentifyTraits{Email: "user@example.com"}.ToTraits()
	if err == nil {
		err = rt.TrackSomeTrackEvent(
			user,
			ruddertyper.TrackSomeTrackEventProperties{SomeString: "hello", SomeInteger: 1},
			ruddertyper.WithAnalyticsContext(analytics.Context{Traits: traits}),
		)
	}
}

func Example_pageEvents() {
	client := analytics.New("<WRITE_KEY>", "<DATA_PLANE_URL>")
	defer client.Close()
	rt := ruddertyper.New(client)
	user := ruddertyper.Identity{UserID: "user-123"}

	// No properties
	err := rt.Page(user, "", nil)

	// Properties only
	err = rt.Page(user, "", &ruddertyper.PageProperties{ProductID: "prod-123"})

	// Name + properties
	err = rt.Page(user, "Product Detail", &ruddertyper.PageProperties{ProductID: "prod-123"})

	// Category + name + properties + options
	err = rt.Page(
		user,
		"Product Detail",
		&ruddertyper.PageProperties{ProductID: "prod-123", ProductName: ruddertyper.Ptr("iPhone")},
		ruddertyper.WithCategory("Products"),
		ruddertyper.WithIntegrations(analytics.Integrations{"All": true}),
	)
	_ = err
}

func Example_screenEvents() {
	client := analytics.New("<WRITE_KEY>", "<DATA_PLANE_URL>")
	defer client.Close()
	rt := ruddertyper.New(client)
	user := ruddertyper.Identity{UserID: "user-123"}

	// With properties
	err := rt.Screen(user, "Product Detail", &ruddertyper.ScreenProperties{ProductID: "prod-123"})

	// With options
	err = rt.Screen(
		user,
		"Product Detail",
		&ruddertyper.ScreenProperties{ProductID: "prod-123", ProductName: ruddertyper.Ptr("iPhone")},
		ruddertyper.WithIntegrations(analytics.Integrations{"All": true}),
	)

	// This rule is closed, so a category is refused
	err = rt.Screen(user, "Product Detail", nil, ruddertyper.WithCategory("Products"))
	// errors.Is(err, ruddertyper.ErrCategoryConflict) == true
	_ = err
}

func Example_groupEvents() {
	joinedAt := time.Now()
	client := analytics.New("<WRITE_KEY>", "<DATA_PLANE_URL>")
	defer client.Close()
	rt := ruddertyper.New(client)
	user := ruddertyper.Identity{UserID: "user-123"}

	// Just the group ID
	err := rt.Group(user, "company-456", nil)

	// Group ID + traits
	err = rt.Group(user, "company-456", &ruddertyper.GroupTraits{CompanyName: "Acme Inc"})

	// Full: group ID + traits + options
	err = rt.Group(
		user,
		"company-456",
		&ruddertyper.GroupTraits{CompanyName: "Acme Inc", Industry: ruddertyper.Ptr("Tech")},
		ruddertyper.WithTimestamp(joinedAt),
	)
	_ = err
}

func Example_aliasEvents() {
	client := analytics.New("<WRITE_KEY>", "<DATA_PLANE_URL>")
	defer client.Close()

	err := client.Enqueue(analytics.Alias{UserId: "new-user-id", PreviousId: "old-anonymous-id"})
	_ = err
}
