package typescript

// TSInterfaceProperty is one property in a generated TS interface.
type TSInterfaceProperty struct {
	Name     string // camelCase TS identifier (or quoted serial form when not a valid identifier)
	Type     string // resolved TS type expression, e.g. "string" or "string | null"
	Comment  string
	Optional bool // true → name?: T (omittable under exactOptionalPropertyTypes)
	// QuotedName is true when Name must be wrapped in double quotes (e.g. the
	// JSON key has characters like spaces or `-` that aren't valid in
	// identifier syntax). The template emits `"name"?: T` in that case.
	QuotedName bool
}

// TSInterface → export interface X { ... }
type TSInterface struct {
	Name       string
	Comment    string
	Properties []TSInterfaceProperty
}

// TSTypeAlias → export type Alias = Type;
type TSTypeAlias struct {
	Alias   string
	Type    string
	Comment string
}

// TSMethodArgument is one parameter on a RudderTyper method.
type TSMethodArgument struct {
	Name     string
	Type     string
	Comment  string
	Optional bool // true → name?: T
}

// TSSDKArgument is one positional argument forwarded to the underlying
// RudderAnalytics call. Value is a pre-computed TS expression.
type TSSDKArgument struct {
	Value string
}

// TSOverloadSignature is one typed signature for an overloaded method.
// Overloaded methods (identify, group, page) emit one declaration per
// signature followed by a single implementation whose parameter types are
// unions covering every overload — the spec calls for multiple SDK-aligned
// call shapes per non-track event.
type TSOverloadSignature struct {
	Arguments []TSMethodArgument
}

// TSDispatcherBranch is one if/else branch inside the implementation body of an overloaded method.
// Condition is the JS guard (e.g. `typeof arg0 === "string"`); empty for
// the fallback else. SDKArguments are the args passed to the SDK in this branch.
type TSDispatcherBranch struct {
	Condition    string
	SDKArguments []TSSDKArgument
}

// TSAnalyticsMethod is one method on the generated RudderTyper class.
//
// Simple methods (track) use MethodArguments + SDKArguments.
// Overloaded methods (identify, group, page) use Overloads for the public
// signatures and DispatcherBranches for the implementation body. In
// TypeScript, a single function implements all overloaded signatures, so
// the body needs if/else branches to inspect the arguments at runtime and
// forward to the correct SDK call shape.
type TSAnalyticsMethod struct {
	Name            string
	Comment         string
	EventName       string
	MethodArguments []TSMethodArgument
	SDKMethodName   string // "identify", "track", etc.
	SDKArguments    []TSSDKArgument
	Overloads       []TSOverloadSignature
	DispatcherBranches []TSDispatcherBranch
}

// TSVariantGroup groups the case interfaces and union type alias for one
// discriminated union (custom type or event rule variant). Each group
// renders as N case interfaces followed by the union alias.
type TSVariantGroup struct {
	CaseInterfaces []TSInterface
	UnionAlias     TSTypeAlias
}

// TSContext is the root data object passed to RudderTyper.ts.tmpl.
type TSContext struct {
	// PropertyEnums and CustomTypeAliases are emitted as `export type X = ...`
	// declarations. PropertyEnums covers property-level enum constraints; the
	// other slice covers primitive / array / enum custom types.
	PropertyEnums      []TSTypeAlias
	CustomTypeAliases  []TSTypeAlias
	// CustomInterfaces holds object custom types, emitted as `export interface`.
	CustomInterfaces []TSInterface
	// NestedInterfaces holds inline-schema nested objects from event rules,
	// hoisted to top-level interfaces named `{EventInterface}{PropertyPath}`.
	// Ordered deepest-first so a reader sees leaf shapes before composites.
	NestedInterfaces []TSInterface
	// VariantTypes holds discriminated unions — each group contains the case
	// interfaces and the union type alias for one variant (custom type or event).
	VariantTypes []TSVariantGroup
	// Interfaces holds top-level event interfaces (track props, identify traits).
	Interfaces       []TSInterface
	AnalyticsMethods []TSAnalyticsMethod
	// EventContext is the ruddertyper provenance map injected into every event's
	// options.context. Values are pre-formatted TS literal expressions.
	EventContext        map[string]string
	RudderCLIVersion    string
	TrackingPlanName    string
	TrackingPlanID      string
	TrackingPlanVersion int
	TrackingPlanURL     string
	// UsesSDKApiObject is true when at least one method passes a typed object to
	// the SDK (track props or identify traits) and therefore needs the aliased
	// SDK type imports for the strict cast.
	UsesSDKApiObject      bool
	UsesSDKIdentifyTraits bool
	// UsesApiCallback is true when any generated method accepts an
	// `ApiCallback` parameter (per the spec, all methods accept one). Pulled
	// from the SDK rather than defined locally so the callback signature stays
	// in sync with whatever shape the SDK accepts.
	UsesApiCallback bool
}
