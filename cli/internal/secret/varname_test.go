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

// The name is used verbatim — choosing one that satisfies the substitutor's
// variable grammar is the provider's responsibility.
func TestWithVariableName_UsedVerbatim(t *testing.T) {
	enableVarSubstitution(t)

	assert.Equal(t,
		String{varName: "Book_Access_Key_2"},
		New("", WithVariableName("Book_Access_Key_2")),
	)
}
