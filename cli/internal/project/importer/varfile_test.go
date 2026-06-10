package importer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func enableVarSubstitution(t *testing.T) {
	t.Helper()
	prevExp, prevFlag := viper.Get("experimental"), viper.Get("flags.enableVarSubstitution")
	viper.Set("experimental", true)
	viper.Set("flags.enableVarSubstitution", true)
	t.Cleanup(func() {
		viper.Set("experimental", prevExp)
		viper.Set("flags.enableVarSubstitution", prevFlag)
	})
}

func specEntity(spec map[string]any) writer.FormattableEntity {
	return writer.FormattableEntity{
		Content: &specs.Spec{
			Version:  specs.SpecVersionV1,
			Kind:     "books",
			Metadata: map[string]any{"name": "books"},
			Spec:     spec,
		},
		RelativePath: "books/books.yaml",
	}
}

func TestCollectVariableNames(t *testing.T) {
	entities := []writer.FormattableEntity{
		specEntity(map[string]any{
			"books": []any{
				map[string]any{"id": "a", "accessKey": "{{ .B_ACCESS_KEY }}"},
				map[string]any{"id": "b", "nested": map[string]any{"token": "{{ .A_TOKEN }}"}},
				map[string]any{"id": "c", "plain": "no token", "count": 3},
			},
		}),
		{Content: "select * from {{ .A_TOKEN }}", RelativePath: "raw.txt"},
	}

	assert.Equal(t,
		[]string{"A_TOKEN", "B_ACCESS_KEY"},
		collectVariableNames(entities),
	)
}

func TestScaffoldSecretsVarFile(t *testing.T) {
	enableVarSubstitution(t)
	dir := t.TempDir()

	entities := []writer.FormattableEntity{
		specEntity(map[string]any{"accessKey": "{{ .BOOKS_ACCESS_KEY }}"}),
	}

	path, err := scaffoldSecretsVarFile(dir, entities)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, SecretsVarFileName), path)

	assert.Equal(t, map[string]any{"BOOKS_ACCESS_KEY": ""}, readVarFile(t, path))
}

// Re-imports must keep values the user already filled in and only add
// placeholders for new variables.
func TestScaffoldSecretsVarFile_MergesExisting(t *testing.T) {
	enableVarSubstitution(t)
	dir := t.TempDir()

	existing := filepath.Join(dir, SecretsVarFileName)
	require.NoError(t, os.WriteFile(existing, []byte("BOOKS_ACCESS_KEY: \"filled-in\"\nUNRELATED: \"kept\"\n"), 0600))

	entities := []writer.FormattableEntity{
		specEntity(map[string]any{
			"accessKey": "{{ .BOOKS_ACCESS_KEY }}",
			"writeKey":  "{{ .BOOKS_WRITE_KEY }}",
		}),
	}

	path, err := scaffoldSecretsVarFile(dir, entities)
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		"BOOKS_ACCESS_KEY": "filled-in",
		"BOOKS_WRITE_KEY":  "",
		"UNRELATED":        "kept",
	}, readVarFile(t, path))
}

func TestScaffoldSecretsVarFile_NoVariables(t *testing.T) {
	enableVarSubstitution(t)
	dir := t.TempDir()

	path, err := scaffoldSecretsVarFile(dir, []writer.FormattableEntity{
		specEntity(map[string]any{"plain": "value"}),
	})
	require.NoError(t, err)
	assert.Empty(t, path)
	assert.NoFileExists(t, filepath.Join(dir, SecretsVarFileName))
}

func TestScaffoldSecretsVarFile_GateOff(t *testing.T) {
	dir := t.TempDir()

	path, err := scaffoldSecretsVarFile(dir, []writer.FormattableEntity{
		specEntity(map[string]any{"accessKey": "{{ .BOOKS_ACCESS_KEY }}"}),
	})
	require.NoError(t, err)
	assert.Empty(t, path)
	assert.NoFileExists(t, filepath.Join(dir, SecretsVarFileName))
}

func readVarFile(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	vars := make(map[string]any)
	require.NoError(t, yaml.Unmarshal(data, &vars))
	return vars
}
