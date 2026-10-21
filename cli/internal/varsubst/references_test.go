package varsubst

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestReplaceReferences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantData string
		wantRefs []Reference
	}{
		{
			name:     "no tokens",
			input:    "key: value",
			wantData: "key: value",
		},
		{
			name:     "quoted and unquoted tokens",
			input:    "a: '{{ .A }}'\nb: \"{{.B}}\"\nc: {{ .C }}",
			wantData: "a: '__rudder_variable_reference_0__'\nb: \"__rudder_variable_reference_1__\"\nc: __rudder_variable_reference_2__",
			wantRefs: []Reference{{Name: "A"}, {Name: "B"}, {Name: "C"}},
		},
		{
			name:     "default is recorded and dropped",
			input:    "port: {{ .PORT | 5432 }}",
			wantData: "port: __rudder_variable_reference_0__",
			wantRefs: []Reference{{Name: "PORT", HasDefault: true}},
		},
		{
			name:     "tokens inside a longer value",
			input:    "url: https://{{ .HOST }}/{{ .PATH }}",
			wantData: "url: https://__rudder_variable_reference_0__/__rudder_variable_reference_1__",
			wantRefs: []Reference{{Name: "HOST"}, {Name: "PATH"}},
		},
		{
			name:     "comment and malformed tokens are left as written",
			input:    "# {{ .IN_COMMENT }}\nbad: '{{ .9BAD }}'\nok: '{{ .OK }}'",
			wantData: "# {{ .IN_COMMENT }}\nbad: '{{ .9BAD }}'\nok: '__rudder_variable_reference_0__'",
			wantRefs: []Reference{{Name: "OK"}},
		},
		{
			name:     "UI templates without a dot are not references",
			input:    "to: '{{ message.token || \"x\" }}'",
			wantData: "to: '{{ message.token || \"x\" }}'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, refs := ReplaceReferences([]byte(tt.input))
			assert.Equal(t, tt.wantData, string(data))
			assert.Equal(t, tt.wantRefs, refs)
		})
	}
}

// An unquoted token followed by more text is not YAML until substitution runs.
// The placeholders make it parse, and each value maps back to its reference.
func TestReplaceReferences_ParsesAsWritten(t *testing.T) {
	input := []byte("url: {{ .HOST }}/hook\nkey: {{ .KEY }}\nquoted: '{{ .QUOTED | fallback }}'\n")

	var unparsed map[string]any
	require.Error(t, yaml.Unmarshal(input, &unparsed))

	data, refs := ReplaceReferences(input)
	var parsed map[string]string
	require.NoError(t, yaml.Unmarshal(data, &parsed))

	_, ok := ReplacedReference(parsed["url"], refs)
	assert.False(t, ok, "a placeholder with text around it is not a whole reference")

	ref, ok := ReplacedReference(parsed["key"], refs)
	assert.True(t, ok)
	assert.Equal(t, Reference{Name: "KEY"}, ref)

	ref, ok = ReplacedReference(parsed["quoted"], refs)
	assert.True(t, ok)
	assert.Equal(t, Reference{Name: "QUOTED", HasDefault: true}, ref)
}

func TestReplacedReference(t *testing.T) {
	refs := []Reference{{Name: "A"}}

	tests := []struct {
		name   string
		value  string
		want   Reference
		wantOK bool
	}{
		{name: "placeholder", value: "__rudder_variable_reference_0__", want: Reference{Name: "A"}, wantOK: true},
		{name: "index out of range", value: "__rudder_variable_reference_1__"},
		{name: "surrounding text", value: " __rudder_variable_reference_0__"},
		{name: "literal", value: "sk-not-a-reference"},
		{name: "unreplaced token", value: "{{ .A }}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ReplacedReference(tt.value, refs)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}
