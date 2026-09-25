package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	tekuri "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGeneratedSourceSchemaAcceptsOptionalNestedConfig(t *testing.T) {
	catalog := requireSchemaCatalog(t)
	compiled := requireCompiledKindSchema(t, catalog, "event-stream-source")

	var specDocument any
	require.NoError(t, yaml.Unmarshal([]byte("version: rudder/v1\nkind: event-stream-source\nmetadata:\n  name: example\nspec:\n  id: example\n  name: Example\n  type: javascript\n"), &specDocument))
	require.NoError(t, compiled.Validate(specDocument))
}

func TestGeneratedDestinationSchemaAcceptsNumericOneOfFixture(t *testing.T) {
	catalog := requireSchemaCatalog(t)
	compiled := requireCompiledKindSchema(t, catalog, "destination")
	fixture := filepath.Join("..", "..", "tests", "testdata", "destinations", "update", "am.yaml")
	document := requireYAMLDocument(t, fixture)
	require.NoError(t, compiled.Validate(document))
}

func TestGeneratedDestinationSchemaRejectsUnsupportedConnectionMode(t *testing.T) {
	catalog := requireSchemaCatalog(t)
	compiled := requireCompiledKindSchema(t, catalog, "destination")

	var document any
	require.NoError(t, yaml.Unmarshal([]byte(`version: rudder/v1
kind: destination
metadata:
  name: webhook
spec:
  id: webhook
  display_name: Webhook
  type: webhook
  enabled: true
  definition_version: 1
  config:
    webhook_url: https://webhooks.example.com/rudder
    webhook_method: POST
    connection_mode:
      warehouse: cloud
`), &document))
	require.Error(t, compiled.Validate(document))
}

func TestGeneratedRETLConnectionSchemaRejectsOmittedConfig(t *testing.T) {
	catalog := requireSchemaCatalog(t)
	compiled := requireCompiledKindSchema(t, catalog, "retl-connections")

	var document any
	require.NoError(t, yaml.Unmarshal([]byte(`version: rudder/v1
kind: retl-connections
metadata:
  name: missing-config
spec:
  connections:
    - id: users-to-webhook
      source: "#retl-source-sql-model:users"
      destination: "#destination:webhook"
`), &document))
	require.Error(t, compiled.Validate(document))
}

func TestGeneratedRootSchemaMatchesLegacyTrackingPlanVersions(t *testing.T) {
	catalog := requireSchemaCatalog(t)
	compiled := requireCompiledRootSchema(t, catalog)
	legacy := requireYAMLDocument(t, filepath.Join("..", "..", "internal", "providers", "datacatalog", "localcatalog", "testdata", "trackingplan_1.yaml"))
	require.NoError(t, compiled.Validate(legacy))

	var invalidV1 any
	require.NoError(t, yaml.Unmarshal([]byte(`version: rudder/v1
kind: tp
metadata:
  name: legacy-name
spec:
  id: legacy-name
  display_name: Legacy Name
`), &invalidV1))
	require.Error(t, compiled.Validate(invalidV1))
}

func TestGeneratedRootSchemaValidatesExpectedSpecFixtures(t *testing.T) {
	catalog := requireSchemaCatalog(t)
	compiled := requireCompiledRootSchema(t, catalog)
	root := filepath.Join("..", "..", "tests", "testdata")

	var validated []string
	require.NoError(t, filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		require.NoError(t, err)
		if entry.IsDir() || !isYAMLFile(path) || strings.Contains(filepath.Base(path), ".vars.") {
			return nil
		}

		document := requireYAMLDocument(t, path)
		if !isSpecDocument(document) {
			return nil
		}

		validated = append(validated, path)
		require.NoError(t, compiled.Validate(document), path)
		return nil
	}))
	require.NotEmpty(t, validated)
}

func requireSchemaCatalog(t *testing.T) schemaCatalog {
	t.Helper()
	Initialise("test")
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
	viper.Set("experimental", true)
	viper.Set("flags.unverifiedDestinations", true)
	viper.Set("flags.retlConnectionSupport", true)
	viper.Set("flags.retlTableSupport", true)
	t.Cleanup(viper.Reset)

	catalog, err := GenerateSchemas()
	require.NoError(t, err)
	return catalog
}

type schemaCatalog interface {
	MarshalKind(kind string) ([]byte, error)
	MarshalRoot() ([]byte, error)
}

func requireCompiledKindSchema(t *testing.T, catalog schemaCatalog, kind string) *tekuri.Schema {
	t.Helper()
	data, err := catalog.MarshalKind(kind)
	require.NoError(t, err)
	return requireCompiledSchema(t, kind, data)
}

func requireCompiledRootSchema(t *testing.T, catalog schemaCatalog) *tekuri.Schema {
	t.Helper()
	data, err := catalog.MarshalRoot()
	require.NoError(t, err)
	return requireCompiledSchema(t, "root", data)
}

func requireCompiledSchema(t *testing.T, name string, data []byte) *tekuri.Schema {
	t.Helper()
	var schemaDocument any
	require.NoError(t, json.Unmarshal(data, &schemaDocument))
	compiler := tekuri.NewCompiler()
	url := "mem://" + name + ".json"
	require.NoError(t, compiler.AddResource(url, schemaDocument))
	compiled, err := compiler.Compile(url)
	require.NoError(t, err)
	return compiled
}

func requireYAMLDocument(t *testing.T, path string) any {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var document any
	require.NoError(t, yaml.Unmarshal(data, &document))
	return document
}

func isSpecDocument(document any) bool {
	object, ok := document.(map[string]any)
	if !ok {
		return false
	}
	_, hasVersion := object["version"]
	_, hasKind := object["kind"]
	_, hasSpec := object["spec"]
	return hasVersion && hasKind && hasSpec
}

func isYAMLFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".yaml" || ext == ".yml"
}
