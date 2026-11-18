package importer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

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
		// A raw code file is loaded verbatim, never substituted, so its
		// "{{ .NAME }}" is code and needs no var file entry.
		{Content: "export function transformEvent(e) { return '{{ .CODE_ONLY }}' }", RelativePath: "transformations/t.js"},
	}

	assert.Equal(t,
		[]string{"A_TOKEN", "B_ACCESS_KEY"},
		collectVariableNames(entities),
	)
}

func TestScaffoldSecretsVarFile(t *testing.T) {
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
	assert.Contains(t, string(raw), "# For a multi-line value such as a PEM certificate or key, use a YAML literal")
	assert.Contains(t, string(raw), "# The generated '{{ .VAR }}' references keep those line breaks.")
	assert.Contains(t, string(raw), "BOOKS_ACCESS_KEY:\n")
}

// An existing var file may hold real secret values the user filled in, so
// scaffolding never overwrites or merges into it — it errors out instead.
func TestScaffoldSecretsVarFile_ExistingFileErrors(t *testing.T) {
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
	dir := t.TempDir()

	path, err := scaffoldSecretsVarFile(t.Context(), dir, []writer.FormattableEntity{
		specEntity(map[string]any{"plain": "value"}),
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

// TestScaffoldSecretsVarFile_RefusesSharedVariables pins that two generated
// files never share a secret variable. Names come from the resource id and key
// path joined with "_", so a destination and an account both imported as "pg"
// with a password both reference PG_PASSWORD, and one entry would fill both.
func TestScaffoldSecretsVarFile_RefusesSharedVariables(t *testing.T) {
	dir := t.TempDir()

	_, err := scaffoldSecretsVarFile(t.Context(), dir, []writer.FormattableEntity{
		{Content: map[string]any{"password": "{{ .PG_PASSWORD }}"}, RelativePath: "destinations/pg.yaml"},
		{Content: map[string]any{"password": "{{ .PG_PASSWORD }}"}, RelativePath: "accounts/pg.yaml"},
		{Content: map[string]any{"token": "{{ .OTHER_TOKEN }}", "again": "{{ .OTHER_TOKEN }}"}, RelativePath: "destinations/other.yaml"},
	})
	require.EqualError(t, err, "imported specs would share secret variables: PG_PASSWORD (in destinations/pg.yaml and accounts/pg.yaml); "+
		"the names come from each resource's id, so rename one of the resources whose ids collide and import again")
	_, statErr := os.Stat(filepath.Join(dir, SecretsVarFileName))
	assert.True(t, os.IsNotExist(statErr), "no var file is written for a refused import")
}

// TestScaffoldSecretsVarFile_IgnoresSharedReferencesInRawCode pins that raw code
// files, which are loaded verbatim rather than variable-substituted, never
// count as sharing a secret variable: two transformation scripts may hold the
// same literal "{{ .NAME }}".
func TestScaffoldSecretsVarFile_IgnoresSharedReferencesInRawCode(t *testing.T) {
	_, err := scaffoldSecretsVarFile(t.Context(), t.TempDir(), []writer.FormattableEntity{
		{Content: "export function transformEvent(e) { return '{{ .NAME }}' }", RelativePath: "transformations/a.js"},
		{Content: "def transformEvent(e): return '{{ .NAME }}'", RelativePath: "transformations/b.py"},
	})
	require.NoError(t, err)
}
