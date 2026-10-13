# Go Generator

Generates a type-safe Go package for a tracking plan on top of analytics-go. Each bullet names the sections of the Go Generation Specification it summarizes. The spec's [main][spec] and [appendix][appendix] pages are RudderStack-internal, and each link opens the top of its page.

## Generating

- `rudder-cli typer generate --platform go --tracking-plan-id <id>` generates from your workspace's tracking plan; add `--local --location <project>` to generate from local specs ([Target Versions][spec]).
- `rudder-cli typer options --platform go` lists the options; set one with `--option name=value` ([Generator Options][spec]).
- The output is one file, `ruddertyper.go` in package `ruddertyper`; the `outputFileName` and `packageName` options rename them ([Generator Options][spec]). `--output` (`-o`) sets its directory, `.` by default: point it at the package's own directory, such as `-o internal/ruddertyper`.
- Never edit the file; regenerate it after every tracking plan change ([Generated File Structure][spec]).
- It needs Go 1.21 or later and analytics-go v4.2.2 or later, imported as `analytics "github.com/rudderlabs/analytics-go/v4"` ([Target Versions][spec]).
- Go fails generation on some plan shapes that other platforms accept, such as one match value in two cases of a variant ([Plan Shapes That Fail Generation][appendix]).

## Wiring

```go
client := analytics.New("<WRITE_KEY>", "<DATA_PLANE_URL>")
defer client.Close()

rt := ruddertyper.New(client)
user := ruddertyper.Identity{UserID: "user-123"}
```

- Create the client with `analytics.New`, or with `analytics.NewWithConfig` to set a `Config` such as `Callback`. `ruddertyper.New(nil)` panics, so a wiring mistake fails at startup ([Generated File Structure][spec], [Concurrency and Ownership][appendix]).
- Share one `RudderTyperAnalytics` across goroutines; `Close()` the client on shutdown to flush, because the wrapper never closes it ([Concurrency and Ownership][appendix]).
- The SDK ignores `Config.DefaultContext`; pass context on each call with `WithAnalyticsContext` ([RudderTyper Context][spec]).

## Calling

- Every method takes an `Identity` first. Set `UserID`, `AnonymousID` or both: the SDK keeps no current user and rejects a call that sets neither ([Generated File Structure][spec]).
- Payloads are generated types. For a struct payload, track takes a value and page, screen, identify and group take a pointer (`nil` sends no payload); variant and open-map payloads are passed as is. An event whose rule allows no properties has no payload parameter ([Event Types][spec]).
- Generated names come from the plan: `Track{Event}` takes a `Track{Event}Properties`, custom types are `CustomType{Name}` and variant cases `{Type}Case{Match}`. A name already taken gets a numeric suffix, such as `Identify1` for a second identify rule ([Naming Conventions][spec]).
- Required fields hold values, and optional fields pointers (or `nil`-able slices, maps and interfaces). Set an optional scalar with `Ptr`; integer fields need `ruddertyper.Ptr[int64](2)`, because `Ptr(2)` is an `*int` ([Required and Optional Fields][spec]).

```go
props := ruddertyper.TrackSomeTrackEventProperties{
	SomeString:  "hello",
	SomeInteger: 42,
	SomeBoolean: ruddertyper.Ptr(true),
}
```

- A property that is one type or `null` is a `Nullable[T]`, whose zero value is `null`. Build a value with `NewNullable[T](v)` and write `T` out: `NewNullable(1)` is a `Nullable[int]`, not a `Nullable[int64]` ([Nullable Types][spec], [Discriminator Values][spec]).
- Multi-type values come from their `New{Type}{Member}` constructors ([Multi-Type Properties][spec]), and enum values from their typed constants, or for a mixed enum from its value functions ([Enums][spec]).
- For a variant, pass a case struct (by value, so type switches match); it fixes the discriminator. When generated, the `Default` struct is meant for values no case matches, but sends whatever value it holds, even a case value or, when unset, the zero value such as `""`; a boolean or enum discriminator whose cases cover every value gets none. An optional discriminator also gets a `{Type}Absent` struct, which sends no discriminator ([Variant Rules][spec], [Custom Type with Variants][spec], [Event Rule with Variants][spec]).

```go
err := rt.TrackDeviceEvent(user, ruddertyper.TrackDeviceEventPropertiesCaseMobile{OsVersion: "17.1", AppVersion: "2.0"})
// properties: {"appVersion": "2.0", "deviceType": "mobile", "osVersion": "17.1"}

err = rt.TrackDeviceEvent(user, ruddertyper.TrackDeviceEventPropertiesDefault{DeviceType: "tv"})
// properties: {"deviceType": "tv"}
```

- To read a variant, use one `case` per struct: a combined `case A, B:` clause leaves `v` typed as the interface. `Discriminator()` returns the value without a switch; for an optional discriminator it returns `(value, ok)`, and `ok` is false only for `Absent` ([Type Narrowing Usage][spec], [Variant Rules][spec]).

```go
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
```

- A rule that allows unplanned properties and declares none takes a `map[string]any` (`nil` allowed) ([Event with Additional Properties Allowed][spec]). An open object that declares properties gets an `AdditionalProperties` field for the undeclared keys; a declared key in it is ignored ([Declared Properties with Additional Properties Allowed][spec]).
- Each call deep-copies its payload and options: reuse them once it returns, but no other goroutine may change them during the call ([Serialization][spec], [Concurrency and Ownership][appendix]).

## Options

