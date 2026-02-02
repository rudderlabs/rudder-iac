package utils

import (
	"sort"
	"strings"
)

// SortableResource defines an interface for resources that can be sorted.
// will be extended with more sortable fields in the future like name, displayName, etc.
type SortableResource interface {
	GetLocalID() string
}

func SortByLocalID[T SortableResource](items []T) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetLocalID() < items[j].GetLocalID()
	})
}

// SortLexicographically sorts a slice of any type by comparing their string values.
// It expects that all elements in items are either strings or can be type casted to strings.
func SortLexicographically(items []any) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].(string) < items[j].(string)
	})
}

// ToSnakeCase converts a camelCase or PascalCase string to snake_case.
// Examples:
//   - minLength -> min_length
//   - maxLength -> max_length
//   - ExclusiveMaximum -> exclusive_maximum
//   - enum -> enum (already lowercase)
func ToSnakeCase(s string) string {
	if s == "" {
		return s
	}

	var result []rune
	for i, r := range s {
		// If uppercase and not first character
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result = append(result, '_')
			}
			// Convert to lowercase
			result = append(result, r+32)
		} else {
			result = append(result, r)
		}
	}

	return string(result)
}

// SplitMultiTypeString splits a comma-separated string of types, trims whitespace,
// and returns a slice of type strings.
func SplitMultiTypeString(typeString string) []string {
	typesList := strings.Split(typeString, ",")
	types := make([]string, len(typesList))
	for i, t := range typesList {
		types[i] = strings.TrimSpace(t)
	}
	return types
}

// ToCamelCase converts a snake_case string to camelCase.
// Examples:
//   - min_length -> minLength
//   - max_length -> maxLength
//   - exclusive_maximum -> exclusiveMaximum
//   - enum -> enum (already lowercase)
func ToCamelCase(s string) string {
	if s == "" {
		return s
	}

	var result []rune
	capitalizeNext := false
	for _, r := range s {
		if r == '_' {
			capitalizeNext = true
			continue
		}
		if capitalizeNext && r >= 'a' && r <= 'z' {
			// Convert to uppercase
			result = append(result, r-32)
			capitalizeNext = false
		} else {
			result = append(result, r)
			capitalizeNext = false
		}
	}

	return string(result)
}
