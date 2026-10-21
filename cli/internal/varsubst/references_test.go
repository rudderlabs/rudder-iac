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
			replaced := ReplaceReferences([]byte(tt.input))
			assert.Equal(t, tt.wantData, string(replaced.Data))
			assert.Equal(t, tt.wantRefs, replaced.References)
		})
	}
}

// An unquoted token followed by more text is not YAML until substitution runs.
// The placeholders make it parse, and each value maps back to its reference.
func TestReplaceReferences_ParsesAsWritten(t *testing.T) {
	input := []byte("url: {{ .HOST }}/hook\nkey: {{ .KEY }}\nquoted: '{{ .QUOTED | fallback }}'\n")

	var unparsed map[string]any
	require.Error(t, yaml.Unmarshal(input, &unparsed))

	replaced := ReplaceReferences(input)
	var parsed map[string]string
	require.NoError(t, yaml.Unmarshal(replaced.Data, &parsed))

	_, ok := replaced.Reference(parsed["url"])
	assert.False(t, ok, "a placeholder with text around it is not a whole reference")
	assert.True(t, replaced.HoldsReference(parsed["url"]))

	ref, ok := replaced.Reference(parsed["key"])
	assert.True(t, ok)
	assert.Equal(t, Reference{Name: "KEY"}, ref)

	ref, ok = replaced.Reference(parsed["quoted"])
	assert.True(t, ok)
	assert.Equal(t, Reference{Name: "QUOTED", HasDefault: true}, ref)
}

func TestReplaced_Reference(t *testing.T) {
	replaced := ReplaceReferences([]byte("a: '{{ .A }}'"))

	tests := []struct {
		name         string
		value        string
		want         Reference
		wantOK       bool
		wantHoldsRef bool
	}{
		{name: "placeholder", value: "__rudder_variable_reference_0__", want: Reference{Name: "A"}, wantOK: true, wantHoldsRef: true},
		{name: "index out of range", value: "__rudder_variable_reference_1__", wantHoldsRef: true},
		{name: "signed index", value: "__rudder_variable_reference_+0__", wantHoldsRef: true},
		{name: "surrounding text", value: " __rudder_variable_reference_0__", wantHoldsRef: true},
		{name: "literal", value: "sk-not-a-reference"},
		{name: "unreplaced token", value: "{{ .A }}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := replaced.Reference(tt.value)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantHoldsRef, replaced.HoldsReference(tt.value))
		})
	}
}

// A spec that already holds the placeholder text gets longer placeholders, so
// its literal is never read as a reference.
func TestReplaceReferences_LiteralPlaceholder(t *testing.T) {
	input := []byte("literal: __rudder_variable_reference_0__\nref: '{{ .A }}'\n")

	replaced := ReplaceReferences(input)
	var parsed map[string]string
	require.NoError(t, yaml.Unmarshal(replaced.Data, &parsed))

	assert.Equal(t, "__rudder_variable_reference_0__", parsed["literal"])
	_, ok := replaced.Reference(parsed["literal"])
	assert.False(t, ok)
	assert.False(t, replaced.HoldsReference(parsed["literal"]))

	ref, ok := replaced.Reference(parsed["ref"])
	assert.True(t, ok)
	assert.Equal(t, Reference{Name: "A"}, ref)
}

func TestReplaced_HoldsReference_NoReferences(t *testing.T) {
	replaced := ReplaceReferences([]byte("key: value"))
	assert.False(t, replaced.HoldsReference("__rudder_variable_reference_0__"))
}
