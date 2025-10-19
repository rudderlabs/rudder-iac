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

func handleReservedKeyword(name string) string {
	if KotlinHardKeywords[name] {
		return "_" + name
	}
	return name
}

func handleStartingCharacter(name string) string {
	if len(name) > 0 && unicode.IsDigit(rune(name[0])) {
		return "_" + name
	}
	return name
}

// formatClassName converts a name to PascalCase suitable for Kotlin class names and type aliases
// Returns empty string if input is empty. If prefix is provided, it's prepended to the formatted name.
func FormatClassName(prefix, name string) string {
	formatted := strings.TrimSpace(name)
	if prefix != "" {
		formatted = fmt.Sprintf("%s_%s", prefix, formatted)
	}
	formatted = core.ToPascalCase(formatted)
	formatted = handleStartingCharacter(formatted)
	formatted = handleReservedKeyword(formatted)
	return formatted
}

// FormatPropertyName converts a name to camelCase suitable for Kotlin property names
// Returns empty string if input is empty.
func FormatPropertyName(name string) string {
	formatted := strings.TrimSpace(name)
	formatted = core.ToCamelCase(formatted)
	formatted = handleStartingCharacter(formatted)
	formatted = handleReservedKeyword(formatted)
	return formatted
}

// FormatMethodName converts a name to camelCase suitable for Kotlin method names
// If prefix is provided, it's prepended to the formatted name with proper casing
func FormatMethodName(prefix, name string) string {
	formatted := strings.TrimSpace(name)
	if prefix != "" {
		formatted = fmt.Sprintf("%s_%s", prefix, formatted)
	}
	formatted = core.ToCamelCase(formatted)
	formatted = handleStartingCharacter(formatted)
	formatted = handleReservedKeyword(formatted)
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

// getOrRegisterCustomTypeEnumName returns the registered enum class name for a custom type with enum constraints
func getOrRegisterCustomTypeEnumName(customType *plan.CustomType, nameRegistry *core.NameRegistry) (string, error) {
	enumName := FormatClassName("CustomType", customType.Name)
	return nameRegistry.RegisterName("customtype:"+customType.Name, EnumScope, enumName)
}

// FormatEnumValue converts a string value to UPPER_SNAKE_CASE suitable for Kotlin enum constants
func FormatEnumValue(value any) string {
	formatted := fmt.Sprintf("%v", value)
	formatted = strings.TrimSpace(formatted)
	formatted = core.ReplaceSpecialCharacters(formatted, "_")
	words := core.SplitIntoWords(formatted)
	if len(words) == 0 {
		return ""
	}

	formatted = strings.Join(words, "_")
	formatted = strings.ToUpper(formatted)
	formatted = handleStartingCharacter(formatted)
	formatted = handleReservedKeyword(formatted)
	return formatted
}

func FormatEnumSerialName(value any) string {
	switch v := value.(type) {
	case string:
		// For strings, preserve the original value with quotes
		return fmt.Sprintf("%q", v)
	default:
		return fmt.Sprintf("%v", value)
	}
}

// formatSealedSubclassName formats a match value into a valid Kotlin class name prefixed with "Case"
func formatSealedSubclassName(matchValue any) string {
	valueStr := fmt.Sprintf("%v", matchValue)
	className := FormatClassName("Case", valueStr)
	return className
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

// getOrRegisterPropertyMultiTypeClassName returns the registered sealed class name for a property with multiple types
func getOrRegisterPropertyMultiTypeClassName(property *plan.Property, nameRegistry *core.NameRegistry) (string, error) {
	className := FormatClassName("Property", property.Name)
	return nameRegistry.RegisterName("property:"+property.Name, DataclassScope, className)
}

// getOrRegisterPropertyMultiTypeArrayItemClassName returns the registered sealed class name for array items with multiple types
func getOrRegisterPropertyMultiTypeArrayItemClassName(property *plan.Property, nameRegistry *core.NameRegistry) (string, error) {
	className := FormatClassName("ArrayItem", property.Name)
	return nameRegistry.RegisterName("property:item:"+property.Name, DataclassScope, className)
}
