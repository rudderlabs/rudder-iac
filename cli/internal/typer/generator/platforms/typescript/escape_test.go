package typescript_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/platforms/typescript"
	"github.com/stretchr/testify/assert"
)

func TestEscapeTSStringLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"plain string", "Hello World", "Hello World"},
		{"double quotes", `Product "Premium" Clicked`, `Product \"Premium\" Clicked`},
		{"backslash", `Path\To\Resource`, `Path\\To\\Resource`},
		{"newline", "Line 1\nLine 2", `Line 1\nLine 2`},
		{"carriage return", "Windows\r\nLine", `Windows\r\nLine`},
		{"tab", "Column1\tColumn2", `Column1\tColumn2`},
		{"backslash before quote", `say \"hello\"`, `say \\\"hello\\\"`},
		{"only special characters", `"\n\t\r\\`, `\"\\n\\t\\r\\\\`},
		{"unicode characters pass through", "Hello 世界 🌍", "Hello 世界 🌍"},
		{"multiple escapes combined", "Line 1\nHe said \"hi\"\t\\done", `Line 1\nHe said \"hi\"\t\\done`},
		{"real-world event name with quotes", `User "Premium" Subscription Started`, `User \"Premium\" Subscription Started`},
		{"Windows path", `C:\Program Files\MyApp\config.json`, `C:\\Program Files\\MyApp\\config.json`},
		// Generator emits double-quoted strings only; $ has no special meaning there.
		{"dollar sign passes through", "Price: $100", "Price: $100"},
		{"dollar with braces passes through", "${expression}", "${expression}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, typescript.EscapeTSStringLiteral(tt.input))
		})
	}
}

func TestEscapeJSDocComment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"plain comment", "Triggered when a user signs up", "Triggered when a user signs up"},
		{"close-comment escaped", "User's email /* important */", `User's email /\* important *\/`},
		{"open-comment escaped", "Calculate /* note */ next", `Calculate /\* note *\/ next`},
		{"unrelated slash and star", "Calculate a * b / c", "Calculate a * b / c"},
		{"unicode in comment", "User's 世界 data", "User's 世界 data"},
		{"newline collapsed to space", "First line\nSecond line", "First line Second line"},
		{"carriage return collapsed to space", "Windows\r\nLine", "Windows  Line"},
		{"multi-line with comment markers", "First line\n*/ second", `First line *\/ second`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, typescript.EscapeJSDocComment(tt.input))
		})
	}
}

func TestFormatTSLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"nil value", nil, "null"},
		{"empty string", "", `""`},
		{"plain string", "Hello World", `"Hello World"`},
		{"string with double quotes", `Product "Premium"`, `"Product \"Premium\""`},
		{"string with backslash", `C:\Path\To\File`, `"C:\\Path\\To\\File"`},
		{"string with newline", "Line 1\nLine 2", `"Line 1\nLine 2"`},
		{"unicode string", "Hello 世界", `"Hello 世界"`},
		{"zero", 0, "0"},
		{"positive integer", 42, "42"},
		{"negative integer", -123, "-123"},
		{"int32", int32(100), "100"},
		{"int64", int64(999999), "999999"},
		{"float32", float32(3.14), "3.14"},
		{"float64", float64(2.71828), "2.71828"},
		{"whole-number float", float64(1.0), "1"},
		{"boolean true", true, "true"},
		{"boolean false", false, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, typescript.FormatTSLiteral(tt.input))
		})
	}
}
