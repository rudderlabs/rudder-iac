package datacatalog_test

import (
	"encoding/json"
	"testing"

	compiledschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

func TestProvider_SpecSchemasCoverSupportedKinds(t *testing.T) {
	t.Parallel()

	p := datacatalog.New(&datacatalog.EmptyCatalog{})

	assert.ElementsMatch(t, p.SupportedKinds(), schema.Kinds(p.SpecSchemas()))
}

func TestProvider_TrackingPlanSchemasHideExpandedEventProps(t *testing.T) {
	t.Parallel()

	p := datacatalog.New(&datacatalog.EmptyCatalog{})
	for _, kind := range []string{localcatalog.KindTrackingPlans, localcatalog.KindTrackingPlansV1} {
		raw, err := json.Marshal(p.SpecSchemas()[kind])
		require.NoError(t, err)
		assert.NotContains(t, string(raw), "event_props", kind)
		assert.NotContains(t, string(raw), "localID", kind)
	}
}

func TestProvider_SpecSchemasSelectDataCatalogBodyByVersion(t *testing.T) {
	p := datacatalog.New(&datacatalog.EmptyCatalog{})

	properties := compileSchema(t, p.SpecSchemas()[localcatalog.KindProperties])
	assert.NoError(t, validateYAML(t, properties, `
version: rudder/v0.1
kind: properties
metadata:
  name: legacy
spec:
  properties:
    - id: legacy
      name: Legacy
      propConfig:
        format: email
`))
	assert.NoError(t, validateYAML(t, properties, `
version: rudder/v1
kind: properties
metadata:
  name: current
spec:
  properties:
    - id: current
      name: Current
      config:
        format: email
`))

	events := compileSchema(t, p.SpecSchemas()[localcatalog.KindEvents])
	assert.NoError(t, validateYAML(t, events, `
version: rudder/v0.1
kind: events
metadata:
  name: legacy
spec:
  events:
    - id: legacy
`))
	assert.Error(t, validateYAML(t, events, `
version: rudder/v1
kind: events
metadata:
  name: current
spec:
  events:
    - id: current
`))
}

func compileSchema(t *testing.T, generated any) *compiledschema.Schema {
	t.Helper()
	raw, err := json.Marshal(generated)
	require.NoError(t, err)
	var document any
	require.NoError(t, json.Unmarshal(raw, &document))
	compiler := compiledschema.NewCompiler()
	require.NoError(t, compiler.AddResource("mem://schema.json", document))
	compiled, err := compiler.Compile("mem://schema.json")
	require.NoError(t, err)
	return compiled
}

func validateYAML(t *testing.T, validator *compiledschema.Schema, input string) error {
	t.Helper()
	var document any
	require.NoError(t, yaml.Unmarshal([]byte(input), &document))
	return validator.Validate(document)
}

func TestProvider_SupportedMatchPatterns(t *testing.T) {
	t.Parallel()

	p := datacatalog.New(&datacatalog.EmptyCatalog{})

	var want []vrules.MatchPattern
	for _, kind := range []string{
		localcatalog.KindProperties,
		localcatalog.KindEvents,
		localcatalog.KindCategories,
		localcatalog.KindCustomTypes,
	} {
		want = append(want, prules.LegacyVersionPatterns(kind)...)
		want = append(want, prules.V1VersionPatterns(kind)...)
	}
	want = append(want, prules.LegacyVersionPatterns(localcatalog.KindTrackingPlans)...)
	want = append(want, prules.V1VersionPatterns(localcatalog.KindTrackingPlansV1)...)

	assert.ElementsMatch(t, want, p.SupportedMatchPatterns())
}
