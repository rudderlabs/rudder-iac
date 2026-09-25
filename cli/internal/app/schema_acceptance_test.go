package app

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/connection"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema"
	compiledschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type fixtureEnvelope struct {
	Version string         `yaml:"version"`
	Kind    string         `yaml:"kind"`
	Spec    map[string]any `yaml:"spec"`
}

func TestGeneratedSchemasRejectInvalidProjectFixtures(t *testing.T) {
	enableAllSchemaKinds(t)

	schemas, err := GenerateSchemas()
	require.NoError(t, err)
	compiled := compileSchemas(t, schemas)

	invalid := []struct {
		name string
		kind string
		yaml string
	}{
		{
			name: "unknown nested retl schedule field",
			kind: "retl-connections",
			yaml: `version: rudder/v1
kind: retl-connections
metadata:
  name: orders-to-http
spec:
  connections:
    - id: orders-to-http
      source: "#retl-source-sql-model:orders-model"
      destination: "#destination:e2e-retl-http"
      config:
        sync_behaviour: upsert
        schedule:
          type: basic
          every_minutes: 30
          evry_minutes: 15
        identifiers:
          - from: id
            to: user_id
`,
		},
		{
			name: "transformation requires code or file",
			kind: "transformation",
			yaml: `version: rudder/v1
kind: transformation
metadata:
  name: broken-transform
spec:
  id: broken_transform
  name: Broken Transform
  language: javascript
`,
		},
		{
			name: "sql model requires sql or file",
			kind: "retl-source-sql-model",
			yaml: `version: rudder/v1
kind: retl-source-sql-model
metadata:
  name: orders-model
spec:
  id: orders-model
  display_name: Orders Model
  account: "#account:retl-pg"
  primary_key: id
  source_definition: postgres
`,
		},
		{
			name: "sql model requires account or account_id",
			kind: "retl-source-sql-model",
			yaml: `version: rudder/v1
kind: retl-source-sql-model
metadata:
  name: orders-model
spec:
  id: orders-model
  display_name: Orders Model
  primary_key: id
  source_definition: postgres
  sql: SELECT id, email FROM orders
`,
		},
		{
			name: "property types validate each item",
			kind: "properties",
			yaml: `version: rudder/v1
kind: properties
metadata:
  name: broken-properties
spec:
  properties:
    - id: email_address
      name: Email Address
      types:
        - string
        - bogus
`,
		},
		{
			name: "destination rejects unsupported connection mode",
			kind: "destination",
			yaml: `version: rudder/v1
kind: destination
metadata:
  name: firebase
spec:
  id: firebase
  display_name: Firebase
  type: firebase
  enabled: true
  definition_version: 1
  config:
    connection_mode:
      warehouse: cloud
`,
		},
		{
			name: "retl connection requires config",
			kind: "retl-connections",
			yaml: `version: rudder/v1
kind: retl-connections
metadata:
  name: missing-config
spec:
  connections:
    - id: users-to-webhook
      source: "#retl-source-sql-model:users"
      destination: "#destination:webhook"
`,
		},
	}

	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			validator, ok := compiled[tt.kind]
			require.True(t, ok, "generated schema for %q is required", tt.kind)
			assert.Error(t, validateFixture(validator, []byte(tt.yaml)))
		})
	}
}

func TestGeneratedSchemasValidateProjectFixtures(t *testing.T) {
	enableAllSchemaKinds(t)

	schemas, err := GenerateSchemas()
	require.NoError(t, err)
	compiled := compileSchemas(t, schemas)
	root := compileRootSchema(t, schemas)

	var (
		fixtureCount int
		seenKinds    = make(map[string]bool)
	)
	testdata := filepath.Join("..", "..", "tests", "testdata")
	require.NoError(t, filepath.WalkDir(testdata, func(path string, entry fs.DirEntry, walkErr error) error {
		require.NoError(t, walkErr)
		if entry.IsDir() || (filepath.Ext(path) != ".yaml" && filepath.Ext(path) != ".yml") {
			return nil
		}

		data, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		var envelope fixtureEnvelope
		if yaml.Unmarshal(data, &envelope) != nil || envelope.Version == "" || envelope.Kind == "" || envelope.Spec == nil {
			// Variable files, expected snapshots, manifests, and support YAML are not specs.
			return nil
		}

		validator, ok := compiled[envelope.Kind]
		if !ok && flagGatedKind(envelope.Kind) {
			return nil
		}
		require.True(t, ok, "full spec fixture %s uses an unsupported kind %q", path, envelope.Kind)
		fixtureCount++
		seenKinds[envelope.Kind] = true
		assert.NoError(t, validateFixture(validator, data), path)
		assert.NoError(t, validateFixture(root, data), "root schema: "+path)
		return nil
	}))

	assert.Greater(t, fixtureCount, 100, "expected broad cli/tests/testdata coverage")
	for kind, generated := range schemas {
		if seenKinds[kind] {
			continue
		}
		assert.NoError(t, validateFixture(compileSchema(t, kind, generated), representativeFixture(kind)),
			"active kind %q needs a valid fixture", kind)
	}
}

