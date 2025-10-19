package core_test

import (
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
		// Unicode test cases - ensure proper rune-based handling
		{"chinese_characters", "用户_标识", "用户标识"},
		{"cyrillic_characters", "тип_данных", "ТипДанных"},
		{"greek_characters", "όνομα_χρήστη", "ΌνομαΧρήστη"},
		{"arabic_characters", "اسم_المستخدم", "اسمالمستخدم"},
		{"mixed_unicode_ascii", "user_用户_id", "User用户Id"},
		{"latin_diacritics", "café_crème", "CaféCrème"},
		{"japanese_katakana", "ユーザー_識別子", "ユーザー識別子"},
		{"emoji_preserved", "🎯", "🎯"},
		{"hebrew_characters", "שם_משתמש", "שםמשתמש"},
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
		// Unicode test cases - ensure proper rune-based handling
		{"chinese_characters", "用户_标识", "用户标识"},
		{"cyrillic_characters", "получить_данные", "получитьДанные"},
		{"greek_characters", "όνομα_χρήστη", "όνομαΧρήστη"},
		{"arabic_characters", "اسم_المستخدم", "اسمالمستخدم"},
		{"mixed_unicode_ascii", "get_用户_data", "get用户Data"},
		{"latin_diacritics", "café_créme", "caféCréme"},
		{"japanese_hiragana", "データ_取得", "データ取得"},
		{"emoji_preserved", "🚀_launch", "🚀Launch"},
		{"thai_characters", "ชื่อ_ผู้ใช้", "ชื่อผู้ใช้"},
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
		"用户_数据",      // Unicode test
		"тип_класса", // Cyrillic test
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			pascal := core.ToPascalCase(input)
			camel := core.ToCamelCase(input)

			// Pascal case should be camel case with first letter uppercase
			if len(camel) > 0 && len(pascal) > 0 {
				// Use rune-based comparison for Unicode correctness
				pascalRunes := []rune(pascal)
				camelRunes := []rune(camel)
				expectedPascal := string([]rune{pascalRunes[0]}) + string(camelRunes[1:])
				assert.Equal(t, expectedPascal, pascal,
					"ToPascalCase should be ToCamelCase with first rune capitalized")
			}
		})
	}
}
