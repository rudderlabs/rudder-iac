package swift

import (
	"fmt"
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

// tokenize splits a string into words by spaces, underscores, hyphens, dots,
// and camelCase boundaries. Non-letter/digit characters are stripped.
func tokenize(s string) []string {
	// Replace common separators (including dots for decimal numbers) with spaces
	s = strings.NewReplacer("_", " ", "-", " ", ".", " ").Replace(s)

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
		r := []rune(t)
		if len(r) == 0 {
			continue
		}
		b.WriteRune(unicode.ToUpper(r[0]))
		b.WriteString(string(r[1:]))
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
		r := []rune(t)
		if i == 0 {
			b.WriteString(string(r))
		} else {
			b.WriteRune(unicode.ToUpper(r[0]))
			b.WriteString(string(r[1:]))
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
// Strings that tokenize to nothing (pure emoji/symbols) are encoded as Unicode codepoints.
// e.g. "GET" → "get", 200 → "n200", "1.5" → "n15", "🎯" → "u1F3AF"
func formatEnumCaseName(s string) string {
	name := formatPropertyName(s)

	// If tokenize yielded nothing (pure emoji/symbols), formatPropertyName returns
	// s unchanged and the result is not a valid identifier. Encode as codepoints.
	if len(tokenize(s)) == 0 {
		name = unicodeEscape(s)
	}

	if r := []rune(name); len(r) > 0 && unicode.IsDigit(r[0]) {
		name = "n" + name
	}
	return name
}

// unicodeEscape encodes s as "u" followed by the hex codepoints of its runes.
// Used as a fallback for strings that produce no valid identifier tokens.
func unicodeEscape(s string) string {
	var b strings.Builder
	b.WriteRune('u')
	for _, r := range s {
		fmt.Fprintf(&b, "%X", r)
	}
	return b.String()
}
