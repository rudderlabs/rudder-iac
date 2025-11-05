package kotlin

import (
	"testing"
)

func TestEscapeKotlinStringLiteral(t *testing.T) {
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
			name:     "simple string",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "string with quotes",
			input:    `Product "Premium" Clicked`,
			expected: `Product \"Premium\" Clicked`,
		},
		{
			name:     "string with backslashes",
			input:    `Path\To\Resource`,
			expected: `Path\\To\\Resource`,
		},
		{
			name:     "string with newlines",
			input:    "Line 1\nLine 2\nLine 3",
			expected: `Line 1\nLine 2\nLine 3`,
		},
		{
			name:     "string with tabs",
			input:    "Column1\tColumn2\tColumn3",
			expected: `Column1\tColumn2\tColumn3`,
		},
		{
			name:     "string with carriage return",
			input:    "Windows\r\nLine",
			expected: `Windows\r\nLine`,
		},
		{
			name:     "string with backspace",
			input:    "Text\bwith\bbackspace",
			expected: `Text\bwith\bbackspace`,
		},
		{
			name:     "string with form feed",
			input:    "Page1\fPage2",
			expected: `Page1\fPage2`,
		},
		{
			name:     "multiple special characters with literal backslash-n",
			input:    `Quote: " Backslash: \ Newline: \n`,
			expected: `Quote: \" Backslash: \\ Newline: \\n`,
		},
		{
			name:     "only special characters with literal backslashes",
			input:    `"\n\t\r\\`,
			expected: `\"\\n\\t\\r\\\\`,
		},
		{
			name:     "unicode characters",
			input:    "Hello 世界 🌍",
			expected: "Hello 世界 🌍",
		},
		{
			name:     "mixed unicode and special chars with literal backslash-n",
			input:    `"Hello 世界"\nNext line`,
			expected: `\"Hello 世界\"\\nNext line`,
		},
		{
			name:     "real-world event name",
			input:    `User "Premium" Subscription Started`,
			expected: `User \"Premium\" Subscription Started`,
		},
		{
			name:     "real-world path",
			input:    `C:\Program Files\MyApp\config.json`,
			expected: `C:\\Program Files\\MyApp\\config.json`,
		},
		{
			name:     "string with dollar sign",
			input:    "Price: $100",
			expected: `Price: \$100`,
		},
		{
			name:     "string with expression-like syntax",
			input:    "${expression}",
			expected: `\${expression}`,
		},
		{
			name:     "string with multiple dollar signs",
			input:    "$USD $100 $EUR",
			expected: `\$USD \$100 \$EUR`,
		},
		{
			name:     "mixed special characters with dollar signs",
			input:    `Path: "C:\$Folder" and $variable`,
			expected: `Path: \"C:\\\$Folder\" and \$variable`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapeKotlinStringLiteral(tt.input)
			if result != tt.expected {
				t.Errorf("EscapeKotlinStringLiteral(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEscapeKotlinComment(t *testing.T) {
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
			name:     "simple comment",
			input:    "This is a simple comment",
			expected: "This is a simple comment",
		},
		{
			name:     "comment with closing sequence",
			input:    "User's email /* important */",
			expected: `User's email /\* important *\/`,
		},
		{
			name:     "comment with multiple closing sequences",
			input:    "First */ and second */ sequences",
			expected: `First *\/ and second *\/ sequences`,
		},
		{
			name:     "comment with opening sequence",
			input:    "Calculate /* intermediate */ result",
			expected: `Calculate /\* intermediate *\/ result`,
		},
		{
			name:     "only special sequences",
			input:    "/* */",
			expected: `/\* *\/`,
		},
		{
			name:     "asterisk without slash",
			input:    "Calculate a * b * c",
			expected: "Calculate a * b * c",
		},
		{
			name:     "slash without asterisk",
			input:    "Calculate a / b / c",
			expected: "Calculate a / b / c",
		},
		{
			name:     "unicode in comment",
			input:    "User's 世界 data /* 重要 */",
			expected: `User's 世界 data /\* 重要 *\/`,
		},
		{
			name:     "real-world comment",
			input:    "The user's email address /* must be validated */",
			expected: `The user's email address /\* must be validated *\/`,
		},
		{
			name:     "nested comment-like structure",
			input:    "/* outer /* inner */ outer */",
			expected: `/\* outer /\* inner *\/ outer *\/`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapeKotlinComment(tt.input)
			if result != tt.expected {
				t.Errorf("EscapeKotlinComment(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatKotlinLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "nil value",
			input:    nil,
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: `""`,
		},
		{
			name:     "simple string",
			input:    "Hello World",
			expected: `"Hello World"`,
		},
		{
			name:     "string with quotes",
			input:    `Product "Premium"`,
			expected: `"Product \"Premium\""`,
		},
		{
			name:     "string with backslashes",
			input:    `C:\Path\To\File`,
			expected: `"C:\\Path\\To\\File"`,
		},
		{
			name:     "string with newline",
			input:    "Line 1\nLine 2",
			expected: `"Line 1\nLine 2"`,
		},
		{
			name:     "integer",
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
			name:     "boolean true",
			input:    true,
			expected: "true",
		},
		{
			name:     "boolean false",
			input:    false,
			expected: "false",
		},
		{
			name:     "zero integer",
			input:    0,
			expected: "0",
		},
		{
			name:     "unicode string",
			input:    "Hello 世界",
			expected: `"Hello 世界"`,
		},
		{
			name:     "event name with special chars",
			input:    `User "Premium" Event`,
			expected: `"User \"Premium\" Event"`,
		},
		{
			name:     "string with dollar sign",
			input:    "Price: $100",
			expected: `"Price: \$100"`,
		},
		{
			name:     "string with variable-like syntax",
			input:    "$variable",
			expected: `"\$variable"`,
		},
		{
			name:     "string with expression-like syntax",
			input:    "${expression}",
			expected: `"\${expression}"`,
		},
		{
			name:     "event name with dollar signs",
			input:    "$Variable$String",
			expected: `"\$Variable\$String"`,
		},
		{
			name:     "mixed special characters with dollar signs",
			input:    `Text with "quotes" and $variable`,
			expected: `"Text with \"quotes\" and \$variable"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatKotlinLiteral(tt.input)
			if result != tt.expected {
				t.Errorf("FormatKotlinLiteral(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
