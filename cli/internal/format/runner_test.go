package format

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, dir, rel, content string) string {
	t.Helper()
	path := filepath.Join(dir, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestRun_WriteMode_RewritesUnformattedFiles(t *testing.T) {
	dir := t.TempDir()
	unformatted := writeFile(t, dir, "a.yaml", "metadata:\n    name: a\n")
	formatted := writeFile(t, dir, "b.yaml", "metadata:\n  name: b\n")

	results, err := Run([]string{dir}, Options{})
	require.NoError(t, err)

	require.Len(t, results, 2)
	byPath := map[string]Result{}
	for _, r := range results {
		byPath[r.Path] = r
	}
	assert.True(t, byPath[unformatted].Changed)
	assert.False(t, byPath[formatted].Changed)

	got, err := os.ReadFile(unformatted)
	require.NoError(t, err)
	assert.Equal(t, "metadata:\n  name: a\n", string(got))
}

func TestRun_CheckMode_DoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "a.yaml", "metadata:\n    name: a\n")

	results, err := Run([]string{path}, Options{Check: true})
	require.NoError(t, err)

	require.Len(t, results, 1)
	assert.True(t, results[0].Changed)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "metadata:\n    name: a\n", string(got), "check mode must not modify the file")
}

func TestRun_DiffMode_ProducesUnifiedDiffWithoutWriting(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "a.yaml", "metadata:\n    name: a\n")

	results, err := Run([]string{path}, Options{Diff: true})
	require.NoError(t, err)

	require.Len(t, results, 1)
	assert.True(t, results[0].Changed)
	assert.Contains(t, results[0].Diff, "---")
	assert.Contains(t, results[0].Diff, "+++")
	assert.Contains(t, results[0].Diff, "name: a")

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "metadata:\n    name: a\n", string(got), "diff mode must not modify the file")
}

func TestRun_IncludesVarFiles(t *testing.T) {
	dir := t.TempDir()
	varFile := writeFile(t, dir, "secrets.vars.yaml", "KEY:    value\n")

	results, err := Run([]string{dir}, Options{Check: true})
	require.NoError(t, err)

	require.Len(t, results, 1)
	assert.Equal(t, varFile, results[0].Path)
	assert.True(t, results[0].Changed)
}

func TestRun_SkipsNonYAMLFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "README.md", "# hello\n")
	writeFile(t, dir, "notes.txt", "text\n")
	writeFile(t, dir, "spec.yaml", "metadata:\n  name: a\n")

	results, err := Run([]string{dir}, Options{Check: true})
	require.NoError(t, err)

	require.Len(t, results, 1)
	assert.Equal(t, filepath.Join(dir, "spec.yaml"), results[0].Path)
}

func TestRun_ExplicitFileFormattedRegardlessOfExtension(t *testing.T) {
	// A path passed explicitly is honored even without a YAML extension is NOT
	// desired; we only format YAML. But an explicit .yml file must be picked up.
	dir := t.TempDir()
	path := writeFile(t, dir, "spec.yml", "metadata:\n    name: a\n")

	results, err := Run([]string{path}, Options{Check: true})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].Changed)
}

func TestRun_InvalidYAMLReportsError(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "bad.yaml", "key: : : bad\n")

	results, err := Run([]string{path}, Options{Check: true})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Error(t, results[0].Err)
	assert.False(t, results[0].Changed)
}

func TestRun_MissingPathErrors(t *testing.T) {
	_, err := Run([]string{filepath.Join(t.TempDir(), "does-not-exist")}, Options{})
	assert.Error(t, err)
}

func TestRun_DefaultsToCurrentDirectory(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "spec.yaml", "metadata:\n    name: a\n")

	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	results, err := Run(nil, Options{Check: true})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].Changed)
}
