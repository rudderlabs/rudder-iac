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

func TestSingleQuoteTokens(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "double-quoted token single-quoted",
			in:   `accessKey: "{{ .ACCESS_KEY }}"`,
			want: `accessKey: '{{ .ACCESS_KEY }}'`,
		},
		{
			name: "token with default single-quoted",
			in:   `region: "{{ .REGION | us-east-1 }}"`,
			want: `region: '{{ .REGION | us-east-1 }}'`,
		},
		{
			// Not a variable reference, so it keeps its double quotes.
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
		{
			// The quote before the token is an escape inside a longer string,
			// so there is no whole scalar to rewrite.
			name: "token after an escaped quote keeps quotes",
			in:   `a: "x\"{{ .X }}"`,
			want: `a: "x\"{{ .X }}"`,
		},
		{
			name: "several scalars on one line",
			in:   `a: ["{{ .A }}", "{{ .B }}"]`,
			want: `a: ['{{ .A }}', '{{ .B }}']`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, string(SingleQuoteTokens([]byte(tt.in))))
		})
	}
}
