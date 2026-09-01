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

func TestMaskTokensForYAMLParse(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "mapping scalar token masked",
			in:   `port: {{ .DB_PORT | 5432 }}`,
			want: `port: {{ .DB_PORT | 5432 }}`,
		},
		{
			name: "sequence item token masked",
			in:   "hosts:\n  - {{ .DB_HOST }}\n",
			want: "hosts:\n  - {{ .DB_HOST }}\n",
		},
		{
			name: "already quoted token masked",
			in:   `password: "{{ .DB_PASSWORD }}"`,
			want: `password: "{{ .DB_PASSWORD }}"`,
		},
		{
			name: "embedded token masked",
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
			want: `password: {{ .DB_PASSWORD }} # injected by CI`,
		},
		{
			name: "flow collection token masked",
			in:   `tags: [{{ .A }}, {{ .B }}]`,
			want: `tags: [{{ .A }}, {{ .B }}]`,
		},
		{
			name: "block scalar token masked without added quotes",
			in: "code: |\n" +
				"  export function transformEvent(event) {\n" +
				"    const cfg = {\n" +
				"      endpoint: {{ .ENDPOINT }}\n" +
				"    };\n" +
				"    return event;\n" +
				"  }\n",
			want: "code: |\n" +
				"  export function transformEvent(event) {\n" +
				"    const cfg = {\n" +
				"      endpoint: {{ .ENDPOINT }}\n" +
				"    };\n" +
				"    return event;\n" +
				"  }\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			masked, mask := MaskTokensForYAMLParse([]byte(tt.in))
			if tt.want == tt.in && tt.name != "comment token unchanged" && tt.name != "ui template unchanged" {
				assert.NotEqual(t, tt.in, string(masked))
			}
			assert.Equal(t, tt.want, mask.RestoreString(string(masked)))
			assert.False(t, mask.ContainsSentinel(mask.RestoreString(string(masked))))
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
