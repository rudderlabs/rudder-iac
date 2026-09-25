package swift

import (
	"fmt"
	"strings"
)

// EscapeSwiftStringLiteral escapes a string for use inside Swift string literals.
func EscapeSwiftStringLiteral(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}

// EscapeSwiftComment escapes content for /// doc comments (single-line style).
func EscapeSwiftComment(s string) string {
	return strings.ReplaceAll(s, "\n", " ")
}

// FormatSwiftLiteral formats a Go value as a Swift literal.
func FormatSwiftLiteral(value any) string {
	if value == nil {
		return "nil"
	}
	switch v := value.(type) {
	case string:
		return fmt.Sprintf(`"%s"`, EscapeSwiftStringLiteral(v))
	case int, int32, int64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%g", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
