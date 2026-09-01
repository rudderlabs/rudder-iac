package varsubst

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractVariableNames(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "no tokens",
			in:   "accessKey: plain-value",
			want: []string{},
		},
		{
			name: "single token",
			in:   `accessKey: "{{ .ACCESS_KEY }}"`,
			want: []string{"ACCESS_KEY"},
		},
		{
			name: "multiple tokens in order",
			in:   `a: "{{ .SECOND }}" preceded by {{ .FIRST }} no wait`,
			want: []string{"SECOND", "FIRST"},
		},
		{
			name: "token with default",
			in:   `a: "{{ .REGION | us-east-1 }}"`,
			want: []string{"REGION"},
		},
		{
			name: "malformed tokens are skipped",
			in:   `a: "{{ NO_DOT }}" b: "{{ .9starts-with-digit }}"`,
			want: []string{},
		},
		{
			name: "repeated token reported each time",
			in:   `a: "{{ .KEY }}" b: "{{ .KEY }}"`,
			want: []string{"KEY", "KEY"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ExtractVariableNames([]byte(tt.in)))
		})
	}
}

func TestQuoteTokensForYAMLParse(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "mapping scalar token quoted",
			in:   `port: {{ .DB_PORT | 5432 }}`,
			want: `port: "{{ .DB_PORT | 5432 }}"`,
		},
		{
			name: "sequence item token quoted",
			in:   "hosts:\n  - {{ .DB_HOST }}\n",
			want: "hosts:\n  - \"{{ .DB_HOST }}\"\n",
		},
		{
			name: "already quoted token unchanged",
			in:   `password: "{{ .DB_PASSWORD }}"`,
			want: `password: "{{ .DB_PASSWORD }}"`,
		},
		{
			name: "embedded token unchanged",
			in:   `url: "https://{{ .DB_HOST }}/orders"`,
			want: `url: "https://{{ .DB_HOST }}/orders"`,
		},
		{
			name: "comment token unchanged",
			in:   `password: plain # {{ .DB_PASSWORD }}`,
			want: `password: plain # {{ .DB_PASSWORD }}`,
		},
		{
			name: "ui template unchanged",
			in:   `password: {{ config.password || fallback }}`,
			want: `password: {{ config.password || fallback }}`,
		},
		{
			name: "trailing comment preserved",
			in:   `password: {{ .DB_PASSWORD }} # injected by CI`,
			want: `password: "{{ .DB_PASSWORD }}" # injected by CI`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, string(QuoteTokensForYAMLParse([]byte(tt.in))))
		})
	}
}

func TestUnquoteTokens(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "quoted token unquoted",
			in:   `accessKey: "{{ .ACCESS_KEY }}"`,
			want: `accessKey: {{ .ACCESS_KEY }}`,
		},
		{
			name: "token with default unquoted",
			in:   `region: "{{ .REGION | us-east-1 }}"`,
			want: `region: {{ .REGION | us-east-1 }}`,
		},
		{
			// Not a variable reference, so it keeps its quotes — unquoting a
			// non-variable token would emit a scalar starting with '{', which
			// YAML reads as a flow mapping.
			name: "non variable token keeps quotes",
			in:   `a: "{{ NO_DOT }}"`,
			want: `a: "{{ NO_DOT }}"`,
		},
		{
			name: "ui template keeps quotes",
			in:   `a: "{{ config.bucketName || my-bucket }}"`,
			want: `a: "{{ config.bucketName || my-bucket }}"`,
		},
		{
			name: "token embedded in longer string keeps quotes",
			in:   `a: "prefix {{ .KEY }} suffix"`,
			want: `a: "prefix {{ .KEY }} suffix"`,
		},
		{
			name: "unquoted token untouched",
			in:   `accessKey: {{ .ACCESS_KEY }}`,
			want: `accessKey: {{ .ACCESS_KEY }}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, string(UnquoteTokens([]byte(tt.in))))
		})
	}
}
