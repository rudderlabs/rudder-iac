package demo

import (
	"testing"
	"time"
)

// Say records a line of narration at this point in the run.
//
// Demo prose is otherwise derived from subtest names, which carry the step but
// never the reason. Call this where the reason is the point — a regression the
// step exists to prove, a behaviour that looks wrong until it is explained.
//
//	demo.Say(t, "Before DEX-917 the connection was created but never read back.")
//
// It costs one comparison when recording is off.
func Say(t *testing.T, text string) {
	t.Helper()

	if !Enabled() {
		return
	}

	now := time.Now().UTC()
	Append(Record{Kind: KindSay, Start: now, End: now, Text: text})
}
