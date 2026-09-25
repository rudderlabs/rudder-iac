package app

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	tekuri "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/require"
)

func TestGeneratedSchemasCompileIndependently(t *testing.T) {
	Initialise("test")
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
	catalog, err := GenerateSchemas()
	require.NoError(t, err)

	for _, kind := range catalog.Kinds() {
		data, err := catalog.MarshalKind(kind)
		require.NoError(t, err)
		requireSchemaCompiles(t, kind, data)
	}
	data, err := catalog.MarshalRoot()
	require.NoError(t, err)
	requireSchemaCompiles(t, "root", data)
}

func requireSchemaCompiles(t *testing.T, name string, data []byte) {
	t.Helper()
	var document any
	require.NoError(t, json.Unmarshal(data, &document))
	compiler := tekuri.NewCompiler()
	url := "mem://" + name + ".json"
	require.NoError(t, compiler.AddResource(url, document))
	_, err := compiler.Compile(url)
	require.NoError(t, err)
}
