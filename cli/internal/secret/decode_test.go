package secret

import (
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Spec maps carry secrets as bare strings after YAML load and variable
// substitution. UnmarshalMapstructure is implemented on the type itself, so a
// plain mapstructure.Decode — with no hook wiring — must land them in every
// secret-typed field shape. This is what lets providers with hand-rolled
// LoadSpec decoders adopt secrets without touching their decoder config.
func TestUnmarshalMapstructure(t *testing.T) {
	type spec struct {
		Name       string
		AccessKey  String
		WriteKey   *String
		Importable ImportableSecret
		PtrImport  *ImportableSecret
	}

	var out spec
	require.NoError(t, mapstructure.Decode(map[string]any{
		"name":       "main",
		"accessKey":  "hunter2-but-long",
		"writeKey":   "wk-value",
		"importable": "imp-value",
		"ptrImport":  "ptr-value",
	}, &out))

	wk := New("wk-value")
	pi := ImportableSecret{String: New("ptr-value")}
	assert.Equal(t, spec{
		Name:       "main",
		AccessKey:  New("hunter2-but-long"),
		WriteKey:   &wk,
		Importable: ImportableSecret{String: New("imp-value")},
		PtrImport:  &pi,
	}, out)
}

// Values that are already String (e.g. spec maps built in code) pass through.
func TestUnmarshalMapstructure_PassThrough(t *testing.T) {
	var out struct{ AccessKey String }
	require.NoError(t, mapstructure.Decode(map[string]any{"accessKey": New("hunter2")}, &out))
	assert.Equal(t, New("hunter2"), out.AccessKey)
}

func TestUnmarshalMapstructure_RejectsNonString(t *testing.T) {
	var out struct{ AccessKey String }
	err := mapstructure.Decode(map[string]any{"accessKey": 42}, &out)
	require.ErrorContains(t, err, "expected a string")
}
