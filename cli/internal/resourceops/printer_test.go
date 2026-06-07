package resourceops_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/rudderlabs/rudder-iac/cli/internal/resourceops"
)

func TestEncodeYAML_ValidOutput(t *testing.T) {
	input := map[string]any{
		"kind":    "EventStreamSource",
		"version": "1",
		"enabled": true,
	}

	out, err := resourceops.EncodeYAML(input)
	require.NoError(t, err)
	require.NotEmpty(t, out)

	// Must be parseable YAML.
	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(out), &parsed))

	assert.Equal(t, "EventStreamSource", parsed["kind"])
	assert.Equal(t, "1", parsed["version"])
	assert.Equal(t, true, parsed["enabled"])
}

func TestEncodeYAML_StringValuesDoubleQuoted(t *testing.T) {
	input := map[string]any{
		"name": "my-source",
	}

	out, err := resourceops.EncodeYAML(input)
	require.NoError(t, err)

	// The project formatter double-quotes string values.
	assert.True(t, strings.Contains(out, `"my-source"`), "expected double-quoted string value in YAML output: %q", out)
}

func TestEncodeJSON_ValidOutput(t *testing.T) {
	input := map[string]any{
		"kind":    "EventStreamSource",
		"version": "1",
		"enabled": true,
		"nested": map[string]any{
			"count": 42,
		},
	}

	out, err := resourceops.EncodeJSON(input)
	require.NoError(t, err)
	require.NotEmpty(t, out)

	// Must be valid JSON.
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))

	// Top-level keys must use the lowercase YAML field names.
	assert.Equal(t, "EventStreamSource", parsed["kind"])
	assert.Equal(t, "1", parsed["version"])
	assert.Equal(t, true, parsed["enabled"])

	// Nested map round-trips without loss.
	nested, ok := parsed["nested"].(map[string]any)
	require.True(t, ok, "expected nested map")
	assert.EqualValues(t, 42, nested["count"])
}

func TestEncodeJSON_RoundTripsBoolean(t *testing.T) {
	input := map[string]any{
		"active": false,
	}

	out, err := resourceops.EncodeJSON(input)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))

	assert.Equal(t, false, parsed["active"])
}
