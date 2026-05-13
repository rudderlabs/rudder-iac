package typescript

import (
	"strings"
	"unicode"
)

// tokenize splits a string into words by spaces, underscores, hyphens, dots,
// and camelCase boundaries. Non-letter/digit characters are stripped from word
// edges (but not from the middle of a token), and all tokens are lowercased.
//
// Mirrors Swift's tokenize: kept separate from core.SplitIntoWords so acronym
// runs (e.g. "XMLParser") collapse to a single token rather than splitting,
// which keeps generated identifiers stable for plans that contain acronyms.
func tokenize(s string) []string {
	s = strings.NewReplacer("_", " ", "-", " ", ".", " ").Replace(s)

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

// FormatTypeName converts prefix and name to PascalCase for use as a TypeScript
// type/interface/class name. e.g. ("Identify", "Traits") → "IdentifyTraits".
// prefix may be empty.
//
// PascalCase identifiers are not escaped: TS reserved words are all-lowercase
// keywords, so PascalCase output cannot collide with them.
func FormatTypeName(prefix, name string) string {
	combined := name
	if prefix != "" {
		combined = prefix + " " + name
	}
	tokens := tokenize(combined)
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

// FormatPropertyName converts a string to camelCase for TypeScript
// property/variable names. e.g. "device_type" → "deviceType". When the result
// would be a reserved word, an underscore suffix is appended ("class" →
// "class_") to avoid collision with TS keywords or built-in types.
func FormatPropertyName(s string) string {
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
	if tsReservedWords[name] {
		return name + "_"
	}
	return name
}

// FormatMethodName converts prefix and name to camelCase for TypeScript method
// names. e.g. ("track", "User Signed Up") → "trackUserSignedUp". prefix may be
// empty.
func FormatMethodName(prefix, name string) string {
	combined := name
	if prefix != "" {
		combined = prefix + " " + name
	}
	return FormatPropertyName(combined)
}