func TestGeneratedRootSchemaMatchesLegacyTrackingPlanVersions(t *testing.T) {
	enableAllSchemaKinds(t)
	schemas, err := GenerateSchemas()
	require.NoError(t, err)
	root := compileRootSchema(t, schemas)

	legacyPath := filepath.Join("..", "providers", "datacatalog", "localcatalog", "testdata", "trackingplan_1.yaml")
	legacy, err := os.ReadFile(legacyPath)
	require.NoError(t, err)
	assert.NoError(t, validateFixture(root, legacy))

	invalidV1 := []byte("version: rudder/v1\nkind: tp\nmetadata:\n  name: legacy-name\nspec:\n  id: legacy-name\n  display_name: Legacy Name\n")
	assert.Error(t, validateFixture(root, invalidV1))
}

func enableAllSchemaKinds(t *testing.T) {
	t.Helper()
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	t.Setenv("RUDDERSTACK_X_RETL_TABLE_SUPPORT", "true")
	t.Setenv("RUDDERSTACK_X_RETL_CONNECTION_SUPPORT", "true")
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
	previous := viper.Get("flags.retlConnectionSupport")
	previousUnverified := viper.Get("flags.unverifiedDestinations")
	viper.Set("flags.retlConnectionSupport", true)
	viper.Set("flags.unverifiedDestinations", true)
	t.Cleanup(func() {
		viper.Set("flags.retlConnectionSupport", previous)
		viper.Set("flags.unverifiedDestinations", previousUnverified)
	})
}

func compileSchemas(t *testing.T, schemas schema.Set) map[string]*compiledschema.Schema {
	t.Helper()
	compiled := make(map[string]*compiledschema.Schema, len(schemas))
	for kind, generated := range schemas {
		compiled[kind] = compileSchema(t, kind, generated)
	}

	_ = compileRootSchema(t, schemas)
	return compiled
}

func compileRootSchema(t *testing.T, schemas schema.Set) *compiledschema.Schema {
	t.Helper()
	root, err := schema.MarshalRoot(schemas)
	require.NoError(t, err)
	var document any
	require.NoError(t, json.Unmarshal(root, &document))
	compiler := compiledschema.NewCompiler()
	require.NoError(t, compiler.AddResource("mem://root.json", document))
	compiled, err := compiler.Compile("mem://root.json")
	require.NoError(t, err)
	return compiled
}

func compileSchema(t *testing.T, kind string, generated any) *compiledschema.Schema {
	t.Helper()
	raw, err := json.Marshal(generated)
	require.NoError(t, err)
	var document any
	require.NoError(t, json.Unmarshal(raw, &document))
	compiler := compiledschema.NewCompiler()
	url := "mem://" + kind + ".json"
	require.NoError(t, compiler.AddResource(url, document))
	compiled, err := compiler.Compile(url)
	require.NoError(t, err, kind)
	return compiled
}

func flagGatedKind(kind string) bool {
	return kind == connection.ResourceKind || kind == table.ResourceKind
}

func representativeFixture(kind string) []byte {
	if kind == "import-manifest" {
		return []byte(`version: rudder/v1
kind: import-manifest
metadata:
  name: import-manifest
spec:
  workspaces:
    - workspace_id: workspace-1
      resources:
        - urn: source:source-1
          remote_id: remote-1
`)
	}
	if kind == "data-graph" {
		return []byte(`version: rudder/v1
kind: data-graph
metadata:
  name: Customer graph
spec:
  id: customer-graph
  account_id: warehouse-account
  models:
    - id: customer
      display_name: Customer
      type: entity
      table: customers
      primary_id: id
`)
	}
	return nil
}

func validateFixture(validator *compiledschema.Schema, data []byte) error {
	var yamlDocument any
	if err := yaml.Unmarshal(data, &yamlDocument); err != nil {
		return err
	}
	raw, err := json.Marshal(yamlDocument)
	if err != nil {
		return err
	}
	var document any
	if err := json.Unmarshal(raw, &document); err != nil {
		return err
	}
	return validator.Validate(document)
}
