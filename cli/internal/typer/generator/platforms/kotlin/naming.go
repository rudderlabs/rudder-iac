package kotlin

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
)

var (
	TypeAliasScope = "typealias"
	DataclassScope = "dataclass"
	EnumScope      = "enum"
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

// FormatMethodName converts a name to camelCase suitable for Kotlin method names
// If prefix is provided, it's prepended to the formatted name with proper casing
func FormatMethodName(prefix, name string) string {
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

	// Add prefix if provided
	if prefix != "" {
		// Convert prefix to camelCase and append the PascalCase name
		formatted = core.ToCamelCase(prefix) + core.ToPascalCase(name)
	}

	return formatted
}

// getOrRegisterCustomTypeClassName returns the registered class name for a custom type
func getOrRegisterCustomTypeClassName(customType *plan.CustomType, nameRegistry *core.NameRegistry) (string, error) {
	className := FormatClassName("CustomType", customType.Name)
	return nameRegistry.RegisterName("customtype:"+customType.Name, DataclassScope, className)
}

// getOrRegisterCustomTypeAliasName returns the registered alias name for a primitive custom type
func getOrRegisterCustomTypeAliasName(customType *plan.CustomType, nameRegistry *core.NameRegistry) (string, error) {
	aliasName := FormatClassName("CustomType", customType.Name)
	return nameRegistry.RegisterName("customtype:"+customType.Name, TypeAliasScope, aliasName)
}

// getOrRegisterPropertyAliasName returns the registered alias name for a property-specific type
func getOrRegisterPropertyAliasName(property *plan.Property, nameRegistry *core.NameRegistry) (string, error) {
	aliasName := FormatClassName("Property", property.Name)
	return nameRegistry.RegisterName("property:"+property.Name, TypeAliasScope, aliasName)
}

// getOrRegisterPropertyEnumName returns the registered enum class name for a property with enum constraints
func getOrRegisterPropertyEnumName(property *plan.Property, nameRegistry *core.NameRegistry) (string, error) {
	enumName := FormatClassName("Property", property.Name)
	return nameRegistry.RegisterName("property:"+property.Name, EnumScope, enumName)
}

// FormatEnumValue converts a string value to UPPER_SNAKE_CASE suitable for Kotlin enum constants
func FormatEnumValue(value string) string {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return ""
	}

	// Convert to upper case and replace invalid characters with underscores
	formatted := strings.ToUpper(trimmedValue)

	// Replace non-alphanumeric characters with underscores
	var result strings.Builder
	for _, r := range formatted {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(r)
		} else {
			result.WriteRune('_')
		}
	}

	formatted = result.String()

	// If it starts with a number, prefix with underscore
	if len(formatted) > 0 && unicode.IsDigit(rune(formatted[0])) {
		formatted = "_" + formatted
	}

	// Handle reserved keywords by prefixing with underscore
	originalLower := strings.ToLower(trimmedValue)
	if KotlinReservedKeywords[originalLower] {
		formatted = "_" + formatted
	}

	return formatted
}

func getOrRegisterEventDataClassName(rule *plan.EventRule, nameRegistry *core.NameRegistry) (string, error) {
	// Generate class name based on event type, name, and section
	var prefix string
	var baseName string

	if rule.Event.EventType == plan.EventTypeTrack {
		// For track events: add "Track" prefix and use the event name
		prefix = "Track"
		baseName = rule.Event.Name
	} else {
		// For other events: use the type as prefix (capitalized) and skip the name
		switch rule.Event.EventType {
		case plan.EventTypeIdentify:
			prefix = "Identify"
		case plan.EventTypePage:
			prefix = "Page"
		case plan.EventTypeScreen:
			prefix = "Screen"
		case plan.EventTypeGroup:
			prefix = "Group"
		default:
			return "", fmt.Errorf("unsupported event type: %s", rule.Event.EventType)
		}
		baseName = ""
	}

	var suffix string
	switch rule.Section {
	case plan.IdentitySectionProperties:
		suffix = "Properties"
	case plan.IdentitySectionTraits:
		suffix = "Traits"
	default:
		return "", fmt.Errorf("unsupported event rule section: %s", rule.Section)
	}

	className := FormatClassName(prefix, baseName+suffix)

	// Register the class name with a unique key
	registrationKey := "event:" + string(rule.Event.EventType) + ":" + rule.Event.Name
	return nameRegistry.RegisterName(
		registrationKey,
		DataclassScope,
		className,
	)
}

// KotlinCollisionHandler provides a Kotlin-specific collision handler for the NameRegistry
func KotlinCollisionHandler(name string, existingNames []string) string {
	return core.DefaultCollisionHandler(name, existingNames)
}

// CreateKotlinNameRegistry creates a new NameRegistry with Kotlin-specific collision handling
func CreateKotlinNameRegistry() *core.NameRegistry {
	return core.NewNameRegistry(KotlinCollisionHandler)
}
