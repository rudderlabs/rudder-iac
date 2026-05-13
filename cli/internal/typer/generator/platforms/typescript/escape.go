package typescript

import (
	"fmt"
	"strings"
)

// EscapeTSStringLiteral escapes a string for use inside TypeScript double-quoted
// string literals. Backticks are not handled because the generator does not emit
// template literals.
func EscapeTSStringLiteral(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}

// EscapeJSDocComment escapes content for use inside a /** ... */ JSDoc block.
// Newlines are collapsed to spaces so multi-line descriptions stay on the
// single comment line emitted by the templates (mirrors Swift's
// EscapeSwiftComment); `*/` and `/*` are escaped so a description that
// contains them cannot terminate or nest the comment block.
func EscapeJSDocComment(s string) string {
	s = strings.ReplaceAll(s, "*/", `*\/`)
	s = strings.ReplaceAll(s, "/*", `/\*`)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}

// FormatTSLiteral formats a Go value as a TypeScript literal expression.
func FormatTSLiteral(value any) string {
	if value == nil {
		return "null"
	}
	switch v := value.(type) {
	case string:
		return fmt.Sprintf(`"%s"`, EscapeTSStringLiteral(v))
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
