package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// draft202012 is the JSON Schema dialect every emitted schema must declare.
const draft202012 = "https://json-schema.org/draft/2020-12/schema"

func TestRegistryCoversMainlineKinds(t *testing.T) {
	kinds := Kinds()

	// The mainline (non-experimental) kinds must all be present. This guards
	// against a provider adding a kind without a corresponding schema mapping.
	for _, kind := range []string{
		"properties",
		"events",
		"categories",
		"custom-types",
		"tracking-plan",
		"retl-source-sql-model",
		"event-stream-source",
		"transformation",
		"transformation-library",
	} {
		assert.Contains(t, kinds, kind, "registry should cover kind %q", kind)
	}
}

func TestForKindEmitsDraft202012Envelope(t *testing.T) {
	s, err := ForKind("tracking-plan")
	require.NoError(t, err)

	raw, err := json.Marshal(s)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(raw, &doc))

	assert.Equal(t, draft202012, doc["$schema"], "must declare Draft 2020-12 dialect")

	props, ok := doc["properties"].(map[string]any)
	require.True(t, ok, "schema must expose top-level properties")
	for _, field := range []string{"version", "kind", "metadata", "spec"} {
		assert.Contains(t, props, field, "envelope must include %q", field)
	}

	// The kind field must be pinned to a constant so editors can associate the
	// right sub-schema and reject a mistyped kind.
	kindProp, ok := props["kind"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "tracking-plan", kindProp["const"])

	required, _ := doc["required"].([]any)
	assert.Subset(t, required, []any{"version", "kind", "spec"})
}

func TestForKindUnknownKind(t *testing.T) {
	_, err := ForKind("does-not-exist")
	assert.ErrorIs(t, err, ErrUnknownKind)
}

func TestAllGeneratesEveryKind(t *testing.T) {
	all, err := All()
	require.NoError(t, err)

	for _, kind := range Kinds() {
		s, ok := all[kind]
		require.True(t, ok, "All() must include kind %q", kind)
		assert.NotNil(t, s)
	}
}

func TestRootSchemaDiscriminatesByKind(t *testing.T) {
	root, err := Root()
	require.NoError(t, err)

	raw, err := json.Marshal(root)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(raw, &doc))

	assert.Equal(t, draft202012, doc["$schema"])
	oneOf, ok := doc["oneOf"].([]any)
	require.True(t, ok, "root schema must branch on kind via oneOf")
	assert.Len(t, oneOf, len(Kinds()))
}
