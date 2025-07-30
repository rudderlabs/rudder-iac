package kotlin

import (
	"strings"
	"unicode"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
)

// formatClassName converts a name to PascalCase suitable for Kotlin class names and type aliases
// Returns empty string if input is empty. If prefix is provided, it's prepended to the formatted name.
func FormatClassName(prefix, name string) string {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return ""
	}

	originalLower := strings.ToLower(trimmedName)
	formatted := core.ToPascalCase(trimmedName)

	// If it starts with a number, prefix with underscore
	if len(formatted) > 0 && unicode.IsDigit(rune(formatted[0])) {
		formatted = "_" + formatted
	}

	// Handle reserved keywords
	if KotlinReservedKeywords[originalLower] {
		formatted = "_" + formatted
	}

	// Add prefix if provided
	if prefix != "" {
		formatted = prefix + formatted
	}

	return formatted
}

// formatPropertyName converts a name to camelCase suitable for Kotlin property names
// Returns empty string if input is empty.
func formatPropertyName(name string) string {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return ""
	}

	originalLower := strings.ToLower(trimmedName)
	formatted := core.ToCamelCase(trimmedName)

	// If it starts with a number, prefix with underscore
	if len(formatted) > 0 && unicode.IsDigit(rune(formatted[0])) {
		formatted = "_" + formatted
	}

	// Handle reserved keywords
	if KotlinReservedKeywords[originalLower] {
		formatted = "_" + formatted
	}

	return formatted
}

// getOrRegisterCustomTypeClassName returns the registered class name for a custom type
func getOrRegisterCustomTypeClassName(customType *plan.CustomType, nameRegistry *core.NameRegistry) (string, error) {
	className := FormatClassName("CustomType", customType.Name)
	return nameRegistry.RegisterName("customtype:"+customType.Name, "dataclass", className)
}

// getOrRegisterCustomTypeAliasName returns the registered alias name for a primitive custom type
func getOrRegisterCustomTypeAliasName(customType *plan.CustomType, nameRegistry *core.NameRegistry) (string, error) {
	aliasName := FormatClassName("CustomType", customType.Name)
	return nameRegistry.RegisterName("customtype:"+customType.Name, "typealias", aliasName)
}

// getOrRegisterPropertyAliasName returns the registered alias name for a property-specific type
func getOrRegisterPropertyAliasName(property *plan.Property, nameRegistry *core.NameRegistry) (string, error) {
	aliasName := FormatClassName("Property", property.Name)
	return nameRegistry.RegisterName("property:"+property.Name, "typealias", aliasName)
}

// KotlinCollisionHandler provides a Kotlin-specific collision handler for the NameRegistry
func KotlinCollisionHandler(name string, existingNames []string) string {
	return core.DefaultCollisionHandler(name, existingNames)
}

// CreateKotlinNameRegistry creates a new NameRegistry with Kotlin-specific collision handling
func CreateKotlinNameRegistry() *core.NameRegistry {
	return core.NewNameRegistry(KotlinCollisionHandler)
}
