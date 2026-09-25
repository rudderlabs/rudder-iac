package cmddocs_test

import (
	"io/fs"
	"os"
	"path/filepath"
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
		if command != root && (!command.IsAvailableCommand() || command.IsAdditionalHelpTopicCommand()) {
			return
		}
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
	assert.NotContains(t, paths, "rudder-cli help")
	_, err := os.Stat(filepath.Join(outputDir, "commands", "rudder-cli_tp.md"))
	assert.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(filepath.Join(outputDir, "commands", "rudder-cli_help.md"))
	assert.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(filepath.Join(manDir, "rudder-cli-help.1"))
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
	require.NoError(t, os.WriteFile(staleManPage, []byte(".nh\n.TH \"RUDDER-CLI-STALE\" \"1\"\n"), 0o644))

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

func TestGenerateIsDeterministic(t *testing.T) {
	t.Setenv("SOURCE_DATE_EPOCH", "946684800")
	firstDir := t.TempDir()
	secondDir := t.TempDir()

	require.NoError(t, cmddocs.Generate(preparedDocsTree(), filepath.Join(firstDir, "docs"), filepath.Join(firstDir, "man")))
	require.NoError(t, cmddocs.Generate(preparedDocsTree(), filepath.Join(secondDir, "docs"), filepath.Join(secondDir, "man")))

	assert.Equal(t, generatedTree(t, firstDir), generatedTree(t, secondDir))
	manPage, err := os.ReadFile(filepath.Join(firstDir, "man", "rudder-cli-apply.1"))
	require.NoError(t, err)
	assert.Contains(t, string(manPage), `"Jan 2000"`)
}

func generatedTree(t *testing.T, root string) map[string][]byte {
	t.Helper()
	files := make(map[string][]byte)
	require.NoError(t, filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relativePath)] = contents
		return nil
	}))
	return files
}

func preparedDocsTree() *cobra.Command {
	root := cmd.NewRootCommand(cmd.ModeDocs)
	cmd.PrepareDocsTree(root)
	return root
}

func walk(command *cobra.Command, visit func(*cobra.Command)) {
	visit(command)
	for _, child := range command.Commands() {
		walk(child, visit)
	}
}
