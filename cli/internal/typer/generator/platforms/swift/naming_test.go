package swift_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/platforms/swift"
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
		{"track prefix", "Track", "User Signed Up", "TrackUserSignedUp"},
		{"property prefix snake_case", "Property", "device_type", "PropertyDeviceType"},
		{"identify prefix empty name", "Identify", "", "Identify"},
		{"case prefix", "Case", "search page", "CaseSearchPage"},
		{"group prefix empty", "Group", "", "Group"},
		// no prefix
		{"no prefix snake_case", "", "some_name", "SomeName"},
		{"no prefix complex", "", "user_email-address.type", "UserEmailAddressType"},
		// special chars: only stripped from token edges, not from the middle
		{"leading dollar in name", "", "$variable", "Variable"},
		{"quotes around word", "", `"premium"`, "Premium"},
		// @ in the middle stays — tokenizer only trims edge non-letter/digit chars
		{"at in middle passes through", "", "user@event", "User@event"},
		// PascalCase reserved words must be backtick-escaped
		{"reserved word Any", "", "any", "`Any`"},
		{"reserved word Self", "", "self", "`Self`"},
		{"reserved word Type", "", "type", "`Type`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, swift.FormatTypeName(tt.prefix, tt.input))
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
		// camelCase tokenizer does not insert space before digit→letter boundaries
		// so "user123Id" is one token, entirely lowercased → "user123id"
		{"number then uppercase letter stays one token", "user123Id", "user123id"},
		// empty / whitespace
		{"empty string", "", ""},
		// whitespace-only has no tokens → FormatPropertyName returns the original string
		{"whitespace only returns original", "   ", "   "},
		// special characters: only trimmed from the START/END of a space-split token
		{"leading dollar trimmed", "$variable", "variable"},
		{"trailing exclamation trimmed", "event!", "event"},
		{"leading and trailing non-ident trimmed", "!event!", "event"},
		{"quotes stripped from edges", `"premium"`, "premium"},
		// mid-token special chars are NOT stripped — they remain in the output
		{"percent in middle stays", "user%status", "user%status"},
		{"dollar in middle stays", "price$amount", "price$amount"},
		{"at in middle stays", "user@email", "user@email"},
		{"slash in middle stays", "path/to/file", "path/to/file"},
		{"colon in middle stays", "key:value", "key:value"},
		// complex multi-separator (only _ - . and spaces split tokens)
		{"mixed separators", "user_email-address.type", "userEmailAddressType"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, swift.FormatPropertyName(tt.input))
		})
	}
}

func TestFormatPropertyName_ReservedWords(t *testing.T) {
	// All-lowercase reserved words are escaped with backticks after tokenization.
	// Source: https://docs.swift.org/swift-book/documentation/the-swift-programming-language/lexicalstructure/
	lowerCaseReserved := []string{
		// declarations
		"associatedtype", "class", "deinit", "enum", "extension", "fileprivate",
		"func", "import", "init", "inout", "internal", "let", "open", "operator",
		"private", "protocol", "public", "rethrows", "static", "struct", "subscript",
		"typealias", "var",
		// statements
		"break", "case", "continue", "default", "defer", "do", "else", "fallthrough",
		"for", "guard", "if", "in", "repeat", "return", "switch", "where", "while",
		// expressions and types (lowercase only — PascalCase ones tokenize to lowercase
		// and the map key is case-sensitive, so only these exact lowercase forms match)
		"as", "catch", "false", "is", "nil", "self", "super", "throw", "throws", "true", "try",
	}

	for _, word := range lowerCaseReserved {
		t.Run(word, func(t *testing.T) {
			assert.Equal(t, "`"+word+"`", swift.FormatPropertyName(word))
		})
	}
}

func TestFormatPropertyName_PascalCaseReservedWords(t *testing.T) {
	// PascalCase reserved words (Any, Self, Type) tokenize to lowercase.
	// The reserved-word map is case-sensitive, so only the exact stored casing triggers escaping.
	// "Self" → tokenizes to "self" → "self" IS in the map → escaped as `self`
	// "Any"  → tokenizes to "any"  → "any"  is NOT in the map (stored as "Any") → not escaped
	// "Type" → tokenizes to "type" → "type" is NOT in the map (stored as "Type") → not escaped
	tests := []struct {
		input    string
		expected string
	}{
		{"Self", "`self`"}, // "Self" → "self" which is in the map
		{"Any", "any"},     // "Any"  → "any"  which is NOT in the map
		{"Type", "type"},   // "Type" → "type" which is NOT in the map
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, swift.FormatPropertyName(tt.input))
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
		// event method names
		{"track event", "track", "user signed up", "trackUserSignedUp"},
		{"identify no prefix", "", "identify", "identify"},
		{"screen event", "screen", "home view", "screenHomeView"},
		{"group no prefix", "", "group", "group"},
		// separator styles in name
		{"snake_case name", "track", "user_id", "trackUserId"},
		{"kebab-case name", "track", "email-address", "trackEmailAddress"},
		// empty prefix
		{"empty prefix camelCase", "", "firstName", "firstName"},
		{"empty prefix PascalCase", "", "FirstName", "firstName"},
		// leading special chars on a token are trimmed
		{"leading dollar in name trimmed", "track", "$variable", "trackVariable"},
		// dollar in the middle of a token stays (same rule as FormatPropertyName)
		{"dollar mid-token stays", "track", "variable$string", "trackVariable$string"},
		// empty / whitespace
		{"empty prefix and name", "", "", ""},
		{"whitespace only name returns original", "", "   ", "   "},
		// reserved words
		{"reserved word no prefix", "", "var", "`var`"},
		{"reserved word with prefix", "track", "func", "trackFunc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, swift.FormatMethodName(tt.prefix, tt.input))
		})
	}
}

func TestFormatEnumCaseName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// normal identifiers
		{"uppercase", "GET", "get"},
		{"snake_case", "some_value", "someValue"},
		{"mixed case", "PendingApproval", "pendingApproval"},
		{"kebab-case", "active-status", "activeStatus"},
		// digit-leading → "n" prefix
		{"integer string", "200", "n200"},
		{"float string", "1.5", "n15"},
		{"digit then alpha", "42abc", "n42abc"},
		// emoji / symbol-only → unicode escape
		{"single emoji", "🎯", "u1F3AF"},
		{"multi-emoji", "🎯🚀", "u1F3AF1F680"},
		// empty string: no tokens → unicodeEscape("") → "u" (prefix with no runes)
		{"empty string yields unicode prefix", "", "u"},
		// reserved word → backtick-escaped
		{"reserved default", "default", "`default`"},
		{"reserved case", "case", "`case`"},
		{"reserved in", "in", "`in`"},
		{"reserved true", "true", "`true`"},
		{"reserved nil", "nil", "`nil`"},
		// leading special char trimmed
		{"leading dollar trimmed", "$price", "price"},
		{"leading at trimmed", "@user", "user"},
		// trailing digit after trimming → "n" prefix
		{"leading special before digit", "!200", "n200"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, swift.FormatEnumCaseName(tt.input))
		})
	}
}
