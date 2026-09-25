package cmddocs_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmddocs"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateWritesOneArtifactPerDocumentedCommand(t *testing.T) {
	root := preparedDocsTree()
	outputDir := filepath.Join(t.TempDir(), "generated")
	manDir := filepath.Join(t.TempDir(), "man")

	var paths []string
	walk(root, func(command *cobra.Command) {
		paths = append(paths, command.CommandPath())
	})

	require.NoError(t, cmddocs.Generate(root, outputDir, manDir))

	for _, commandPath := range paths {
		markdownName := strings.ReplaceAll(commandPath, " ", "_") + ".md"
		yamlName := strings.ReplaceAll(commandPath, " ", "_") + ".yaml"
		manName := strings.ReplaceAll(commandPath, " ", "-") + ".1"

		markdown, err := os.ReadFile(filepath.Join(outputDir, "commands", markdownName))
		require.NoError(t, err, commandPath)
		assert.Contains(t, string(markdown), "command: \""+commandPath+"\"")
		_, err = os.Stat(filepath.Join(outputDir, "commands", yamlName))
		assert.NoError(t, err, commandPath)
		_, err = os.Stat(filepath.Join(manDir, manName))
		assert.NoError(t, err, commandPath)
	}

	assert.Contains(t, paths, "rudder-cli completion")
	assert.Contains(t, paths, "rudder-cli completion bash")
	assert.Contains(t, paths, "rudder-cli debug")
	assert.Contains(t, paths, "rudder-cli experimental")
	assert.Contains(t, paths, "rudder-cli help")
	_, err := os.Stat(filepath.Join(outputDir, "commands", "rudder-cli_tp.md"))
	assert.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(filepath.Join(manDir, "rudder-cli-migrate.1"))
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestGeneratePreservesUnrelatedFiles(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "generated")
	commandsDir := filepath.Join(outputDir, "commands")
	manDir := filepath.Join(t.TempDir(), "man")
	require.NoError(t, os.MkdirAll(commandsDir, 0o755))
	require.NoError(t, os.MkdirAll(manDir, 0o755))

	unrelatedCommandMarkdown := filepath.Join(commandsDir, "README.md")
	unrelatedCommandYAML := filepath.Join(commandsDir, "openapi.yaml")
	unrelatedManPage := filepath.Join(manDir, "git.1")
	staleCommandMarkdown := filepath.Join(commandsDir, "rudder-cli_stale.md")
	staleCommandYAML := filepath.Join(commandsDir, "rudder-cli_stale.yaml")
	staleManPage := filepath.Join(manDir, "rudder-cli-stale.1")

	for _, file := range []string{unrelatedCommandMarkdown, unrelatedCommandYAML, unrelatedManPage} {
		require.NoError(t, os.WriteFile(file, []byte("sentinel"), 0o644))
	}
	require.NoError(t, os.WriteFile(staleCommandMarkdown, []byte("---\ncommand: \"rudder-cli stale\"\n---\n"), 0o644))
	require.NoError(t, os.WriteFile(staleCommandYAML, []byte("name: rudder-cli stale\n"), 0o644))
	require.NoError(t, os.WriteFile(staleManPage, []byte(".TH \"RUDDER-CLI-STALE\" \"1\"\n"), 0o644))

	require.NoError(t, cmddocs.Generate(preparedDocsTree(), outputDir, manDir))

	for _, file := range []string{unrelatedCommandMarkdown, unrelatedCommandYAML, unrelatedManPage} {
		contents, err := os.ReadFile(file)
		require.NoError(t, err, file)
		assert.Equal(t, "sentinel", string(contents), file)
	}
	for _, file := range []string{staleCommandMarkdown, staleCommandYAML, staleManPage} {
		_, err := os.Stat(file)
		assert.ErrorIs(t, err, os.ErrNotExist, file)
	}
}

func TestGeneratedCommandDocsAreCurrent(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "generated")
	manDir := filepath.Join(t.TempDir(), "man")
	root := preparedDocsTree()
	expected := documentedFilenames(root)

	require.NoError(t, cmddocs.Generate(root, outputDir, manDir))

	repoRoot := repositoryRoot(t)
	assertGeneratedDirectoryCurrent(t, filepath.Join(outputDir, "commands"), filepath.Join(repoRoot, "docs", "generated", "commands"), expected.commands)
	assertGeneratedDirectoryCurrent(t, manDir, filepath.Join(repoRoot, "man"), expected.man)
}

type generatedFilenames struct {
	commands map[string]struct{}
	man      map[string]struct{}
}

func documentedFilenames(root *cobra.Command) generatedFilenames {
	files := generatedFilenames{
		commands: make(map[string]struct{}),
		man:      make(map[string]struct{}),
	}
	walk(root, func(command *cobra.Command) {
		commandPath := command.CommandPath()
		files.commands[strings.ReplaceAll(commandPath, " ", "_")+".md"] = struct{}{}
		files.commands[strings.ReplaceAll(commandPath, " ", "_")+".yaml"] = struct{}{}
		files.man[strings.ReplaceAll(commandPath, " ", "-")+".1"] = struct{}{}
	})
	return files
}

func assertGeneratedDirectoryCurrent(t *testing.T, generatedDir, committedDir string, expected map[string]struct{}) {
	t.Helper()

	entries, err := os.ReadDir(committedDir)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.IsDir() || !isGeneratedArtifact(entry.Name()) {
			continue
		}
		_, ok := expected[entry.Name()]
		assert.True(t, ok, "unexpected checked-in command doc: %s", filepath.Join(committedDir, entry.Name()))
	}

	for filename := range expected {
		generated, err := os.ReadFile(filepath.Join(generatedDir, filename))
		require.NoError(t, err, filename)
		committed, err := os.ReadFile(filepath.Join(committedDir, filename))
		require.NoError(t, err, filename)
		assert.Equal(t, string(generated), string(committed), filename)
	}
}

func isGeneratedArtifact(name string) bool {
	return strings.HasPrefix(name, "rudder-cli") && (strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".1"))
}

func preparedDocsTree() *cobra.Command {
	root := cmd.NewRootCommand(cmd.ModeDocs)
	cmd.PrepareDocsTree(root)
	return root
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}

func walk(command *cobra.Command, visit func(*cobra.Command)) {
	visit(command)
	for _, child := range command.Commands() {
		walk(child, visit)
	}
}
