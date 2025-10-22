package kotlin

import (
	"fmt"
	"strings"
)

// EscapeKotlinStringLiteral escapes a string for use in Kotlin string literals.
// It handles backslashes, quotes, newlines, tabs, carriage returns, and other special characters
// to ensure the resulting string is valid Kotlin syntax.
//
// Examples:
//   - `Hello "World"` → `Hello \"World\"`
//   - `Path\To\File` → `Path\\To\\File`
//   - `Line 1\nLine 2` → `Line 1\\nLine 2`
func EscapeKotlinStringLiteral(s string) string {
	var builder strings.Builder
	builder.Grow(len(s) + 10) // Pre-allocate with some extra space

	for _, ch := range s {
		switch ch {
		case '\\':
			builder.WriteString(`\\`)
		case '"':
			builder.WriteString(`\"`)
		case '\n':
			builder.WriteString(`\n`)
		case '\r':
			builder.WriteString(`\r`)
		case '\t':
			builder.WriteString(`\t`)
		case '\b':
			builder.WriteString(`\b`)
		case '\f':
			builder.WriteString(`\f`)
		default:
			builder.WriteRune(ch)
		}
	}

	return builder.String()
}

// EscapeKotlinComment escapes content for use in KDoc comments.
// It prevents premature comment closure by escaping the `*/` sequence,
// which would otherwise terminate the comment block.
//
// Examples:
//   - `User's email /* important */` → `User's email /* important *\/`
//   - `Calculate a * b / c` → `Calculate a * b / c` (unchanged)
func EscapeKotlinComment(s string) string {
	// Replace */ with *\/ to prevent premature comment closure
	// We also escape /* to prevent confusion, though it's not strictly necessary
	s = strings.ReplaceAll(s, "*/", `*\/`)
	s = strings.ReplaceAll(s, "/*", `/\*`)
	return s
}

// FormatKotlinLiteral formats a Go value as a Kotlin literal.
// This function assumes the value IS a literal (not a variable reference).
//
// Supported types:
//   - string: formats as escaped string literal "value"
//   - int, int32, int64: formats as integer literal
//   - float32, float64: formats as floating point literal (trailing zeros removed)
//   - bool: formats as boolean literal (true/false)
//   - nil: returns empty string
//
// Examples:
//   - `"Hello"` (string) → `"Hello"`
//   - `42` (int) → `42`
//   - `3.14` (float) → `3.14`
//   - `true` (bool) → `true`
//   - `nil` → ``
func FormatKotlinLiteral(value any) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		// String literal: escape and wrap in quotes
		return fmt.Sprintf(`"%s"`, EscapeKotlinStringLiteral(v))
	case int:
		return fmt.Sprintf("%d", v)
	case int32:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float32:
		// Format with %g to remove trailing zeros
		return fmt.Sprintf("%g", v)
	case float64:
		// Format with %g to remove trailing zeros
		return fmt.Sprintf("%g", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		// Fallback: convert to string
		return fmt.Sprintf("%v", v)
	}
}
