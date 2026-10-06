package golang

// Registry scopes: Go has one namespace per package, methods live on
// RudderTyperAnalytics, and each struct's fields share a namespace with its
// methods.
const (
	packageScope = "package"
	methodScope  = "methods"
)

func fieldScope(structName string) string { return "struct:" + structName + ":fields" }

// runtimeNames are the exported package-level names of the generated runtime.
// They are registered before any plan name, so a colliding plan name takes the
// numeric suffix instead of shadowing them. The set is static, whatever the
// plan uses.
var runtimeNames = []string{
	"RudderTyperAnalytics", "New", "Identity", "Option",
	"WithAnalyticsContext", "WithIntegrations", "WithTimestamp", "WithMessageID", "WithCategory",
	"Ptr", "Null", "Nullable", "NewNullable",
	"ErrNilPayload", "ErrInvalidValue", "ErrCategoryConflict",
}

// reservedMethodNames are RudderTyperAnalytics method names no event may take.
var reservedMethodNames = []string{"Alias"}

// propertiesStructMethods are the exported methods of a properties payload
// struct; a field cannot share a name with them.
var propertiesStructMethods = []string{"MarshalJSON", "ToProperties"}

const additionalPropertiesField = "AdditionalProperties"
