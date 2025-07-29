package kotlin_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/kotlin"
	"github.com/stretchr/testify/assert"
)

func TestFormatClassName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"snake_case", "user_id", "UserId"},
		{"kebab-case", "email-address", "EmailAddress"},
		{"space separated", "first name", "FirstName"},
		{"camelCase", "firstName", "FirstName"},
		{"PascalCase", "FirstName", "FirstName"},
		{"single word", "user", "User"},
		{"with numbers", "user123", "User123"},
		{"number at end", "user123Id", "User123Id"},
		{"leading number", "123user", "_123user"},
		{"reserved keyword", "class", "_Class"},
		{"reserved keyword uppercase", "Class", "_Class"},
		{"reserved keyword mixed", "CLASS", "_Class"},
		{"empty string", "", ""},
		{"whitespace only", "   ", ""},
		{"complex name", "user_email-address.type", "UserEmailAddressType"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := kotlin.FormatClassName("", tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatClassName_ReservedKeywords(t *testing.T) {
	// Test various reserved keywords
	reservedKeywords := []string{
		"class", "interface", "object", "fun", "val", "var",
		"if", "else", "when", "for", "while", "try", "catch",
	}

	for _, keyword := range reservedKeywords {
		t.Run(keyword, func(t *testing.T) {
			result := kotlin.FormatClassName("", keyword)
			assert.Equal(t, byte('_'), result[0], "Reserved keyword should be prefixed with underscore")
			assert.NotEqual(t, keyword, result, "Result should not be the same as reserved keyword")
		})
	}
}
