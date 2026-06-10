package secret

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func enableVarSubstitution(t *testing.T) {
	t.Helper()
	prevExp, prevFlag := viper.Get("experimental"), viper.Get("flags.enableVarSubstitution")
	viper.Set("experimental", true)
	viper.Set("flags.enableVarSubstitution", true)
	t.Cleanup(func() {
		viper.Set("experimental", prevExp)
		viper.Set("flags.enableVarSubstitution", prevFlag)
	})
}

// With scaffolding enabled, the marshals emit the variable reference — that is
// the whole point: the token must survive into exported YAML where a plain
// secret would have been redacted into a useless literal.
func TestImportableSecret_MarshalsVariableReference(t *testing.T) {
	enableVarSubstitution(t)

	s := NewImportable(NewUnknown(), "BOOK_THE_HOBBIT_ACCESS_KEY")

	jsonBytes, err := json.Marshal(s)
	require.NoError(t, err)
	assert.Equal(t, `"{{ .BOOK_THE_HOBBIT_ACCESS_KEY }}"`, string(jsonBytes))

	yamlVal, err := s.MarshalYAML()
	require.NoError(t, err)
	assert.Equal(t, "{{ .BOOK_THE_HOBBIT_ACCESS_KEY }}", yamlVal)
}

// Even when a known value is wrapped, only the reference is emitted — the real
// value must never reach an export surface.
func TestImportableSecret_NeverLeaksValue(t *testing.T) {
	enableVarSubstitution(t)

	s := NewImportable(New("hunter2-but-long"), "ACCESS_KEY")

	jsonBytes, err := json.Marshal(s)
	require.NoError(t, err)
	assert.NotContains(t, string(jsonBytes), "hunter2")

	// The embedded String keeps masking every formatting surface. (The String()
	// method is shadowed by the embedded field's name, but fmt routes through
	// the promoted Format/GoString methods, so redaction holds.)
	assert.NotContains(t, fmt.Sprintf("%v %s %q %#v", s, s, s, s), "hunter2")
	assert.NotContains(t, fmt.Sprint(s), "hunter2")
}

// With the gate off the name is dropped at construction, so the secret
// serializes as a masked literal — the pre-scaffolding behaviour.
func TestImportableSecret_GateOff(t *testing.T) {
	s := NewImportable(NewUnknown(), "ACCESS_KEY")
	assert.Equal(t, ImportableSecret{String: NewUnknown()}, s)

	jsonBytes, err := json.Marshal(s)
	require.NoError(t, err)
	assert.Equal(t, `"(unknown)"`, string(jsonBytes))
}

func TestImportableSecret_EmptyNameSerializesLikeString(t *testing.T) {
	enableVarSubstitution(t)

	jsonBytes, err := json.Marshal(ImportableSecret{String: New("sk_live_abcd1234")})
	require.NoError(t, err)
	assert.Equal(t, `"****1234"`, string(jsonBytes))
}

func TestNewImportable_NormalizesVarName(t *testing.T) {
	enableVarSubstitution(t)

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"kebab external id", "BOOK_the-hobbit_ACCESS_KEY", "BOOK_THE_HOBBIT_ACCESS_KEY"},
		{"already valid", "ACCESS_KEY", "ACCESS_KEY"},
		{"special chars fold", "retl/src.main password", "RETL_SRC_MAIN_PASSWORD"},
		{"leading digit prefixed", "1984-access-key", "_1984_ACCESS_KEY"},
		{"trims stray separators", "-access-key-", "ACCESS_KEY"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NewImportable(String{}, tt.in).VarName)
		})
	}
}
