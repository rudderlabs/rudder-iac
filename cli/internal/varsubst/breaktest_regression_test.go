package varsubst

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// These values come from a var file at apply time. A successful YAML parse is
// insufficient: a changed value can silently corrupt a credential or add keys.
func TestSubstituteBytesPreservesScalarValueAndDocumentShape(t *testing.T) {
	tests := []struct {
		name  string
		slot  string
		value string
	}{
		{name: "newline cannot inject a key", slot: "key: {{ .VALUE }}\n", value: "first\ninjected: true"},
		{name: "hash cannot truncate an unquoted value", slot: "key: {{ .VALUE }}\n", value: "abc # suffix"},
		{name: "tab is literal in a double quoted secret", slot: "key: \"{{ .VALUE }}\"\n", value: "abc\\tdef"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, errs := NewSubstitutor(mapResolver{"VALUE": tt.value}).SubstituteBytes([]byte(tt.slot))
			require.Empty(t, errs)
			var decoded map[string]any
			require.NoError(t, yaml.Unmarshal(got, &decoded))
			assert.Equal(t, map[string]any{"key": tt.value}, decoded)
		})
	}
}
