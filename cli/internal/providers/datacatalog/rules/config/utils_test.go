package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNumericTypeChecks combines testing for isNumber, isInteger, toNumber, and toInteger
// to reduce redundancy while maintaining full coverage of numeric type handling
func TestNumericTypeChecks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    any
		isNum    bool    // isNumber result
		isInt    bool    // isInteger result
		toNumVal float64 // toNumber value
		toNumOk  bool    // toNumber ok flag
		toIntVal int64   // toInteger value
		toIntOk  bool    // toInteger ok flag
	}{
		// All signed integer types - test complete type coverage
		{"int", int(42), true, true, 42.0, true, 42, true},
		{"int8", int8(-100), true, true, -100.0, true, -100, true},
		{"int16", int16(1000), true, true, 1000.0, true, 1000, true},
		{"int32", int32(-50000), true, true, -50000.0, true, -50000, true},
		{"int64", int64(999999), true, true, 999999.0, true, 999999, true},

		// All unsigned integer types
		{"uint", uint(99), true, true, 99.0, true, 99, true},
		{"uint8", uint8(255), true, true, 255.0, true, 255, true},
		{"uint16", uint16(65535), true, true, 65535.0, true, 65535, true},
		{"uint32", uint32(100000), true, true, 100000.0, true, 100000, true},
		{"uint64", uint64(123456789), true, true, 123456789.0, true, 123456789, true},

		// Float types with whole numbers
		{"float32 whole", float32(10.0), true, true, float64(float32(10.0)), true, 10, true},
		{"float64 whole", float64(5.0), true, true, 5.0, true, 5, true},
		{"negative whole float", float64(-10.0), true, true, -10.0, true, -10, true},
		{"zero float", float64(0.0), true, true, 0.0, true, 0, true},

		// Fractional floats - isNum=true, isInt=false, only toNumber succeeds
		{"fractional float", float64(3.14), true, false, 3.14, true, 0, false},
		{"small fraction", float64(1.001), true, false, 1.001, true, 0, false},
		{"float32 fraction", float32(2.5), true, false, float64(float32(2.5)), true, 0, false},

		// Non-numeric - all return false/fail
		{"string", "123", false, false, 0.0, false, 0, false},
		{"bool", true, false, false, 0.0, false, 0, false},
		{"nil", nil, false, false, 0.0, false, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test isNumber
			assert.Equal(t, tt.isNum, isNumber(tt.value), "isNumber mismatch for %v", tt.value)

			// Test isInteger
			assert.Equal(t, tt.isInt, isInteger(tt.value), "isInteger mismatch for %v", tt.value)

			// Test toNumber
			numVal, numOk := toNumber(tt.value)
			assert.Equal(t, tt.toNumOk, numOk, "toNumber ok flag mismatch for %v", tt.value)
			if numOk {
				assert.Equal(t, tt.toNumVal, numVal, "toNumber value mismatch for %v", tt.value)
			}

			// Test toInteger
			intVal, intOk := toInteger(tt.value)
			assert.Equal(t, tt.toIntOk, intOk, "toInteger ok flag mismatch for %v", tt.value)
			if intOk {
				assert.Equal(t, tt.toIntVal, intVal, "toInteger value mismatch for %v", tt.value)
			}
		})
	}
}

// TestValidationHelpers combines isValidFormat and isValidPrimitiveType
// since both use identical patterns (slices.Contains)
func TestValidationHelpers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		format   string // for isValidFormat
		formatOk bool
		primType string // for isValidPrimitiveType
		primOk   bool
	}{
		// Valid in both
		{"valid format & primitive", "email", true, "string", true},
		{"datetime & number", "date-time", true, "number", true},
		{"uuid & boolean", "uuid", true, "boolean", true},
		{"date & integer", "date", true, "integer", true},
		{"hostname & array", "hostname", true, "array", true},

		// Valid format, invalid primitive
		{"format only", "ipv4", true, "float64", false},
		{"time & custom", "time", true, "custom-type", false},

		// Invalid format, valid primitive
		{"primitive only", "url", false, "object", true},
		{"json & null", "json", false, "null", true},

		// Both invalid
		{"both invalid", "foobar", false, "datetime", false},
		{"empty strings", "", false, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test isValidFormat
			assert.Equal(t, tt.formatOk, isValidFormat(tt.format), "isValidFormat mismatch for %q", tt.format)

			// Test isValidPrimitiveType
			assert.Equal(t, tt.primOk, isValidPrimitiveType(tt.primType), "isValidPrimitiveType mismatch for %q", tt.primType)
		})
	}
}

// TestFindDuplicateIndices verifies duplicate detection across different types
func TestFindDuplicateIndices(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		array    []any
		expected []int
	}{
		{"no duplicates", []any{1, 2, 3}, []int{}},
		{"empty array", []any{}, []int{}},
		{"single duplicate", []any{1, 2, 1}, []int{2}},
		{"multiple same value", []any{"a", "b", "a", "c", "a"}, []int{2, 4}},
		{"multiple different duplicates", []any{1, 2, 1, 3, 2}, []int{2, 4}},
		{"all same", []any{5, 5, 5, 5}, []int{1, 2, 3}},
		{"mixed types", []any{true, 1, true, "x", 1}, []int{2, 4}},
		{"float duplicates", []any{3.14, 2.5, 3.14}, []int{2}},
		{"bool duplicates", []any{false, true, false, true}, []int{2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := findDuplicateIndices(tt.array)
			assert.Equal(t, tt.expected, result, "findDuplicateIndices mismatch for %v", tt.array)
		})
	}
}

// TestGetComparableKey verifies key conversion for reflection-based comparison
// Minimal testing since this is primarily exercised through findDuplicateIndices
func TestGetComparableKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		value        any
		checkType    bool // whether to check exact type
		expectedType any  // for type assertion
	}{
		{"string", "hello", true, ""},
		{"bool", true, true, true},
		{"int returns int64", int(42), true, int64(0)},
		{"uint returns uint64", uint(99), true, uint64(0)},
		{"float returns float64", float32(3.14), true, float64(0)},
		{"complex types", struct{ name string }{name: "test"}, false, nil},
		{"slice", []int{1, 2, 3}, false, nil},
		{"map", map[string]int{"a": 1}, false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			key := getComparableKey(tt.value)

			// Always returns something (non-nil/non-zero)
			assert.NotNil(t, key, "getComparableKey should never return nil")

			// Type checks for primitives
			if tt.checkType {
				assert.IsType(t, tt.expectedType, key, "getComparableKey type mismatch for %v", tt.value)
			}
		})
	}
}
