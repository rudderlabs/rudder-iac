package secret

import (
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Spec maps carry secrets as bare strings after YAML load and variable
// substitution; the hook must land them in every secret-typed field shape.
func TestStringDecodeHook(t *testing.T) {
	type spec struct {
		Name       string
		AccessKey  String
		WriteKey   *String
		Importable ImportableSecret
		PtrImport  *ImportableSecret
	}

	var out spec
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: StringDecodeHook(),
		Result:     &out,
	})
	require.NoError(t, err)

	require.NoError(t, decoder.Decode(map[string]any{
		"name":       "main",
		"accessKey":  "hunter2-but-long",
		"writeKey":   "wk-value",
		"importable": "imp-value",
		"ptrImport":  "ptr-value",
	}))

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
