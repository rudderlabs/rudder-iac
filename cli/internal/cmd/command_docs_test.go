package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVisibleCommandsHaveCompleteDocumentation(t *testing.T) {
	var incomplete []string
	root := NewDocumentationRootCmd()
	root.InitDefaultHelpCmd()

	walkDocumentedCommands(root, func(command *cobra.Command) {
		if strings.TrimSpace(command.Short) == "" || strings.TrimSpace(command.Long) == "" || strings.TrimSpace(command.Example) == "" {
			incomplete = append(incomplete, command.CommandPath())
		}
	})

	assert.Empty(t, incomplete, "visible and conditionally visible commands must define Long and Example")
}

func TestRuntimeDefaultCommandsAreDeliberatelyClassified(t *testing.T) {
	root := NewDocumentationRootCmd()
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()

	helpCommand, _, err := root.Find([]string{"help"})
	require.NoError(t, err)
	assert.False(t, helpCommand.Hidden)

	completionCommand, _, err := root.Find([]string{"completion"})
	require.NoError(t, err)
	assert.True(t, completionCommand.Hidden)
}

func walkDocumentedCommands(command *cobra.Command, visit func(*cobra.Command)) {
	if command.Deprecated != "" {
		return
	}
	if command.Hidden && !isConditionallyVisible(command) {
		return
	}

	visit(command)
	for _, child := range command.Commands() {
		walkDocumentedCommands(child, visit)
	}
}

func isConditionallyVisible(command *cobra.Command) bool {
	return command.Name() == "debug" || command.Name() == "experimental"
}
