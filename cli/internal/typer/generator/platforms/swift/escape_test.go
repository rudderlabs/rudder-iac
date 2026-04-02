package swift_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/platforms/swift"
	"github.com/stretchr/testify/assert"
)

func TestEscapeSwiftStringLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "plain string",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "double quotes",
			input:    `Product "Premium" Clicked`,
			expected: `Product \"Premium\" Clicked`,
		},
		{
			name:     "backslash",
			input:    `Path\To\Resource`,
			expected: `Path\\To\\Resource`,
		},
		{
			name:     "newline",
			input:    "Line 1\nLine 2",
			expected: `Line 1\nLine 2`,
		},
		{
			name:     "carriage return",
			input:    "Windows\r\nLine",
			expected: `Windows\r\nLine`,
		},
		{
			name:     "tab",
			input:    "Column1\tColumn2",
			expected: `Column1\tColumn2`,
		},
		{
			name:     "backslash before quote",
			input:    `say \"hello\"`,
			expected: `say \\\"hello\\\"`,
		},
		{
			name:     "only special characters",
			input:    `"\n\t\r\\`,
			expected: `\"\\n\\t\\r\\\\`,
		},
		{
			name:     "unicode characters pass through",
			input:    "Hello 世界 🌍",
			expected: "Hello 世界 🌍",
		},
		{
			name:     "multiple escapes combined",
			input:    "Line 1\nHe said \"hi\"\t\\done",
			expected: `Line 1\nHe said \"hi\"\t\\done`,
		},
		{
			name:     "real-world event name with quotes",
			input:    `User "Premium" Subscription Started`,
			expected: `User \"Premium\" Subscription Started`,
		},
		{
			name:     "Windows path",
			input:    `C:\Program Files\MyApp\config.json`,
			expected: `C:\\Program Files\\MyApp\\config.json`,
		},
		// Swift uses \() for interpolation, not $, so $ needs no escaping
		{
			name:     "dollar sign passes through",
			input:    "Price: $100",
			expected: "Price: $100",
		},
		{
			name:     "dollar sign with braces passes through",
			input:    "${expression}",
			expected: "${expression}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, swift.EscapeSwiftStringLiteral(tt.input))
		})
	}
}

func TestEscapeSwiftComment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "plain comment",
			input:    "Triggered when a user signs up",
			expected: "Triggered when a user signs up",
		},
		{
			name:     "newline replaced with space",
			input:    "First line\nSecond line",
			expected: "First line Second line",
		},
		{
			name:     "multiple newlines",
			input:    "Line 1\nLine 2\nLine 3",
			expected: "Line 1 Line 2 Line 3",
		},
		{
			name:     "carriage return and newline both replaced",
			input:    "Windows\r\nLine",
			expected: "Windows\r Line",
		},
		{
			name:     "unicode in comment",
			input:    "User's 世界 data",
			expected: "User's 世界 data",
		},
		{
			name:     "quote and backslash unchanged",
			input:    `User's "premium" /* note */`,
			expected: `User's "premium" /* note */`,
		},
		{
			name:     "real-world multiline description",
			input:    "Triggered when a user clicks.\nSee docs for details.",
			expected: "Triggered when a user clicks. See docs for details.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, swift.EscapeSwiftComment(tt.input))
		})
	}
}

func TestFormatSwiftLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		// nil
		{
			name:     "nil value",
			input:    nil,
			expected: "nil",
		},
		// strings
		{
			name:     "empty string",
			input:    "",
			expected: `""`,
		},
		{
			name:     "plain string",
			input:    "Hello World",
			expected: `"Hello World"`,
		},
		{
			name:     "string with double quotes",
			input:    `Product "Premium"`,
			expected: `"Product \"Premium\""`,
		},
		{
			name:     "string with backslash",
			input:    `C:\Path\To\File`,
			expected: `"C:\\Path\\To\\File"`,
		},
		{
			name:     "string with newline",
			input:    "Line 1\nLine 2",
			expected: `"Line 1\nLine 2"`,
		},
		{
			name:     "dollar sign not escaped in Swift",
			input:    "Price: $100",
			expected: `"Price: $100"`,
		},
		{
			name:     "unicode string",
			input:    "Hello 世界",
			expected: `"Hello 世界"`,
		},
		// integers
		{
			name:     "zero",
			input:    0,
			expected: "0",
		},
		{
			name:     "positive integer",
			input:    42,
			expected: "42",
		},
		{
			name:     "negative integer",
			input:    -123,
			expected: "-123",
		},
		{
			name:     "int32",
			input:    int32(100),
			expected: "100",
		},
		{
			name:     "int64",
			input:    int64(999999),
			expected: "999999",
		},
		// floats
		{
			name:     "float32",
			input:    float32(3.14),
			expected: "3.14",
		},
		{
			name:     "float64",
			input:    float64(2.71828),
			expected: "2.71828",
		},
		{
			name:     "whole-number float",
			input:    float64(1.0),
			expected: "1",
		},
		// booleans
		{
			name:     "boolean true",
			input:    true,
			expected: "true",
		},
		{
			name:     "boolean false",
			input:    false,
			expected: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, swift.FormatSwiftLiteral(tt.input))
		})
	}
}
