package kotlin

// KotlinTypeAlias represents a Kotlin type alias declaration
type KotlinTypeAlias struct {
	Alias   string // The alias name (e.g., "EmailType")
	Comment string // Documentation comment
	Type    string // The underlying type (e.g., "String")
}

// KotlinProperty represents a property in a Kotlin data class
type KotlinProperty struct {
	Name       string // The property name in camelCase (e.g., "firstName")
	SerialName string // The serialized property name. (e.g., "first_name")
	Type       string // The property type (e.g., "String", "CustomTypeEmail")
	Comment    string // Documentation comment for the property
	Nullable   bool   // Whether the property is nullable
	Default    string // The default value for the property, e.g., "null"
	Abstract   bool   // Whether this is an abstract property
	Override   bool   // Whether this property overrides an abstract property
}

// KotlinDataClass represents a Kotlin data class declaration
type KotlinDataClass struct {
	Name          string            // The class name in PascalCase (e.g., "UserProfile")
	Comment       string            // Documentation comment for the class
	Properties    []KotlinProperty  // Properties of the data class
	NestedClasses []KotlinDataClass // Nested data classes within this class
}

// KotlinEnumValue represents a single value in a Kotlin enum
type KotlinEnumValue struct {
	Name  string // The Kotlin constant name (e.g., "GET")
	Value any    // The (unformatted) value associated with the enum constant (e.g., "GET")
}

// KotlinEnum represents a Kotlin enum class declaration
type KotlinEnum struct {
	Name    string            // The enum name in PascalCase (e.g., "PropertyMyEnum")
	Comment string            // Documentation comment for the enum
	Values  []KotlinEnumValue // The enum values with their serial names
}

// KotlinSealedSubclass represents a data class subclass of a sealed class
type KotlinSealedSubclass struct {
	Name           string           // Subclass name (e.g., "CaseSearch", "Default")
	Comment        string           // Documentation comment
	Properties     []KotlinProperty // Constructor properties
	BodyProperties []KotlinProperty // Body properties
	IsDataClass    bool             // If false, generate as regular class instead of data class
}

// KotlinSealedClass represents a Kotlin sealed class with subclasses
type KotlinSealedClass struct {
	Name       string                 // Sealed class name (e.g., "CustomTypePageType")
	Comment    string                 // Documentation comment
	Properties []KotlinProperty       // Direct properties of the sealed class
	Subclasses []KotlinSealedSubclass // List of subclasses
}

// KotlinMethodArgument represents an argument in a generated Kotlin method's signature
type KotlinMethodArgument struct {
	Name             string // e.g., "groupId", "properties"
	Type             string // e.g., "String", "TrackProductClickedProperties"
	Nullable         bool   // e.g., true for "userId: String?"
	Default          any    // The default value: nil (no default), string (for literals), or variable name
	IsLiteralDefault bool   // If true, Default should be formatted as a Kotlin literal
}

// SDKCallArgument represents an argument passed to an internal RudderStack SDK method
type SDKCallArgument struct {
	Name            string // The parameter name for the named argument, e.g., "name", "properties"
	Value           any    // The value: string (literal or variable), int, bool, etc.
	ShouldSerialize bool   // Whether this argument should be serialized to JsonObject
	IsLiteral       bool   // If true, Value should be formatted as a Kotlin literal
}

// SDKCall represents a call to an internal RudderStack SDK method
type SDKCall struct {
	MethodName string            // The name of the SDK method, e.g., "track"
	Arguments  []SDKCallArgument // The list of arguments for the SDK call
}

// RudderAnalyticsMethod represents a method in the generated RudderAnalytics object
type RudderAnalyticsMethod struct {
	Name            string                 // The public method name, e.g., "productClicked"
	MethodArguments []KotlinMethodArgument // Arguments for the public method's signature
	SDKCall         SDKCall                // The structured, internal SDK call
	Comment         string                 // KDoc comment
}

// KotlinContext holds all the information needed to generate Kotlin code
type KotlinContext struct {
	TypeAliases            []KotlinTypeAlias       // Type aliases for primitive custom types
	DataClasses            []KotlinDataClass       // Data classes for object custom types
	SealedClasses          []KotlinSealedClass     // Sealed classes for variant types
	Enums                  []KotlinEnum            // Enum classes for properties with enum constraints
	RudderAnalyticsMethods []RudderAnalyticsMethod // Methods for the RudderAnalytics object
	EventContext           map[string]string       // Fixed context to be included with every event
	RudderCLIVersion       string                  // Version of Rudder CLI used to generate this code
	TrackingPlanName       string                  // Name of the tracking plan
	TrackingPlanID         string                  // ID of the tracking plan
	TrackingPlanVersion    int                     // Version of the tracking plan
	TrackingPlanURL        string                  // URL to the tracking plan (if available)
}

// NewKotlinContext creates a new KotlinContext with initialized slices
func NewKotlinContext() *KotlinContext {
	return &KotlinContext{
		TypeAliases:            make([]KotlinTypeAlias, 0),
		DataClasses:            make([]KotlinDataClass, 0),
		SealedClasses:          make([]KotlinSealedClass, 0),
		Enums:                  make([]KotlinEnum, 0),
		RudderAnalyticsMethods: make([]RudderAnalyticsMethod, 0),
	}
}
