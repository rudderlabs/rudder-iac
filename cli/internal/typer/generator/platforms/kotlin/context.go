package kotlin

// KotlinTypeAlias represents a Kotlin type alias declaration
type KotlinTypeAlias struct {
	Alias   string // The alias name (e.g., "EmailType")
	Comment string // Documentation comment
	Type    string // The underlying type (e.g., "String")
}

// KotlinProperty represents a property in a Kotlin data class
type KotlinProperty struct {
	Name         string // The property name in camelCase (e.g., "firstName")
	OriginalName string // The original property name from the plan (e.g., "first_name"), used for serialization
	Type         string // The property type (e.g., "String", "CustomTypeEmail")
	Comment      string // Documentation comment for the property
	Optional     bool   // Whether the property is optional (nullable)
}

// KotlinDataClass represents a Kotlin data class declaration
type KotlinDataClass struct {
	Name       string           // The class name in PascalCase (e.g., "UserProfile")
	Comment    string           // Documentation comment for the class
	Properties []KotlinProperty // Properties of the data class
}

// KotlinContext holds all the information needed to generate Kotlin code
type KotlinContext struct {
	TypeAliases []KotlinTypeAlias // Type aliases for primitive custom types
	DataClasses []KotlinDataClass // Data classes for object custom types
}

// NewKotlinContext creates a new KotlinContext with initialized slices
func NewKotlinContext() *KotlinContext {
	return &KotlinContext{
		TypeAliases: make([]KotlinTypeAlias, 0),
		DataClasses: make([]KotlinDataClass, 0),
	}
}
