package config

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"

	catalogRules "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
)

var (
	// Legacy custom type reference pattern
	customTypeLegacyReferenceRegex = regexp.MustCompile(
		catalogRules.CustomTypeLegacyReferenceRegex)
)

// isNumber checks if value is any numeric type
func isNumber(val any) bool {
	switch val.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float32, float64:
		return true
	}
	return false
}

// isInteger checks if value is an integer (includes float64 that are whole numbers)
func isInteger(val any) bool {
	switch v := val.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float32:
		return float32(int(v)) == v
	case float64:
		return float64(int64(v)) == v
	}
	return false
}

// toNumber converts value to float64 if it's a number
func toNumber(val any) (float64, bool) {
	switch v := val.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}

// toInteger converts value to int64 if it's an integer
func toInteger(val any) (int64, bool) {
	switch v := val.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		return int64(v), true
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		return int64(v), true
	case float32:
		if float32(int(v)) == v {
			return int64(v), true
		}
	case float64:
		if float64(int64(v)) == v {
			return int64(v), true
		}
	}
	return 0, false
}

// isValidFormat checks if format value is recognized
func isValidFormat(format string) bool {
	return slices.Contains(catalogRules.ValidFormatValues, format)
}

// isValidPrimitiveType checks if type is a valid primitive
func isValidPrimitiveType(typeName string) bool {
	return slices.Contains(catalogRules.ValidPrimitiveTypes, typeName)
}

// findDuplicateIndices checks for duplicate values in an array using reflection
// Returns the indices of all duplicate values (later occurrences)
func findDuplicateIndices(arr []any) []int {
	seen := make(map[any]int)
	duplicates := []int{}

	for i, val := range arr {
		// Use reflection to get a comparable key
		key := getComparableKey(val)

		if _, exists := seen[key]; exists {
			// Record the index of the duplicate (the later occurrence)
			duplicates = append(duplicates, i)
		} else {
			seen[key] = i
		}
	}

	return duplicates
}

// getComparableKey converts a value to a comparable key for duplicate detection
func getComparableKey(val any) any {
	// Use reflection to handle different types
	v := reflect.ValueOf(val)

	switch v.Kind() {
	case reflect.String:
		return val.(string)
	case reflect.Bool:
		return val.(bool)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint()
	case reflect.Float32, reflect.Float64:
		return v.Float()
	default:
		// For complex types, use the value itself as string representation
		return fmt.Sprintf("%v", val)
	}
}
