package dev

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHelpAnswersThePlayTestQuestions(t *testing.T) {
	t.Parallel()
	listen := findCommand(t, []string{"listen"})
	require.Contains(t, listen.Long, "If the app already targets a fixed data plane URL, start the listener on that port: --port N.")
	require.Contains(t, listen.Long, "App checklist")

	events := findCommand(t, []string{"events", "list"})
	require.Contains(t, events.Long, "seq counts every request, so gaps are control requests (sourceConfig, preflight).")
	require.Contains(t, events.Flags().Lookup("view").Usage, "requests list has the same three")
	require.Contains(t, findCommand(t, []string{"requests", "list"}).Flags().Lookup("view").Usage, "summary, compact or full")
	require.Contains(t, findCommand(t, []string{"requests", "show"}).Flags().Lookup("view").Usage, "compact or full")
}
