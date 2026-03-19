package swift

import (
	"strings"
	"unicode"
)

var swiftReservedWords = map[string]bool{
	"as": true, "associatedtype": true, "break": true, "case": true,
	"catch": true, "class": true, "continue": true, "default": true,
	"defer": true, "deinit": true, "do": true, "else": true,
	"enum": true, "extension": true, "fallthrough": true, "false": true,
	"fileprivate": true, "for": true, "func": true, "guard": true,
	"if": true, "import": true, "in": true, "init": true, "inout": true,
	"internal": true, "is": true, "let": true, "nil": true, "open": true,
	"operator": true, "private": true, "protocol": true, "public": true,
	"repeat": true, "rethrows": true, "return": true, "self": true,
	"Self": true, "static": true, "struct": true, "subscript": true,
	"super": true, "switch": true, "throw": true, "throws": true,
	"true": true, "try": true, "typealias": true, "var": true,
	"where": true, "while": true, "Any": true, "Type": true,
}

// tokenize splits a string into words by spaces, underscores, hyphens,
// and camelCase boundaries.
func tokenize(s string) []string {
	// Replace separators with spaces
	s = strings.NewReplacer("_", " ", "-", " ").Replace(s)

	// Insert space before uppercase letters that follow lowercase (camelCase split)
	var spaced strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) && unicode.IsLower(runes[i-1]) {
			spaced.WriteRune(' ')
		}
		spaced.WriteRune(r)
	}

	var tokens []string
	for _, word := range strings.Fields(spaced.String()) {
		word = strings.TrimFunc(word, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})
		if word != "" {
			tokens = append(tokens, strings.ToLower(word))
		}
	}
	return tokens
}

// FormatTypeName converts a string to PascalCase for use as a Swift type name.
// e.g. "user signed up" → "UserSignedUp"
func FormatTypeName(s string) string {
	tokens := tokenize(s)
	var b strings.Builder
	for _, t := range tokens {
		if len(t) == 0 {
			continue
		}
		b.WriteString(strings.ToUpper(t[:1]) + t[1:])
	}
	return b.String()
}

// formatPropertyName converts a string to camelCase for Swift property/variable names.
// e.g. "device_type" → "deviceType"
func formatPropertyName(s string) string {
	tokens := tokenize(s)
	if len(tokens) == 0 {
		return s
	}
	var b strings.Builder
	for i, t := range tokens {
		if len(t) == 0 {
			continue
		}
		if i == 0 {
			b.WriteString(t)
		} else {
			b.WriteString(strings.ToUpper(t[:1]) + t[1:])
		}
	}
	name := b.String()
	if swiftReservedWords[name] {
		return "`" + name + "`"
	}
	return name
}

// formatMethodName converts a string to camelCase for Swift method names.
// e.g. "Track User Signed Up" → "trackUserSignedUp"
func formatMethodName(s string) string {
	return formatPropertyName(s)
}

// formatEnumCaseName converts a value to a valid Swift enum case name.
// Numeric values get an "n" prefix. Reserved words get backtick escaping.
// e.g. "GET" → "get", 200 → "n200"
func formatEnumCaseName(s string) string {
	name := formatPropertyName(s)
	if len(name) > 0 && unicode.IsDigit(rune(name[0])) {
		name = "n" + name
	}
	return name
}
