package kotlin

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
)

const globalTypeScope = "types"

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

// getOrRegisterCustomTypeName returns the registered type name for a custom type.
func getOrRegisterCustomTypeName(customType *plan.CustomType, nameRegistry *core.NameRegistry) (string, error) {
	typeName := FormatClassName("CustomType", customType.Name)
	return nameRegistry.RegisterName("customtype:"+customType.Name, globalTypeScope, typeName)
}

// getOrRegisterPropertyTypeName returns the registered type name for a property-specific type.
func getOrRegisterPropertyTypeName(property *plan.Property, nameRegistry *core.NameRegistry) (string, error) {
	typeName := FormatClassName("Property", property.Name)
	return nameRegistry.RegisterName("property:"+property.Name, globalTypeScope, typeName)
}

// getOrRegisterEnumValue returns the registered enum value name for an enum value
// Uses the enum's scope to ensure values within the same enum are deduplicated
// typeName should be the name of the Kotlin type that contains the enum
func getOrRegisterEnumValue(typeName string, value any, nameRegistry *core.NameRegistry) (string, error) {
	enumScope := fmt.Sprintf("enum:%s", typeName)
	formatted := FormatEnumValue(value)
	valueKey := fmt.Sprintf("%v", value)

	// If FormatEnumValue returns empty string (e.g., for emojis or symbols only),
	// use underscore as a placeholder which will be numbered by the collision handler
	if formatted == "" {
		formatted = "_"
	}

	// Check if the result is only underscores (_, __, ___, etc.)
	// These are reserved in Kotlin, so we need to append a number
	// by registering a dummy enum id to trigger the collision handler
	isOnlyUnderscores := true
	for _, r := range formatted {
		if r != '_' {
			isOnlyUnderscores = false
			break
		}
	}
	if isOnlyUnderscores {
		_, err := nameRegistry.RegisterName(fmt.Sprintf("enum:placeholder:%s", formatted), enumScope, formatted)
		if err != nil {
			return "", err
		}
	}

	n, err := nameRegistry.RegisterName(valueKey, enumScope, formatted)
	if err != nil {
		return "", fmt.Errorf("failed to register name for enum %q", typeName)
	}

	return n, nil
}

// FormatEnumValue converts a string value to UPPER_SNAKE_CASE suitable for Kotlin enum constants
func FormatEnumValue(value any) string {
	formatted := fmt.Sprintf("%v", value)
	formatted = strings.TrimSpace(formatted)
	afterReplacement := core.ReplaceSpecialCharacters(formatted, "_")

	words := core.SplitIntoWords(afterReplacement)
	if len(words) == 0 {
		// If no words were found, check if we have underscores from special char replacement
		// This handles cases like "!!!" -> "___" where the underscores should be preserved
		if len(afterReplacement) > 0 && strings.Trim(afterReplacement, "_") == "" {
			// The string contains only underscores, preserve them
			formatted = afterReplacement
		} else {
			// Empty or whitespace only
			return ""
		}
	} else {
		formatted = strings.Join(words, "_")
	}

	formatted = strings.ToUpper(formatted)
	formatted = handleStartingCharacter(formatted)
	formatted = handleReservedKeyword(formatted)
	return formatted
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
		globalTypeScope,
		className,
	)
}

// getOrRegisterPropertyArrayItemTypeName returns the registered type name for array items with multiple types
func getOrRegisterPropertyArrayItemTypeName(property *plan.Property, nameRegistry *core.NameRegistry) (string, error) {
	typeName := FormatClassName("ArrayItem", property.Name)
	return nameRegistry.RegisterName("property:item:"+property.Name, globalTypeScope, typeName)
}

// KotlinCollisionHandler provides a Kotlin-specific collision handler for the NameRegistry
func KotlinCollisionHandler(name string, existingNames []string) string {
	return core.DefaultCollisionHandler(name, existingNames)
}

// CreateKotlinNameRegistry creates a new NameRegistry with Kotlin-specific collision handling
func CreateKotlinNameRegistry() *core.NameRegistry {
	return core.NewNameRegistry(KotlinCollisionHandler)
}
