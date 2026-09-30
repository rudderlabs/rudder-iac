package dev

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHelpAnswersThePlayTestQuestions(t *testing.T) {
	t.Parallel()
	listen := findCommand(t, []string{"listen"})
	require.Contains(t, listen.Flags().Lookup("write-key").Usage, "reject every other key")
	require.Contains(t, findCommand(t, []string{"events", "list"}).Flags().Lookup("write-key").Usage, "Filter by")
	require.Contains(t, findCommand(t, []string{"requests", "show"}).Flags().Lookup("view").Usage, "compact or full")
}
