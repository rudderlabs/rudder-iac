package kotlin

import (
	"strings"
	"unicode"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
)

// FormatClassName converts a name to PascalCase suitable for Kotlin class names and type aliases
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

// KotlinCollisionHandler provides a Kotlin-specific collision handler for the NameRegistry
func KotlinCollisionHandler(name string, existingNames []string) string {
	return core.DefaultCollisionHandler(name, existingNames)
}

// CreateKotlinNameRegistry creates a new NameRegistry with Kotlin-specific collision handling
func CreateKotlinNameRegistry() *core.NameRegistry {
	return core.NewNameRegistry(KotlinCollisionHandler)
}
