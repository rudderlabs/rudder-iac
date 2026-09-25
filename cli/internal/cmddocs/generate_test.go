package cmddocs

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateWritesCommandReferenceBundle(t *testing.T) {
	outputDir := t.TempDir()

	err := Generate(cmd.NewDocumentationRootCmd(), outputDir)
	require.NoError(t, err)

	markdown, err := os.ReadFile(filepath.Join(outputDir, "commands", "rudder-cli-apply.md"))
	require.NoError(t, err)
	assert.Contains(t, string(markdown), "command: rudder-cli apply")
	assert.NotContains(t, string(markdown), outputDir)
	assert.NotContains(t, string(markdown), "/root/.rudder")

	_, err = os.Stat(filepath.Join(outputDir, "commands", "rudder-cli-apply.yaml"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(outputDir, "man", "rudder-cli-apply.1"))
	require.NoError(t, err)

	helpMarkdown, err := os.ReadFile(filepath.Join(outputDir, "commands", "rudder-cli-help.md"))
	require.NoError(t, err)
	assert.Contains(t, string(helpMarkdown), "command: rudder-cli help")
	_, err = os.Stat(filepath.Join(outputDir, "commands", "rudder-cli-help.yaml"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(outputDir, "man", "rudder-cli-help.1"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(outputDir, "commands", "rudder-cli-completion.md"))
	assert.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(filepath.Join(outputDir, "man", "rudder-cli-completion.1"))
	assert.ErrorIs(t, err, os.ErrNotExist)

	_, err = os.Stat(filepath.Join(outputDir, "commands", "rudder-cli-debug.md"))
	assert.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(filepath.Join(outputDir, "commands", "rudder-cli-tp.md"))
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestGeneratedCommandReferenceIsCheckedIn(t *testing.T) {
	outputDir := t.TempDir()
	require.NoError(t, Generate(cmd.NewDocumentationRootCmd(), outputDir))

	repoRoot := repositoryRoot(t)
	assertDirectoriesMatch(t, filepath.Join(repoRoot, "commands"), filepath.Join(outputDir, "commands"))
	assertDirectoriesMatch(t, filepath.Join(repoRoot, "man"), filepath.Join(outputDir, "man"))
}

func repositoryRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}

func assertDirectoriesMatch(t *testing.T, expectedDir, actualDir string) {
	t.Helper()

	expectedFiles := readFiles(t, expectedDir)
	actualFiles := readFiles(t, actualDir)
	require.Equal(t, mapKeys(expectedFiles), mapKeys(actualFiles))

	for path, expected := range expectedFiles {
		assert.Truef(t, bytes.Equal(expected, actualFiles[path]), "generated file %s is out of date", path)
	}
}

func readFiles(t *testing.T, dir string) map[string][]byte {
	t.Helper()

	files := make(map[string][]byte)
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		require.NoError(t, err)
		if entry.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(dir, path)
		require.NoError(t, err)
		files[relPath], err = os.ReadFile(path)
		return err
	})
	require.NoError(t, err)
	return files
}

func mapKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
