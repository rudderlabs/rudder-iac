package ingest

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// jsonString mirrors GetStringOrEmpty, which the oracle uses to read ids.
func TestJSONStringStringifiesLikeTheOracle(t *testing.T) {
	t.Parallel()

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(`{"s":"x","n":42,"b":true,"z":null,"o":{"a":1}}`), &m))

	require.Equal(t, "x", jsonString(m["s"]))
	require.Equal(t, "42", jsonString(m["n"]))
	require.Equal(t, "true", jsonString(m["b"]))
	require.Equal(t, "", jsonString(m["z"]))
	require.Equal(t, "", jsonString(m["missing"]))
	require.Equal(t, `{"a":1}`, jsonString(m["o"]))
}

func TestNumericUserIDIdentifiesAndExtractNeedsNoID(t *testing.T) {
	t.Parallel()

	require.False(t, identityOf(json.RawMessage(`{"userId":7}`)).nonIdentifiable())
	require.False(t, identityOf(json.RawMessage(`{"type":"extract"}`)).nonIdentifiable())
	require.True(t, identityOf(json.RawMessage(`{"type":"track"}`)).nonIdentifiable())
}

func TestNewUUIDv4HasVersionAndVariantBits(t *testing.T) {
	t.Parallel()

	id := newUUIDv4()

	require.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`, id)
	require.NotEqual(t, id, newUUIDv4())
}
