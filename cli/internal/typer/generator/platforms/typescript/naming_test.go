package typescript_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/platforms/typescript"
	"github.com/stretchr/testify/assert"
)

func TestFormatTypeName(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		input    string
		expected string
	}{
		// basic conversions
		{"snake_case name", "", "user_id", "UserId"},
		{"space separated name", "", "user signed up", "UserSignedUp"},
		{"kebab-case name", "", "email-address", "EmailAddress"},
		{"camelCase name", "", "firstName", "FirstName"},
		{"PascalCase name", "", "FirstName", "FirstName"},
		{"single word", "", "user", "User"},
		{"with numbers", "", "user123", "User123"},
		{"empty name no prefix", "", "", ""},
		{"whitespace only yields empty", "", "   ", ""},
		{"dot separated", "", "a.b.c", "ABC"},
		// prefix + name combinations
		{"identify prefix", "Identify", "Traits", "IdentifyTraits"},
		{"identify with event name", "Identify", "user signed up", "IdentifyUserSignedUp"},
		{"track prefix", "Track", "User Signed Up", "TrackUserSignedUp"},
		{"identify prefix empty name", "Identify", "", "Identify"},
		// special chars: only stripped from token edges, not from the middle
		{"leading dollar in name", "", "$variable", "Variable"},
		{"quotes around word", "", `"premium"`, "Premium"},
		// PascalCase output cannot collide with TS keywords (they're all-lowercase)
		{"PascalCase reserved word not escaped", "", "string", "String"},
		{"PascalCase 'class' not escaped", "", "class", "Class"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, typescript.FormatTypeName(tt.prefix, tt.input))
		})
	}
}

func TestFormatPropertyName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// common separator styles
		{"snake_case", "device_type", "deviceType"},
		{"kebab-case", "email-address", "emailAddress"},
		{"space separated", "user signed up", "userSignedUp"},
		{"dot separated", "a.b.c", "aBC"},
		// case handling
		{"camelCase unchanged", "firstName", "firstName"},
		{"PascalCase lowercased first token", "FirstName", "firstName"},
		{"ALL_CAPS", "ALL_CAPS", "allCaps"},
		{"single word lowercase", "user", "user"},
		// numbers
		{"number at end", "user123", "user123"},
		{"number then uppercase letter stays one token", "user123Id", "user123id"},
		// empty / whitespace
		{"empty string", "", ""},
		{"whitespace only returns original", "   ", "   "},
		// special characters: trimmed from token edges
		{"leading dollar trimmed", "$variable", "variable"},
		{"trailing exclamation trimmed", "event!", "event"},
		{"leading and trailing non-ident trimmed", "!event!", "event"},
		{"quotes stripped from edges", `"premium"`, "premium"},
		// mid-token special chars are NOT stripped
		{"percent in middle stays", "user%status", "user%status"},
		{"at in middle stays", "user@email", "user@email"},
		// complex multi-separator
		{"mixed separators", "user_email-address.type", "userEmailAddressType"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, typescript.FormatPropertyName(tt.input))
		})
	}
}

func TestFormatPropertyName_ReservedWords(t *testing.T) {
	// Reserved words and contextual TS keywords are renamed with an underscore
	// suffix so the property name does not collide with built-in identifiers
	// or shadow built-in types under strict mode.
	reserved := []string{
		// JS reserved
		"break", "case", "catch", "class", "const", "continue", "default", "delete",
		"do", "else", "enum", "export", "extends", "finally", "for", "function",
		"if", "import", "in", "instanceof", "new", "null", "return", "super",
		"switch", "this", "throw", "try", "typeof", "var", "void", "while", "with",
		// strict-mode reserved
		"as", "implements", "interface", "let", "package", "private", "protected",
		"public", "static", "yield",
		// contextual / TS-specific
		"any", "async", "await", "boolean", "constructor", "declare", "from", "get",
		"is", "keyof", "module", "namespace", "never", "number", "of", "readonly",
		"require", "set", "string", "symbol", "type", "undefined", "unknown",
	}

	for _, word := range reserved {
		t.Run(word, func(t *testing.T) {
			assert.Equal(t, word+"_", typescript.FormatPropertyName(word))
		})
	}
}

func TestFormatMethodName(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		input    string
		expected string
	}{
		{"identify no prefix", "", "identify", "identify"},
		{"track event", "track", "user signed up", "trackUserSignedUp"},
		{"snake_case name", "track", "user_id", "trackUserId"},
		{"kebab-case name", "track", "email-address", "trackEmailAddress"},
		{"empty prefix camelCase", "", "firstName", "firstName"},
		{"empty prefix PascalCase", "", "FirstName", "firstName"},
		{"leading dollar trimmed", "track", "$variable", "trackVariable"},
		{"empty prefix and name", "", "", ""},
		// reserved word with prefix is no longer reserved after concatenation
		{"reserved word with prefix", "track", "function", "trackFunction"},
		// reserved word standalone is suffixed
		{"reserved word no prefix", "", "function", "function_"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, typescript.FormatMethodName(tt.prefix, tt.input))
		})
	}
}
