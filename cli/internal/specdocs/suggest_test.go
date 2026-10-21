package specdocs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSuggest(t *testing.T) {
	t.Parallel()

	candidates := []candidate{
		{value: "am", aliases: []string{"Amplitude"}},
		{value: "bq", aliases: []string{"BigQuery"}},
		{value: "bqstream", aliases: []string{"BigQuery Stream"}},
		{value: "ga4", aliases: []string{"Google Analytics 4"}},
		{value: "webhook", aliases: []string{"Webhook"}},
	}

	cases := []struct {
		input string
		want  []string
	}{
		{input: "Amplitude", want: []string{"am"}},
		{input: "big query", want: []string{"bq", "bqstream"}},
		{input: "webhok", want: []string{"webhook"}},
		{input: "google analytics", want: []string{"ga4"}},
		{input: "ga", want: []string{"ga4"}},
		{input: "xy", want: []string{}},
		{input: "--", want: []string{}},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, suggest(tc.input, candidates), tc.input)
	}
}

func TestSuggestSourcesMapsTokens(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"javascript"}, suggestSources("web"))
	assert.Equal(t, []string{"javascript"}, suggestSources("javscript"))
	assert.Equal(t, []string{"dotnet", "go", "java", "node", "php", "python", "ruby", "rust"}, suggestSources("cloud"))
}

func TestSuggestID(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "shop-web", suggestID("Shop.Web"))
	assert.Equal(t, "shop", suggestID("9-shop"))
	assert.Equal(t, "", suggestID("123"))
}
