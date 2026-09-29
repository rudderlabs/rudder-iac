package lister

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/bubbles/table"
	"github.com/stretchr/testify/assert"
)

// The interactive table can only be left by pressing esc, so it is unusable
// anywhere a person is not watching: piped, redirected, in CI, or inside a
// recorded pty, where it either cannot open /dev/tty or blocks forever. This
// is the path taken there, and the class of bug is invisible to anyone running
// the command by hand — hence a test rather than a manual check.
func TestPrintStaticTable(t *testing.T) {
	t.Parallel()

	columns := []table.Column{{Title: "#"}, {Title: "ID"}, {Title: "Name"}}

	t.Run("columns are padded to their widest cell", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer
		printStaticTable(&out, columns, []table.Row{
			{"1", "3JijC8KGEOk7z65rR8aLvkS1e4W", "analytics-pg"},
			{"2", "short", "a"},
		})

		lines := bytes.Split(bytes.TrimRight(out.Bytes(), "\n"), []byte("\n"))
		assert.Len(t, lines, 3, "a header and one line per row")
		assert.Equal(t, "#  ID                           Name", string(lines[0]))
		assert.Contains(t, string(lines[1]), "analytics-pg")
		// The short row's Name still starts in the same column as the header's.
		assert.Equal(t, bytes.Index(lines[0], []byte("Name")), bytes.Index(lines[2], []byte("a")))
	})

	t.Run("no rows still prints the header", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer
		printStaticTable(&out, columns, nil)

		assert.Equal(t, "#  ID  Name\n", out.String())
	})

	t.Run("no trailing whitespace on any line", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer
		printStaticTable(&out, columns, []table.Row{{"1", "id-1", "name-1"}})

		for _, line := range bytes.Split(bytes.TrimRight(out.Bytes(), "\n"), []byte("\n")) {
			assert.Equal(t, string(bytes.TrimRight(line, " ")), string(line))
		}
	})
}

// A pty is a terminal whether or not a person is at the far end, so a session
// recorder or an automation harness passes the terminal check and the TUI
// blocks on esc. The env override is the only way to say "not interactive" in
// that case, and it is what the demo recordings rely on.
func TestNonInteractiveOverride(t *testing.T) {
	t.Setenv("RUDDERSTACK_CLI_NONINTERACTIVE", "1")
	assert.False(t, interactiveStdout(), "an explicit opt-out wins over the terminal check")

	t.Setenv("RUDDERSTACK_CLI_NONINTERACTIVE", "")
	// Not asserting the true case: under `go test` stdout is not a terminal, so
	// it is false for the other reason, and asserting it would prove nothing.
	assert.False(t, interactiveStdout())
}
