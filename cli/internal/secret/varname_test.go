package secret

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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

// With a variable name attached, the marshals emit the reference — that is the
// whole point: the token must survive into exported YAML where a plain secret
// would have been redacted into a useless literal.
func TestWithVariableName_MarshalsVariableReference(t *testing.T) {
	enableVarSubstitution(t)

	s := NewUnknown(WithVariableName("BOOK_THE_HOBBIT_ACCESS_KEY"))

	jsonBytes, err := json.Marshal(s)
	require.NoError(t, err)
	assert.Equal(t, `"{{ .BOOK_THE_HOBBIT_ACCESS_KEY }}"`, string(jsonBytes))

	yamlVal, err := s.MarshalYAML()
	require.NoError(t, err)
	assert.Equal(t, "{{ .BOOK_THE_HOBBIT_ACCESS_KEY }}", yamlVal)
}

// Even when a known value carries a name, only the reference is emitted by the
// marshals and every formatting surface keeps masking — the real value must
// never escape.
func TestWithVariableName_NeverLeaksValue(t *testing.T) {
	enableVarSubstitution(t)

	s := New("hunter2-but-long", WithVariableName("ACCESS_KEY"))

	jsonBytes, err := json.Marshal(s)
	require.NoError(t, err)
	assert.NotContains(t, string(jsonBytes), "hunter2")

	assert.NotContains(t, fmt.Sprintf("%v %s %q %#v", s, s, s, s), "hunter2")
	assert.Equal(t, "****long", s.String())
}

// With the gate off the option is a no-op, so the secret serializes as a
// masked literal — the pre-scaffolding behaviour.
func TestWithVariableName_GateOff(t *testing.T) {
	s := NewUnknown(WithVariableName("ACCESS_KEY"))
	assert.Equal(t, NewUnknown(), s)

	jsonBytes, err := json.Marshal(s)
	require.NoError(t, err)
	assert.Equal(t, `"(unknown)"`, string(jsonBytes))
}

// Loading a spec value over a named secret produces a plain known secret; the
// variable name only ever exists on the export path.
func TestWithVariableName_UnmarshalResetsName(t *testing.T) {
	enableVarSubstitution(t)

	s := NewUnknown(WithVariableName("ACCESS_KEY"))
	require.NoError(t, yaml.Unmarshal([]byte(`"real-value"`), &s))
	assert.Equal(t, New("real-value"), s)

	s = NewUnknown(WithVariableName("ACCESS_KEY"))
	require.NoError(t, json.Unmarshal([]byte(`"real-value"`), &s))
	assert.Equal(t, New("real-value"), s)
}

func TestWithVariableName_Normalizes(t *testing.T) {
	enableVarSubstitution(t)

	tests := []struct {
		name string
		in   string
		want String
	}{
		{"kebab external id", "BOOK_the-hobbit_ACCESS_KEY", String{varName: "BOOK_THE_HOBBIT_ACCESS_KEY"}},
		{"already valid", "ACCESS_KEY", String{varName: "ACCESS_KEY"}},
		{"special chars fold", "retl/src.main password", String{varName: "RETL_SRC_MAIN_PASSWORD"}},
		{"leading digit prefixed", "1984-access-key", String{varName: "_1984_ACCESS_KEY"}},
		{"trims stray separators", "-access-key-", String{varName: "ACCESS_KEY"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, New("", WithVariableName(tt.in)))
		})
	}
}
