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
	// SerialName is the ORIGINAL tracking-plan key (e.g. "product_id") that this
	// property maps to on the wire. Name may be camelCased (e.g. "productId")
	// for an ergonomic interface, but the payload sent to the SDK must use
	// SerialName so it matches the tracking plan. Mirrors Swift's SerialName
	// field and Kotlin's put("first_name", ...) — see DAW-3732.
	SerialName string
}

// TSInterface → export interface X { ... }
type TSInterface struct {
	Name       string
	Comment    string
	Properties []TSInterfaceProperty
}

// TSKeyMapEntry is one camelCase→serial mapping inside a per-interface key map.
//
// When the wire key equals the interface field name (no camelCasing happened),
// no entry is emitted — the runtime remap leaves such keys untouched.
type TSKeyMapEntry struct {
	// FieldName is the camelCase interface property name used as the object key
	// at runtime (the left-hand side of the generated map literal).
	FieldName string
	// SerialName is the original tracking-plan key emitted on the wire.
	SerialName string
	// NestedMapName, when non-empty, is the name of another generated key map
	// (e.g. "UserSignedUpContextKeyMap") whose value must be remapped
	// recursively. This carries nested-object remapping the same way Swift's
	// nested toProperties() does.
	NestedMapName string
}

// TSKeyMap is a per-interface map from camelCase field names back to the
// original tracking-plan keys, emitted as
// `const <Name>: Record<string, KeyMapValue> = { ... }`. Only interfaces with
// at least one renamed key (or a nested object that itself needs remapping)
// produce a map — see collectKeyMaps.
type TSKeyMap struct {
	Name    string // e.g. "UserSignedUpKeyMap"
	Entries []TSKeyMapEntry
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
	// PropsKeyMapName is the name of the key map applied to the props/traits
	// object before the SDK call (e.g. "UserSignedUpKeyMap"). Empty when the
	// event's interface needs no remapping — in that case props are forwarded
	// verbatim. Currently wired for track methods (the canonical case); see the
	// nested-object note in the PR body for the limitation on identify/group/page.
	PropsKeyMapName string
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
	// KeyMaps holds the per-interface camelCase→serial-name maps emitted as
	// `const <Interface>KeyMap` constants. Populated by collectKeyMaps after all
	// interfaces are built. Only interfaces needing remap appear here.
	KeyMaps []TSKeyMap
	// UsesApplyKeyMap is true when at least one method remaps its props, so the
	// generated file needs the shared applyKeyMap runtime helper.
	UsesApplyKeyMap bool
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
