package kotlin_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/platforms/kotlin"
	"github.com/stretchr/testify/assert"
)

func TestFormatClassName(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		input    string
		expected string
	}{
		{"snake_case", "", "user_id", "UserId"},
		{"kebab-case", "", "email-address", "EmailAddress"},
		{"space separated", "", "first name", "FirstName"},
		{"camelCase", "", "firstName", "FirstName"},
		{"PascalCase", "", "FirstName", "FirstName"},
		{"single word", "", "user", "User"},
		{"with numbers", "", "user123", "User123"},
		{"number at end", "", "user123Id", "User123Id"},
		{"leading number", "", "123user", "_123user"},
		{"reserved keyword", "", "class", "Class"},
		{"reserved keyword uppercase", "", "Class", "Class"},
		{"reserved keyword mixed", "", "CLASS", "Class"},
		{"empty string", "", "", ""},
		{"whitespace only", "", "   ", ""},
		{"complex name", "", "user_email-address.type", "UserEmailAddressType"},
		// Tests with prefix
		{"with prefix track", "track", "User Signed Up", "TrackUserSignedUp"},
		{"with prefix get", "get", "user_info", "GetUserInfo"},
		{"with prefix set", "set", "email-address", "SetEmailAddress"},
		{"prefix with empty name", "track", "", "Track"},
		{"empty prefix", "", "testMethod", "TestMethod"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := kotlin.FormatClassName(tt.prefix, tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatPropertyName_ReservedKeywords(t *testing.T) {
	// Test various reserved keywords
	reservedKeywords := []string{}
	for keyword := range kotlin.KotlinHardKeywords {
		reservedKeywords = append(reservedKeywords, keyword)
	}

	for _, keyword := range reservedKeywords {
		t.Run(keyword, func(t *testing.T) {
			result := kotlin.FormatPropertyName(keyword)
			assert.Equal(t, byte('_'), result[0], "Reserved keyword should be prefixed with underscore")
			assert.NotEqual(t, keyword, result, "Result should not be the same as reserved keyword")
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
		{"snake_case", "", "user_id", "userId"},
		{"kebab-case", "", "email-address", "emailAddress"},
		{"space separated", "", "first name", "firstName"},
		{"camelCase", "", "firstName", "firstName"},
		{"PascalCase", "", "FirstName", "firstName"},
		{"single word", "", "user", "user"},
		{"with numbers", "", "user123", "user123"},
		{"number at end", "", "user123Id", "user123Id"},
		{"leading number", "", "123user", "_123user"},
		{"reserved keyword", "", "class", "_class"},
		{"reserved keyword uppercase", "", "Class", "_class"},
		{"reserved keyword mixed", "", "CLASS", "_class"},
		{"empty string", "", "", ""},
		{"whitespace only", "", "   ", ""},
		{"complex name", "", "user_email-address.type", "userEmailAddressType"},
		// Tests with prefix
		{"with prefix track", "track", "User Signed Up", "trackUserSignedUp"},
		{"with prefix get", "get", "user_info", "getUserInfo"},
		{"with prefix set", "set", "email-address", "setEmailAddress"},
		{"prefix with empty name", "track", "", "track"},
		{"empty prefix", "", "testMethod", "testMethod"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := kotlin.FormatMethodName(tt.prefix, tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatEnumValue(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"lowercase string", "active", "ACTIVE"},
		{"uppercase string", "INACTIVE", "INACTIVE"},
		{"mixed case string", "PendingApproval", "PENDING_APPROVAL"},
		{"snake_case", "user_status", "USER_STATUS"},
		{"kebab-case", "email-verified", "EMAIL_VERIFIED"},
		{"space separated", "first name", "FIRST_NAME"},
		{"camelCase", "firstName", "FIRST_NAME"},
		{"PascalCase", "FirstName", "FIRST_NAME"},
		{"with numbers", "status123", "STATUS123"},
		{"number at end", "level1", "LEVEL1"},
		{"leading number", "123user", "_123USER"},
		{"special characters", "user@status", "USER_STATUS"},
		{"multiple special chars", "user-email.address", "USER_EMAIL_ADDRESS"},
		{"dots and dashes", "test.value-here", "TEST_VALUE_HERE"},
		{"reserved keyword", "class", "CLASS"},
		{"reserved keyword type", "int", "INT"},
		{"reserved keyword when", "when", "WHEN"},
		{"empty string", "", ""},
		{"whitespace only", "   ", ""},
		{"integer value", 42, "_42"},
		{"boolean value", true, "TRUE"},
		{"complex mixed", "User-Status_123.Active!", "USER_STATUS_123_ACTIVE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := kotlin.FormatEnumValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
