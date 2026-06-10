package secret

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const refToken = "{{ .SRC_MAIN_ACCESS_TOKEN }}" //nolint:gosec // a variable reference, not a credential

// A ref holds no secret, so every surface must emit it verbatim — that is the
// whole point: the variable reference has to survive the redacting marshals to
// reach the exported YAML intact.
func TestRef_SerializesVerbatim(t *testing.T) {
	ref := NewRef(refToken)

	assert.True(t, ref.IsRef())
	assert.Equal(t, refToken, ref.String())
	assert.Equal(t, refToken, fmt.Sprintf("%v", ref))
	assert.Equal(t, refToken, ref.Reveal())

	jsonBytes, err := json.Marshal(ref)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%q", refToken), string(jsonBytes))

	yamlVal, err := ref.MarshalYAML()
	require.NoError(t, err)
	assert.Equal(t, refToken, yamlVal)
}

func TestRef_DistinctFromValueStates(t *testing.T) {
	assert.NotEqual(t, NewRef(refToken), New(refToken))
	assert.False(t, New("x").IsRef())
	assert.False(t, NewUnknown().IsRef())
	assert.False(t, NewRef(refToken).IsZero())
	assert.False(t, NewRef(refToken).IsUnknown())
}

// Loading a spec value over a ref must produce a plain known secret; the ref
// state only ever exists on the export path.
func TestRef_UnmarshalResetsRef(t *testing.T) {
	s := NewRef(refToken)
	require.NoError(t, yaml.Unmarshal([]byte(`"real-value"`), &s))
	assert.Equal(t, New("real-value"), s)

	s = NewRef(refToken)
	require.NoError(t, json.Unmarshal([]byte(`"real-value"`), &s))
	assert.Equal(t, New("real-value"), s)
}

func TestStringDecodeHook(t *testing.T) {
	type spec struct {
		Name      string
		AccessKey String
	}

	var out spec
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: StringDecodeHook(),
		Result:     &out,
	})
	require.NoError(t, err)

	require.NoError(t, decoder.Decode(map[string]any{
		"name":      "main",
		"accessKey": "hunter2-but-long",
	}))
	assert.Equal(t, spec{Name: "main", AccessKey: New("hunter2-but-long")}, out)
}
