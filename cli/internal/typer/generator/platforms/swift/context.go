package swift

// SwiftTypeAlias → public typealias X = Y
type SwiftTypeAlias struct {
	Alias   string
	Type    string
	Comment string
}

// SwiftEnumValue is one case in a typed enum.
type SwiftEnumValue struct {
	Name  string // Swift identifier (sanitized), e.g. "get"
	Value any    // Original raw value, e.g. "GET" or 200
}

// SwiftEnum → public enum X: String/Int, CaseIterable { case a = "a" }
type SwiftEnum struct {
	Name    string
	Comment string
	RawType string // "String" or "Int"
	Values  []SwiftEnumValue
}

// SwiftMultiTypeCase is one case in a multi-type property enum.
type SwiftMultiTypeCase struct {
	Name           string // e.g. "string", "int", "null"
	AssociatedType string // e.g. "String", "Int", "" (for null)
	IsNull         bool   // true for the null case (no associated value)
}

// SwiftMultiTypeEnum → public enum X { case string(String) ... var value: Any }
type SwiftMultiTypeEnum struct {
	Name    string
	Comment string
	Cases   []SwiftMultiTypeCase
}

// SwiftStructProperty is one stored property in a struct.
type SwiftStructProperty struct {
	Name        string // camelCase Swift identifier
	SerialName  string // original JSON key (for toProperties/toTraits)
	Type        string // Swift type string
	Comment     string
	Optional    bool   // true → T?
	// Pre-computed serialization expression set by generator, e.g.:
	//   plain primitive   → "email"
	//   enum              → "deviceType.rawValue"
	//   multi-type enum   → "pageType.value"
	//   struct            → "address.toProperties()"
	//   variant enum      → "pageType.toProperties()"
	SerializeExpr string
	// ConstantValue is set for discriminator fields in single-match variant cases.
	// When non-empty the property renders as `public let name: Type = value` and
	// is excluded from init — the value is fixed by the variant definition.
	ConstantValue string
}

// SwiftStruct → public struct X { public var p: T; init(...); toProperties()/toTraits() }
type SwiftStruct struct {
	Name            string
	Comment         string
	SerializeMethod string // "toProperties" or "toTraits"
	Properties      []SwiftStructProperty
	NestedStructs   []SwiftStruct
}

// SwiftVariantCase is one case in a discriminated-union enum.
type SwiftVariantCase struct {
	Name               string // Swift case name, e.g. "case1"
	Comment            string
	PayloadType        string // Nested struct name, e.g. "Case1"
	DiscriminatorValue string // e.g. "case_1"
	IsDefault          bool   // true → generates as "case other(Default)"
}

// SwiftVariantEnum → public enum X { case a(A); case other(Default); toProperties() }
type SwiftVariantEnum struct {
	Name    string
	Comment string
	Cases   []SwiftVariantCase
	// Nested payload structs live inside the enum
	PayloadStructs []SwiftStruct
}

// SwiftMethodArgument is one parameter in a RudderTyperAnalytics method.
type SwiftMethodArgument struct {
	Name     string
	Type     string
	Comment  string
	Optional bool
	Default  string // e.g. "nil" or `""` for userId
}

// SwiftSDKCallArgument is one argument in the underlying analytics.X() call.
// Value is a pre-computed Swift expression — the generator handles all logic so the template stays dumb.
// Examples: `"\"Event Name\""`, `"properties.toProperties()"`, `"traits?.toTraits()"`, `"nil"`, `"userId"`
type SwiftSDKCallArgument struct {
	Label string
	Value string
}

// SwiftAnalyticsMethod is one method in RudderTyperAnalytics.
type SwiftAnalyticsMethod struct {
	Name            string
	Comment         string
	EventName       string
	MethodArguments []SwiftMethodArgument
	SDKMethodName   string // "track", "identify", "screen", "group", "alias"
	SDKArguments    []SwiftSDKCallArgument
	AddCategory      bool // screen events have an extra category param
	AddDataToContext bool // context.traits: merge traits into customContext instead of SDK traits param
}

// SwiftContext is the root data object passed to RudderTyper.swift.tmpl.
type SwiftContext struct {
	// UsesRudderValue is true when at least one property has no type constraint,
	// triggering emission of the RudderValue enum in the generated file.
	UsesRudderValue bool
	// TypeAliases holds custom type aliases (e.g. CustomTypeFoo = String) → rendered in MARK: Custom Types
	TypeAliases []SwiftTypeAlias
	// PropertyTypeAliases holds property type aliases (e.g. PropertyFoo = String) → rendered in MARK: Property Types
	PropertyTypeAliases []SwiftTypeAlias
	Enums               []SwiftEnum
	MultiTypeEnums      []SwiftMultiTypeEnum
	VariantEnums        []SwiftVariantEnum
	Structs             []SwiftStruct
	AnalyticsMethods    []SwiftAnalyticsMethod
	EventContext        map[string]string // injected into every event
	RudderCLIVersion    string
	TrackingPlanName    string
	TrackingPlanID      string
	TrackingPlanVersion int
	TrackingPlanURL     string
}
