package kotlin

// KotlinTypeAlias represents a Kotlin type alias declaration
type KotlinTypeAlias struct {
	Alias   string // The alias name (e.g., "EmailType")
	Comment string // Documentation comment
	Type    string // The underlying type (e.g., "String")
}

// KotlinContext holds all the information needed to generate Kotlin code
type KotlinContext struct {
	TypeAliases []KotlinTypeAlias // Type aliases for primitive custom types
}

// NewKotlinContext creates a new KotlinContext with initialized slices
func NewKotlinContext() *KotlinContext {
	return &KotlinContext{
		TypeAliases: make([]KotlinTypeAlias, 0),
	}
}
