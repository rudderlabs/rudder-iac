package core

import (
	"regexp"
	"strings"
	"unicode"
)

// ToPascalCase converts a string to PascalCase (UpperCamelCase)
// Examples: "user_id" -> "UserId", "email-address" -> "EmailAddress"
// Supports Unicode characters correctly by using rune-based operations
func ToPascalCase(input string) string {
	words := SplitIntoWords(input)

	var result strings.Builder
	for _, word := range words {
		if len(word) > 0 {
			// Capitalize first rune, lowercase the rest
			runes := []rune(word)
			result.WriteString(string(unicode.ToUpper(runes[0])))
			if len(runes) > 1 {
				result.WriteString(strings.ToLower(string(runes[1:])))
			}
		}
	}

	return result.String()
}

// ToCamelCase converts a string to camelCase (lowerCamelCase)
// Examples: "user_id" -> "userId", "email-address" -> "emailAddress"
// Supports Unicode characters correctly by using rune-based operations
func ToCamelCase(input string) string {
	words := SplitIntoWords(input)

	if len(words) == 0 {
		return ""
	}

	var result strings.Builder

	// First word is lowercase
	if len(words[0]) > 0 {
		result.WriteString(strings.ToLower(words[0]))
	}

	// Subsequent words are capitalized
	for i := 1; i < len(words); i++ {
		word := words[i]
		if len(word) > 0 {
			// Capitalize first rune, lowercase the rest
			runes := []rune(word)
			result.WriteString(string(unicode.ToUpper(runes[0])))
			if len(runes) > 1 {
				result.WriteString(strings.ToLower(string(runes[1:])))
			}
		}
	}

	return result.String()
}

// SplitIntoWords splits a string into words based on various delimiters and case changes
// Note: If the input contains only underscores, this will return an empty slice
// since underscores are treated as delimiters
func SplitIntoWords(input string) []string {
	if input == "" {
		return []string{}
	}

	// Replace common delimiters with spaces
	re := regexp.MustCompile(`[_\-\s\.]+`)
	normalized := re.ReplaceAllString(input, " ")

	// Split on various boundaries:
	// 1. lowercase letter followed by uppercase letter (camelCase)
	camelCaseRe := regexp.MustCompile(`([a-z])([A-Z])`)
	normalized = camelCaseRe.ReplaceAllString(normalized, "$1 $2")

	// 2. number followed by uppercase letter (like "123Id" -> "123 Id")
	numberUpperRe := regexp.MustCompile(`([0-9])([A-Z])`)
	normalized = numberUpperRe.ReplaceAllString(normalized, "$1 $2")

	// 3. multiple uppercase letters followed by lowercase (like XMLHttp -> XML Http)
	acronymRe := regexp.MustCompile(`([A-Z]+)([A-Z][a-z])`)
	normalized = acronymRe.ReplaceAllString(normalized, "$1 $2")

	// Split on spaces and filter empty strings
	words := strings.Fields(normalized)

	result := make([]string, 0, len(words))
	for _, word := range words {
		if len(word) > 0 {
			result = append(result, word)
		}
	}

	return result
}

func ReplaceSpecialCharacters(input, replacement string) string {
	// Match Unicode letters (L), numbers (N), underscores, and whitespace
	// Replace everything else (punctuation, symbols, etc.) with the replacement
	// This preserves Cyrillic, Greek, Chinese, etc. while replacing special chars
	re := regexp.MustCompile(`[^\pL\pN_\s]`)
	// Replace special characters with the specified replacement string
	return re.ReplaceAllString(input, replacement)
}
