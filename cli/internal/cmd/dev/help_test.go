package dev

import (
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten"
)

// removed are names that earlier designs had. An agent that learned them
// must get a usage error, never a silent match.
var (
	removedCommands = []string{"summary", "exec", "stop", "cursor", "info", "send", "reset", "guide", "requests"}
	removedFlags    = []string{
		"expect", "jq", "detach", "include", "summary", "seq", "state-file", "idle-exit", "quiet",
		"max-bytes", "status-code", "kind", "failed",
	}
)

func walk(c *cobra.Command, visit func(*cobra.Command)) {
	visit(c)
	for _, child := range c.Commands() {
		walk(child, visit)
	}
}

func TestDevHelpPrintsTheGuide(t *testing.T) {
	t.Parallel()

	stdout, _, err := execute("dev", "--help")

	require.NoError(t, err)
	require.True(t, strings.HasPrefix(stdout, strings.TrimRightFunc(devlisten.Guide, unicode.IsSpace)+"\n"),
		"dev --help starts with the guide that /_dev/v1/guide serves")
	require.Contains(t, stdout, "Available Commands:")
}

func TestHelpTree(t *testing.T) {
	t.Parallel()

	var paths []string
	walk(NewCmdDev(), func(c *cobra.Command) {
		paths = append(paths, c.CommandPath())
		for _, name := range append([]string{c.Name()}, c.Aliases...) {
			require.NotContains(t, removedCommands, name, c.CommandPath())
		}
		for _, name := range removedFlags {
			require.Nil(t, c.Flags().Lookup(name), "%s --%s", c.CommandPath(), name)
			require.Nil(t, c.InheritedFlags().Lookup(name), "%s --%s", c.CommandPath(), name)
		}
		if c.Name() == "dev" {
			return
		}
		require.NotEmpty(t, c.Short, c.CommandPath())
		require.NotEmpty(t, c.Long, c.CommandPath())
		require.NotEmpty(t, c.Example, c.CommandPath())
		for _, line := range strings.Split(c.Example, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			require.True(t, strings.HasPrefix(line, "$ rudder-cli "), "%s: %q", c.CommandPath(), line)
		}
	})
	require.Equal(t, []string{"dev", "dev events", "dev events list", "dev listen"}, paths)
}

// Every command line in the help and the guide runs against the real
// command tree: an existing command, existing flags and valid values.
func TestHelpCommandsParse(t *testing.T) {
	t.Parallel()

	var lines []string
	walk(NewCmdDev(), func(c *cobra.Command) {
		lines = append(lines, commandLines(c.Example)...)
		lines = append(lines, commandLines(c.Long)...)
	})
	guide := commandLines(devlisten.Guide)
	require.Greater(t, len(guide), 15, "the guide shows the commands it describes")
	lines = append(lines, guide...)

	for _, line := range lines {
		args := splitShell(line)
		if len(args) == 0 || args[0] != "dev" {
			continue
		}
		root := &cobra.Command{Use: "rudder-cli"}
		root.AddCommand(NewCmdDev())
		c, rest, err := root.Find(args)
		require.NoError(t, err, line)
		c.InitDefaultHelpFlag()
		require.NoError(t, c.ParseFlags(rest), line)
		require.Empty(t, c.Flags().Args(), "%s: no command takes arguments", line)
		checkValues(t, c.Flags(), line)
	}
}

// checkValues applies the read commands' own bounds, so an example with a
// bad --since or --wait fails here and not in a reader's shell.
func checkValues(t *testing.T, flags *pflag.FlagSet, line string) {
	t.Helper()
	if f := flags.Lookup("since"); f != nil && f.Changed && !strings.HasPrefix(f.Value.String(), "$") {
		require.NoError(t, devlisten.ParseSince(f.Value.String(), time.Now()), line)
	}
	if f := flags.Lookup("wait"); f != nil && f.Changed {
		_, err := devlisten.ParseWait(f.Value.String())
		require.NoError(t, err, line)
	}
	if f := flags.Lookup("view"); f != nil && f.Changed {
		require.Contains(t, []string{"list", "compact", "full"}, f.Value.String(), line)
		require.False(t, flags.Changed("fields"), "%s: --fields replaces --view", line)
	}
}

var (
	codeSpan   = regexp.MustCompile("`([^`]*rudder-cli [^`]*)`")
	shellBreak = "|><&;)"
)

// commandLines returns the rudder-cli command lines of a help text: whole
// lines in examples and code blocks, code spans in prose.
func commandLines(text string) []string {
	var out []string
	fenced := false
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			fenced = !fenced
			continue
		}
		candidates := []string{trimmed}
		if !fenced && !strings.HasPrefix(trimmed, "$ ") {
			candidates = nil
			for _, m := range codeSpan.FindAllStringSubmatch(line, -1) {
				candidates = append(candidates, m[1])
			}
		}
		for _, c := range candidates {
			for {
				i := strings.Index(c, "rudder-cli ")
				if i < 0 {
					break
				}
				c = c[i+len("rudder-cli "):]
				out = append(out, c)
			}
		}
	}
	return out
}

// splitShell splits a command line as a POSIX shell would, up to the first
// unquoted pipe, redirection, background or end of a subshell.
func splitShell(line string) []string {
	var (
		words   []string
		word    strings.Builder
		inWord  bool
		quote   rune
		escaped bool
	)
	for _, r := range line {
		switch {
		case escaped:
			word.WriteRune(r)
			escaped = false
		case quote != 0 && r == quote:
			quote = 0
		case quote != 0:
			word.WriteRune(r)
		case r == '\\':
			escaped, inWord = true, true
		case r == '\'' || r == '"':
			quote, inWord = r, true
		case unicode.IsSpace(r):
			if inWord {
				words, inWord = append(words, word.String()), false
				word.Reset()
			}
		case strings.ContainsRune(shellBreak, r):
			if inWord {
				words = append(words, word.String())
			}
			return words
		default:
			word.WriteRune(r)
			inWord = true
		}
	}
	if inWord {
		words = append(words, word.String())
	}
	return slices.DeleteFunc(words, func(w string) bool { return w == "" })
}

func TestSplitShell(t *testing.T) {
	t.Parallel()

	for line, want := range map[string][]string{
		`dev events --event 'Order Completed' --json | jq -e .`: {"dev", "events", "--event", "Order Completed", "--json"},
		`dev listen --port 0 > ready.json 2> listen.log &`:      {"dev", "listen", "--port", "0"},
		`dev events list --url "$url" --since "$cur"`:           {"dev", "events", "list", "--url", "$url", "--since", "$cur"},
		`dev events --json)`: {"dev", "events", "--json"},
	} {
		require.Equal(t, want, splitShell(line), line)
	}
}
