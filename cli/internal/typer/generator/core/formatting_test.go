package core_test

import (
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/stretchr/testify/assert"
)

func TestToPascalCase(t *testing.T) {
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
		{"mixed delimiters", "user_id-email.address", "UserIdEmailAddress"},
		{"single word", "user", "User"},
		{"empty string", "", ""},
		{"numbers", "user123_id", "User123Id"},
		{"consecutive delimiters", "user__id", "UserId"},
		{"leading/trailing delimiters", "_user_id_", "UserId"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := core.ToPascalCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"snake_case", "user_id", "userId"},
		{"kebab-case", "email-address", "emailAddress"},
		{"space separated", "first name", "firstName"},
		{"camelCase", "firstName", "firstName"},
		{"PascalCase", "FirstName", "firstName"},
		{"mixed delimiters", "user_id-email.address", "userIdEmailAddress"},
		{"single word", "user", "user"},
		{"empty string", "", ""},
		{"numbers", "user123_id", "user123Id"},
		{"consecutive delimiters", "user__id", "userId"},
		{"leading/trailing delimiters", "_user_id_", "userId"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := core.ToCamelCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitIntoWords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"snake_case", "user_id", []string{"user", "id"}},
		{"kebab-case", "email-address", []string{"email", "address"}},
		{"space separated", "first name", []string{"first", "name"}},
		{"camelCase", "firstName", []string{"first", "Name"}},
		{"PascalCase", "FirstName", []string{"First", "Name"}},
		{"mixed delimiters", "user_id-email.address", []string{"user", "id", "email", "address"}},
		{"single word", "user", []string{"user"}},
		{"empty string", "", []string{}},
		{"numbers", "user123Id", []string{"user123", "Id"}},
		{"consecutive delimiters", "user__id", []string{"user", "id"}},
		{"leading/trailing delimiters", "_user_id_", []string{"user", "id"}},
		{"dots", "com.example.package", []string{"com", "example", "package"}},
		{"complex case", "XMLHttpRequest", []string{"XML", "Http", "Request"}},
		{"acronyms", "HTTPSConnection", []string{"HTTPS", "Connection"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := core.SplitIntoWords(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitIntoWords_EdgeCases(t *testing.T) {
	// Test edge cases that might cause issues
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"only delimiters", "___---...", []string{}},
		{"numbers only", "123", []string{"123"}},
		{"mixed case with numbers", "HTML5Parser", []string{"HTML5", "Parser"}},
		{"single character", "a", []string{"a"}},
		{"uppercase only", "XML", []string{"XML"}},
		{"lowercase only", "xml", []string{"xml"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := core.SplitIntoWords(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormattingConsistency(t *testing.T) {
	// Test that ToPascalCase and ToCamelCase are consistent
	inputs := []string{
		"user_id",
		"email-address",
		"first name",
		"XMLHttpRequest",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			pascal := core.ToPascalCase(input)
			camel := core.ToCamelCase(input)

			// Pascal case should be camel case with first letter uppercase
			if len(camel) > 0 && len(pascal) > 0 {
				expectedPascal := strings.ToUpper(string(camel[0])) + camel[1:]
				assert.Equal(t, expectedPascal, pascal,
					"ToPascalCase should be ToCamelCase with first letter capitalized")
			}
		})
	}
}