- Every method ends in `opts ...Option`: `WithAnalyticsContext` sets the event context, `WithIntegrations` selects destinations, `WithTimestamp` sets `originalTimestamp`, `WithMessageID` sets `messageId`, and `WithCategory` sets the page or screen category. They apply in order, so a later option overrides an earlier one of the same kind ([Call Options][spec]).

```go
// Category + name + properties + options
err = rt.Page(
	user,
	"Product Detail",
	&ruddertyper.PageProperties{ProductID: "prod-123", ProductName: ruddertyper.Ptr("iPhone")},
	ruddertyper.WithCategory("Products"),
	ruddertyper.WithIntegrations(analytics.Integrations{"All": true}),
)
```

- `WithCategory` sets `properties.category` on page and screen calls (other calls ignore it): it must equal a category the payload already holds, or else the schema must be open and not declare `category`. Anything else fails with `ErrCategoryConflict`. An empty string is checked and sent like any other value; leave the option out to send no category ([Call Options][spec], [Page Events][spec]).
- External IDs travel in the context: `WithAnalyticsContext(analytics.Context{Extra: map[string]any{"externalId": ...}})` ([Call Options][spec]).
- Go does not allow single options and an `opts...` slice in one call. Build one slice instead: `append([]ruddertyper.Option{ruddertyper.WithCategory(category)}, opts...)` ([Alternate Invocations (Function Overloads)][spec]).

## Errors

- Methods return the SDK's errors unchanged, and their own errors wrap three sentinels. Match them with `errors.Is` and `errors.As`, never with `==` or a type assertion ([Errors][spec]):
  - `ErrNilPayload`: a nil or typed-nil track variant payload.
  - `ErrInvalidValue`: an unset value that has no valid zero value (see Caveats), or a payload or option value that cannot be sent.
  - `ErrCategoryConflict`: a `WithCategory` the rule does not accept.
  - SDK errors: `errors.As(err, &analytics.FieldError{})` when the identity or group ID is missing, and `errors.Is(err, analytics.ErrClosed)` after `Close`.
- A `nil` error means the message was queued. Delivery failures and messages over `Config.MaxMessageBytes` reach only the client-wide `Config.Callback`, whose messages hold plain maps and slices, not the generated types ([Errors][spec], [Serialization Internals][appendix]).
- To pair a callback result with its call, pass `ruddertyper.WithMessageID(id)` and compare the message's `MessageId` in the callback ([Alternate Invocations (Function Overloads)][spec]).

## Caveats

- Go cannot force a field to be set, so an unset required field is sent as its zero value (`""`, `0`, `false`, `[]`). Unions without a `null` member, mixed enums and variants have no valid zero value, so an unset one fails the call with `ErrInvalidValue` ([Required and Optional Fields][spec], [Plan Shapes That Fail Generation][appendix]).
- To catch unset required fields at lint time, enable the `exhaustruct` linter for the packages that build payloads ([Required and Optional Fields][spec]).
- Generated code checks shapes, not values: an enum accepts any value of its type, and so does a variant's `Default` (see Calling). Check untrusted input with the enum's `IsValid()` ([Errors][spec], [Enums][spec]).
- Identify traits go where the rule's identity section says, `traits` or `context.traits`, while analytics.js always sends them in `context.traits`. Group traits always go to `traits` ([Identity Sections][spec]).
- Nothing carries over between calls: an event sent after `Identify` has no `context.traits` unless you pass them ([Identify Events][spec]):

```go
traits, err := ruddertyper.IdentifyTraits{Email: "user@example.com"}.ToTraits()
if err == nil {
	err = rt.TrackSomeTrackEvent(
		user,
		ruddertyper.TrackSomeTrackEventProperties{SomeString: "hello", SomeInteger: 1},
		ruddertyper.WithAnalyticsContext(analytics.Context{Traits: traits}),
	)
}
```

- An empty payload omits the `properties` or `traits` key, where analytics.js sends `{}` ([Serialization][spec]).
- Alias is not generated, because tracking plans have no alias event. Call the SDK: `client.Enqueue(analytics.Alias{UserId: "new-user-id", PreviousId: "old-anonymous-id"})` ([Alias Events][spec]).
- Before comparing payloads with analytics.js, read [Wire Differences from analytics.js (SDK-driven)][spec]. The [Parity Matrix (TypeScript / Swift / Go)][appendix] compares every construct across platforms.

## Developing

- `testdata/validator/` holds one golden per plan, each in its own package: `ruddertyper/` (reference), `identitysections/`, `emptyidentity/`, `examples/` (`testutils/examples_plan.go`). `go test ./typer/generator/platforms/golang/...` runs the unit tests and diffs the generator's output against the goldens ([Testing][appendix]).
- `make typer-go-update-testdata` regenerates every golden after a generator or plan change ([Testing][appendix]).
- `make typer-go-validate` runs vet, gofmt and `go test -race` in the `testdata/validator` module (analytics-go v4.2.2), compiling the goldens and `examples_test.go` ([Testing][appendix]). It also checks that this README's samples are quoted from `examples_test.go`.
- CI (`.github/workflows/typer-go-validate.yml`) regenerates the goldens and fails when one is stale, then runs `make typer-go-validate` with Go 1.21.0, against v4.2.2 and against the latest analytics-go v4 ([Testing][appendix]).

[spec]: https://app.notion.com/p/3eaf2b415dd081e2b067dac4a180e40d
[appendix]: https://app.notion.com/p/3ebf2b415dd0812a865df4cefe950692
