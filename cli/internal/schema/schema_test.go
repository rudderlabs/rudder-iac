package schema

import (
	"encoding/json"
	"testing"

	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// draft202012 is the JSON Schema dialect every emitted schema must declare.
const draft202012 = "https://json-schema.org/draft/2020-12/schema"

type trackingPlanSpec struct {
	ID          string `json:"id" validate:"required"`
	DisplayName string `json:"display_name" validate:"required"`
}

func TestForKindEmitsDraft202012Envelope(t *testing.T) {
	s := ForKind("tracking-plan", trackingPlanSpec{})

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
	assert.ElementsMatch(t, []any{"version", "kind", "metadata", "spec"}, required)

	metadata, ok := props["metadata"].(map[string]any)
	require.True(t, ok)
	metadataProperties, ok := metadata["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, metadataProperties, "name")
	assert.Contains(t, metadataProperties, "import")
	assert.ElementsMatch(t, []any{"name"}, metadata["required"])
}

func TestForKindUsesYAMLFieldNamesForMapstructureModels(t *testing.T) {
	type mapstructureSpec struct {
		ID                string `json:"id" mapstructure:"id" validate:"required"`
		DefinitionVersion int64  `json:"definition_version" mapstructure:"definition_version" validate:"required"`
	}

	s := ForKind("example", mapstructureSpec{})
	spec := property(t, s, "spec")

	assert.Equal(t, []string{"id", "definition_version"}, spec.Required)
	assert.NotNil(t, property(t, spec, "id"))
	assert.NotNil(t, property(t, spec, "definition_version"))
	_, hasIDTypo := spec.Properties.Get("iD")
	_, hasVersionTypo := spec.Properties.Get("definitionVersion")
	assert.False(t, hasIDTypo)
	assert.False(t, hasVersionTypo)
}

func TestSetHelpers(t *testing.T) {
	schemas := Set{
		"tracking-plan": ForKind("tracking-plan", trackingPlanSpec{}),
		"properties":    ForKind("properties", struct{}{}),
	}

	assert.Equal(t, []string{"properties", "tracking-plan"}, Kinds(schemas))

	raw, err := MarshalKind(schemas, "tracking-plan")
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"const": "tracking-plan"`)

	_, err = MarshalKind(schemas, "does-not-exist")
	assert.ErrorIs(t, err, ErrUnknownKind)
}

func TestVersionsForKindUsesSupportedMatchPatterns(t *testing.T) {
	patterns := []vrules.MatchPattern{
		vrules.MatchKindVersion("example", "rudder/v1"),
		vrules.MatchKindVersion("other", "rudder/v2"),
		vrules.MatchKindVersion("example", "rudder/v0.1"),
		vrules.MatchKindVersion("example", "rudder/v1"),
	}

	assert.Equal(t, []string{"rudder/v0.1", "rudder/v1"}, VersionsForKind("example", patterns))
}

func TestMetadataIsRequiredAndStructured(t *testing.T) {
	s := ForKindVersions("example", trackingPlanSpec{}, "rudder/v1")

	raw, err := json.Marshal(s)
	require.NoError(t, err)
	compiled := compileDocument(t, raw)

	assert.Error(t, validateYAML(t, compiled, "version: rudder/v1\nkind: example\nspec:\n  id: id\n  display_name: Example\n"))
	assert.Error(t, validateYAML(t, compiled, "version: rudder/v1\nkind: example\nmetadata: {}\nspec:\n  id: id\n  display_name: Example\n"))
	assert.NoError(t, validateYAML(t, compiled, "version: rudder/v1\nkind: example\nmetadata:\n  name: example\nspec:\n  id: id\n  display_name: Example\n"))
}

func TestForKindVersionsRejectsUnsupportedVersion(t *testing.T) {
	s := ForKindVersions("example", trackingPlanSpec{}, "rudder/v1")

	raw, err := json.Marshal(s)
	require.NoError(t, err)
	compiled := compileDocument(t, raw)

	assert.NoError(t, validateYAML(t, compiled, "version: rudder/v1\nkind: example\nmetadata:\n  name: example\nspec:\n  id: id\n  display_name: Example\n"))
	assert.Error(t, validateYAML(t, compiled, "version: rudder/v0.1\nkind: example\nmetadata:\n  name: example\nspec:\n  id: id\n  display_name: Example\n"))
}

func TestForVersionedKindSelectsBodyByVersion(t *testing.T) {
	s := ForVersionedKind("example", map[string]any{
		"rudder/v0.1": struct {
			Legacy string `json:"legacy" validate:"required"`
		}{},
		"rudder/v1": struct {
			Current string `json:"current" validate:"required"`
		}{},
	})

	raw, err := json.Marshal(s)
	require.NoError(t, err)
	compiled := compileDocument(t, raw)

	assert.NoError(t, validateYAML(t, compiled, "version: rudder/v0.1\nkind: example\nmetadata:\n  name: example\nspec:\n  legacy: value\n"))
	assert.NoError(t, validateYAML(t, compiled, "version: rudder/v1\nkind: example\nmetadata:\n  name: example\nspec:\n  current: value\n"))
	assert.Error(t, validateYAML(t, compiled, "version: rudder/v1\nkind: example\nmetadata:\n  name: example\nspec:\n  legacy: value\n"))
}

func TestRootSchemaDiscriminatesByKind(t *testing.T) {
	schemas := Set{
		"properties":    ForKind("properties", struct{}{}),
		"tracking-plan": ForKind("tracking-plan", trackingPlanSpec{}),
	}
	root, err := Root(schemas)
	require.NoError(t, err)

	raw, err := json.Marshal(root)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(raw, &doc))

	assert.Equal(t, draft202012, doc["$schema"])
	oneOf, ok := doc["oneOf"].([]any)
	require.True(t, ok, "root schema must branch on kind via oneOf")
	assert.Len(t, oneOf, len(schemas))
}
