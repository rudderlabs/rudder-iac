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
	Optional   bool   // Whether the property is optional (nullable)
}

// KotlinDataClass represents a Kotlin data class declaration
type KotlinDataClass struct {
	Name       string           // The class name in PascalCase (e.g., "UserProfile")
	Comment    string           // Documentation comment for the class
	Properties []KotlinProperty // Properties of the data class
}

// KotlinMethodArgument represents an argument in a generated Kotlin method's signature
type KotlinMethodArgument struct {
	Name     string // e.g., "groupId", "properties"
	Type     string // e.g., "String", "TrackProductClickedProperties"
	Optional bool   // e.g., true for "userId: String?"
}

// SDKCallArgument represents an argument passed to an internal RudderStack SDK method
type SDKCallArgument struct {
	Name            string // The parameter name for the named argument, e.g., "name", "properties"
	Value           string // The value to pass, e.g., "Product Clicked" or "properties". Generator will handle quoting.
	ShouldSerialize bool   // Whether this argument should be serialized to JsonObject
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
	RudderAnalyticsMethods []RudderAnalyticsMethod // Methods for the RudderAnalytics object
}

// NewKotlinContext creates a new KotlinContext with initialized slices
func NewKotlinContext() *KotlinContext {
	return &KotlinContext{
		TypeAliases:            make([]KotlinTypeAlias, 0),
		DataClasses:            make([]KotlinDataClass, 0),
		RudderAnalyticsMethods: make([]RudderAnalyticsMethod, 0),
	}
}
