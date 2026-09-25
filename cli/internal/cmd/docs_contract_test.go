package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocsCommandTreeContract(t *testing.T) {
	root := NewRootCommand(ModeDocs)
	PrepareDocsTree(root)

	var paths []string
	walkCommands(root, func(command *cobra.Command) {
		paths = append(paths, command.CommandPath())
		assert.True(t, DocsEligible(command), command.CommandPath())
		assert.True(t, command.DisableAutoGenTag, command.CommandPath())
		assert.NotEmpty(t, strings.TrimSpace(command.Long), command.CommandPath()+" Long")
		assert.NotEmpty(t, strings.TrimSpace(command.Example), command.CommandPath()+" Example")
	})

	assert.Contains(t, paths, "rudder-cli completion")
	assert.Contains(t, paths, "rudder-cli completion bash")
	assert.Contains(t, paths, "rudder-cli debug")
	assert.Contains(t, paths, "rudder-cli experimental")
	assert.Contains(t, paths, "rudder-cli help")
	for _, path := range paths {
		assert.NotContains(t, path, "rudder-cli tp")
		assert.NotEqual(t, "rudder-cli migrate", path)
	}
}

func TestNewRootCommandBuildsFreshTrees(t *testing.T) {
	first := NewRootCommand(ModeDocs)
	second := NewRootCommand(ModeDocs)

	require.NotSame(t, first, second)
	firstDebug, _, err := first.Find([]string{"debug"})
	require.NoError(t, err)
	secondDebug, _, err := second.Find([]string{"debug"})
	require.NoError(t, err)
	assert.NotSame(t, firstDebug, secondDebug)

	firstTelemetry, _, err := first.Find([]string{"telemetry"})
	require.NoError(t, err)
	secondTelemetry, _, err := second.Find([]string{"telemetry"})
	require.NoError(t, err)
	assert.NotSame(t, firstTelemetry, secondTelemetry)
}

func TestRuntimeCommandTreeUsesDocumentedCobraDefaults(t *testing.T) {
	root := NewRootCommand(ModeRuntime)

	completion, _, err := root.Find([]string{"completion"})
	require.NoError(t, err)
	assert.False(t, completion.Hidden)
	assert.NotEmpty(t, strings.TrimSpace(completion.Long))
	assert.NotEmpty(t, strings.TrimSpace(completion.Example))

	help, _, err := root.Find([]string{"help"})
	require.NoError(t, err)
	assert.False(t, help.Hidden)
	assert.True(t, help.Runnable())
	assert.False(t, help.IsAdditionalHelpTopicCommand())
	assert.NotEmpty(t, strings.TrimSpace(help.Long))
	assert.NotEmpty(t, strings.TrimSpace(help.Example))

	for _, command := range root.Commands() {
		assert.NotEqual(t, "__help", command.Name())
	}
}

func TestDocsEligible(t *testing.T) {
	assert.True(t, DocsEligible(&cobra.Command{}))
	assert.False(t, DocsEligible(&cobra.Command{Hidden: true}))
	assert.False(t, DocsEligible(&cobra.Command{Deprecated: "use another command"}))
	assert.False(t, DocsEligible(nil))
}

func walkCommands(root *cobra.Command, visit func(*cobra.Command)) {
	visit(root)
	for _, child := range root.Commands() {
		walkCommands(child, visit)
	}
}
