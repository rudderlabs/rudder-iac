package varsubst

import (
	"strings"
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
			wantRefs: []Reference{{Name: "A", Text: "{{ .A }}"}, {Name: "B", Text: "{{.B}}"}, {Name: "C", Text: "{{ .C }}"}},
		},
		{
			name:     "default is recorded and dropped",
			input:    "port: {{ .PORT | 5432 }}",
			wantData: "port: __rudder_variable_reference_0__",
			wantRefs: []Reference{{Name: "PORT", HasDefault: true, Text: "{{ .PORT | 5432 }}"}},
		},
		{
			name:     "tokens inside a longer value",
			input:    "url: https://{{ .HOST }}/{{ .PATH }}",
			wantData: "url: https://__rudder_variable_reference_0__/__rudder_variable_reference_1__",
			wantRefs: []Reference{{Name: "HOST", Text: "{{ .HOST }}"}, {Name: "PATH", Text: "{{ .PATH }}"}},
		},
		{
			name:     "comment and malformed tokens are left as written",
			input:    "# {{ .IN_COMMENT }}\nbad: '{{ .9BAD }}'\nok: '{{ .OK }}'",
			wantData: "# {{ .IN_COMMENT }}\nbad: '{{ .9BAD }}'\nok: '__rudder_variable_reference_0__'",
			wantRefs: []Reference{{Name: "OK", Text: "{{ .OK }}"}},
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
	assert.Equal(t, Reference{Name: "KEY", Text: "{{ .KEY }}"}, ref)

	ref, ok = replaced.Reference(parsed["quoted"])
	assert.True(t, ok)
	assert.Equal(t, Reference{Name: "QUOTED", HasDefault: true, Text: "{{ .QUOTED | fallback }}"}, ref)
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
		{name: "placeholder", value: "__rudder_variable_reference_0__", want: Reference{Name: "A", Text: "{{ .A }}"}, wantOK: true, wantHoldsRef: true},
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
	assert.Equal(t, Reference{Name: "A", Text: "{{ .A }}"}, ref)
}

func TestReplaced_HoldsReference_NoReferences(t *testing.T) {
	replaced := ReplaceReferences([]byte("key: value"))
	assert.False(t, replaced.HoldsReference("__rudder_variable_reference_0__"))
}

// Restore turns text read from the placeholder copy back into the spec as
// written, whatever form each token took.
func TestReplaced_Restore(t *testing.T) {
	input := []byte("url: {{ .HOST }}/hook\nkey: '{{.KEY | fallback}}'\nplain: value\n")

	replaced := ReplaceReferences(input)
	var parsed map[string]string
	require.NoError(t, yaml.Unmarshal(replaced.Data, &parsed))

	assert.Equal(t, "{{ .HOST }}/hook", replaced.Restore(parsed["url"]))
	assert.Equal(t, "key: {{.KEY | fallback}}", replaced.Restore("key: "+parsed["key"]))
	assert.Equal(t, "plain: value", replaced.Restore("plain: "+parsed["plain"]))
	assert.Equal(t, string(input), replaced.Restore(string(replaced.Data)))
}

// A literal that looks like a placeholder is left alone, since the spec's
// placeholders got a longer prefix.
func TestReplaced_Restore_LiteralPlaceholder(t *testing.T) {
	replaced := ReplaceReferences([]byte("literal: __rudder_variable_reference_0__\nref: '{{ .A }}'\n"))

	assert.Equal(t, "literal: __rudder_variable_reference_0__", replaced.Restore("literal: __rudder_variable_reference_0__"))
}

// Every key the parser finds in the placeholder copy maps back to where the
// parser places it in the spec as written: after references longer or shorter
// than their placeholders, after a default holding a character wider than a
// byte, and below a reference that spans two lines.
func TestReplaced_SourcePosition(t *testing.T) {
	// The YAML parser counts CR and CRLF as line breaks too, so a file with
	// those endings must map the same way.
	for name, lineBreak := range map[string]string{"LF": "\n", "CRLF": "\r\n", "CR": "\r"} {
		t.Run(name, func(t *testing.T) {
			input := []byte(strings.ReplaceAll("a: '{{ .A }}'\n"+
				"b: {from: '{{ .HDR_NAME }}', to: '{{ .SHARED }}', at: 1}\n"+
				"c: {x: '{{ .X | \"é\" }}', y: 2}\n"+
				"d: '{{ .D |\n  \"multi\" }}'\n"+
				"e: {z: 3}\n", "\n", lineBreak))
			replaced := ReplaceReferences(input)
			require.Len(t, replaced.References, 5)

			written, placeholders := keyPositions(t, input), keyPositions(t, replaced.Data)
			require.Len(t, placeholders, len(written))
			for key, position := range placeholders {
				line, column := replaced.SourcePosition(position[0], position[1])
				assert.Equal(t, written[key], [2]int{line, column}, key)
			}

			// These keys sit elsewhere in the placeholder copy, so the loop
			// checks more than positions that stay put.
			var moved []string
			for key, position := range placeholders {
				if position != written[key] {
					moved = append(moved, key)
				}
			}
			assert.ElementsMatch(t, []string{"b/to", "b/at", "c/y", "e", "e/z"}, moved)
			assert.Equal(t, [2]int{6, 1}, written["e"], "the parser breaks lines at each ending")
		})
	}
}

// keyPositions returns the line and column of every mapping key in data, keyed
// by its path.
func keyPositions(t *testing.T, data []byte) map[string][2]int {
	t.Helper()

	var document yaml.Node
	require.NoError(t, yaml.Unmarshal(data, &document))

	positions := make(map[string][2]int)
	var walk func(node *yaml.Node, path string)
	walk = func(node *yaml.Node, path string) {
		if node.Kind != yaml.MappingNode {
			return
		}
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i]
			keyPath := path + key.Value
			positions[keyPath] = [2]int{key.Line, key.Column}
			walk(node.Content[i+1], keyPath+"/")
		}
	}
	walk(document.Content[0], "")
	return positions
}
