package definitions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsEmptyConfigValue(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value any
		empty bool
	}{
		{"nil", nil, true},
		{"empty string", "", true},
		{"non-empty string", "x", false},
		{"false bool is meaningful", false, false},
		{"true bool", true, false},
		{"empty slice", []any{}, true},
		{"slice of empty strings", []any{"", ""}, true},
		{"slice with a value", []any{"", "x"}, false},
		{"empty map", map[string]any{}, true},
		{"map of empty values", map[string]any{"a": "", "b": []any{""}}, true},
		{"map with a value", map[string]any{"a": "x"}, false},
		{"event filter shape all empty", map[string]any{"whitelist": []any{""}, "blacklist": []any{""}}, true},
		{"number kept", float64(0), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.empty, isEmptyConfigValue(tc.value))
		})
	}
}

func TestPruneEmptyOptional_EmptyConfig(t *testing.T) {
	t.Parallel()

	// A definition with no required fields: every empty key is pruned, non-empty
	// kept. Uses the exported entrypoint via a minimal registered definition is
	// covered in the http package; here we only guard the nil/empty fast path.
	d := &RegisteredDefinition{}
	assert.Empty(t, d.PruneEmptyOptional(map[string]any{}))
	assert.Nil(t, d.PruneEmptyOptional(nil))
}
