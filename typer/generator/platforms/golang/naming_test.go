package golang

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdentifierConstruction(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		pascal string
		field  string
	}{
		{"words split at spaces", "Some Track Event", "SomeTrackEvent", "SomeTrackEvent"},
		{"camelCase", "someString", "SomeString", "SomeString"},
		{"initialism after underscore", "user_id", "UserID", "UserID"},
		{"initialism after case boundary", "userId", "UserID", "UserID"},
		{"initialism after hyphen", "user-id", "UserID", "UserID"},
		{"leading initialism", "ip_address", "IPAddress", "IPAddress"},
		{"consecutive initialisms", "api_url", "APIURL", "APIURL"},
		{"SCREAMING_SNAKE", "PAYMENT_METHOD", "PaymentMethod", "PaymentMethod"},
		{"SCREAMING_SNAKE keeps initialisms", "USER_ID", "UserID", "UserID"},
		{"single upper-case word stays", "GET", "GET", "GET"},
		{"mixed case keeps its case", "smartTV", "SmartTV", "SmartTV"},
		{"acronym run", "XMLParser", "XMLParser", "XMLParser"},
		{"acronym run ends before a word", "HTMLPage", "HTMLPage", "HTMLPage"},
		{"initialisms match case-insensitively", "XMLHttp", "XMLHTTP", "XMLHTTP"},
		{"digit then upper-case boundary", "utf8Value", "UTF8Value", "UTF8Value"},
		{"digit inside a word", "user2Id", "User2ID", "User2ID"},
		{"dollar signs separate words", "$Variable$String", "VariableString", "VariableString"},
		{"dollar signs and punctuation", "$eventWithNameCamelCase$!", "EventWithNameCamelCase", "EventWithNameCamelCase"},
		{"quotes separate words", `Product "Premium" Clicked`, "ProductPremiumClicked", "ProductPremiumClicked"},
		{"dots separate words", "page.view", "PageView", "PageView"},
		{"keyword", "class", "Class", "Class"},
		{"predeclared identifier", "string", "String", "String"},
		{"leading digit gets X on fields", "1st_place", "1stPlace", "X1stPlace"},
		{"caseless letters get X on fields", "用户名", "用户名", "X用户名"},
		{"cyrillic", "типы_данных", "ТипыДанных", "ТипыДанных"},
		{"non-decimal digits separate words", "x²", "X", "X"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pascal, err := pascalCase(tt.input)
			require.NoError(t, err)
			field, err := fieldName(tt.input)
			require.NoError(t, err)

			assert.Equal(t, []string{tt.pascal, tt.field}, []string{pascal, field})
		})
	}
}

func TestIdentifierConstructionRejectsNamesWithoutWords(t *testing.T) {
	for _, input := range []string{"", "!!!", "🎯", "$ $"} {
		t.Run(input, func(t *testing.T) {
			_, err := pascalCase(input)
			assert.EqualError(t, err, `name "`+input+`" has no letters or digits to build a Go identifier from`)

			_, err = fieldName(input)
			assert.Error(t, err)
		})
	}
}
