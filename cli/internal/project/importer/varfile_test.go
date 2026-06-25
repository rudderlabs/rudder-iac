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
		// Every generated entity is scanned, spec file or not.
		{Content: "select * from {{ .RAW_TABLE }}", RelativePath: "raw.txt"},
	}

	assert.Equal(t,
		[]string{"A_TOKEN", "B_ACCESS_KEY", "RAW_TABLE"},
		collectVariableNames(entities),
	)
}

func TestScaffoldSecretsVarFile(t *testing.T) {
	enableVarSubstitution(t)
	dir := t.TempDir()

	entities := []writer.FormattableEntity{
		specEntity(map[string]any{"accessKey": "{{ .BOOKS_ACCESS_KEY }}"}),
	}

	path, err := scaffoldSecretsVarFile(t.Context(), dir, entities)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, SecretsVarFileName), path)

	// nil placeholder: the var-file resolver rejects null values, so an
	// unfilled placeholder fails apply loudly instead of sending "".
	assert.Equal(t, map[string]any{"BOOKS_ACCESS_KEY": nil}, readVarFile(t, path))

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "# Variables referenced by the imported specs")
	assert.Contains(t, string(raw), "BOOKS_ACCESS_KEY:\n")
}

// An existing var file may hold real secret values the user filled in, so
// scaffolding never overwrites or merges into it — it errors out instead.
func TestScaffoldSecretsVarFile_ExistingFileErrors(t *testing.T) {
	enableVarSubstitution(t)
	dir := t.TempDir()

	existing := filepath.Join(dir, SecretsVarFileName)
	require.NoError(t, os.WriteFile(existing, []byte("BOOKS_ACCESS_KEY: \"filled-in\"\n"), 0600))

	entities := []writer.FormattableEntity{
		specEntity(map[string]any{"accessKey": "{{ .BOOKS_ACCESS_KEY }}"}),
	}

	_, err := scaffoldSecretsVarFile(t.Context(), dir, entities)
	require.ErrorIs(t, err, os.ErrExist)
	assert.Equal(t, map[string]any{"BOOKS_ACCESS_KEY": "filled-in"}, readVarFile(t, existing))
}

func TestScaffoldSecretsVarFile_NoVariables(t *testing.T) {
	enableVarSubstitution(t)
	dir := t.TempDir()

	path, err := scaffoldSecretsVarFile(t.Context(), dir, []writer.FormattableEntity{
		specEntity(map[string]any{"plain": "value"}),
	})
	require.NoError(t, err)
	assert.Empty(t, path)
	assert.NoFileExists(t, filepath.Join(dir, SecretsVarFileName))
}

func TestScaffoldSecretsVarFile_GateOff(t *testing.T) {
	dir := t.TempDir()

	path, err := scaffoldSecretsVarFile(t.Context(), dir, []writer.FormattableEntity{
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
