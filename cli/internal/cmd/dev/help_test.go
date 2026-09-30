package dev

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/devlistentest"
)

// removedNames are commands and flags cut from phase 1; help must not
// point anyone at them.
var removedNames = []string{
	"dev summary", "dev exec", "dev stop", "dev cursor", "dev send", "dev reset", "dev info", "dev probe",
	"--expect", "--jq", "--detach", "--idle-exit", "--include", "--quiet", "--state-file", "help dev guide",
	"list list",
}

// helpTexts is every Long and Example of the dev tree, plus the guide.
func helpTexts(t *testing.T) map[string]string {
	t.Helper()
	texts := map[string]string{"guide": devlisten.Guide()}
	for _, c := range allCommands(NewCmdDev(testDeps(t))) {
		texts[c.CommandPath()+" Long"] = c.Long
		texts[c.CommandPath()+" Example"] = c.Example
		c.Flags().VisitAll(func(f *pflagFlag) { texts[c.CommandPath()+" --"+f.Name] = f.Usage })
	}
	return texts
}

func TestCmdDev_HelpNamesNoRemovedCommand(t *testing.T) {
	t.Parallel()
	for where, text := range helpTexts(t) {
		for _, name := range removedNames {
			require.NotContains(t, text, name, where)
		}
	}
}

// Every rudder-cli dev line in the help and the guide parses against the
// real command tree and its flags.
func TestCmdDev_HelpCommandsParse(t *testing.T) {
	t.Parallel()
	root := NewCmdDev(testDeps(t))
	checked := 0
	for where, text := range helpTexts(t) {
		for _, line := range commandLines(text) {
			require.NoError(t, checkCommand(root, line), "%s: %s", where, line)
			checked++
		}
	}
	require.Greater(t, checked, 20)
}

// Every next the listener sends is a runnable dev command.
func TestCmdDev_NextHintsParse(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)
	track(t, s, `{"event":"A","userId":"u1","properties":{"n":1}}`)
	track(t, s, `{"event":"A","userId":"u1","properties":{"n":2}}`)
	post(t, s.URL()+"/v1/track", `{bad`, "dev")
	post(t, s.URL()+"/v1/track", `{"event":"B","userId":"u1"}`, "")

	root := NewCmdDev(testDeps(t))
	var nexts []string
	for _, path := range []string{
		"events?view=counts", "events?view=counts&event=A&event=Missing", "events", "events?limit=1",
		"events?view=compact", "events?fields=properties", "events?view=full&event=A", "events?limit=0",
		"events?view=compact&maxBytes=100", "requests", "requests?view=compact", "requests?failed=true&limit=1",
		"requests/1", "requests/1?maxBytes=100", "requests/99", "events?sinc=1", "events?view=raw",
		"events?serverId=0000000000000000", "events?fields=nope", "events?fields=properties&view=full", "",
	} {
		nexts = append(nexts, collectNexts(t, s.URL()+"/_dev/v1/"+path)...)
	}
	require.Greater(t, len(nexts), 15)
	for _, next := range nexts {
		switch {
		case strings.HasPrefix(next, "rudder-cli "):
			require.NoError(t, checkCommand(root, next), next)
		case strings.HasPrefix(next, "curl "), !strings.Contains(next, " "):
			// a shell line for curl users, or links.next, a relative URL
		default:
			t.Errorf("next is neither a command nor a URL: %s", next)
		}
	}
}

func post(t *testing.T, url, body, key string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.SetBasicAuth(key, "")
	}
	resp, err := testHTTP.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
}

// collectNexts returns every string under a "next" key of the answer.
func collectNexts(t *testing.T, url string) []string {
	t.Helper()
	resp, err := testHTTP.Get(url) //nolint:noctx // test URL
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var doc any
	require.NoError(t, json.Unmarshal(body, &doc), url)
	var out []string
	var walk func(v any)
	walk = func(v any) {
		switch v := v.(type) {
		case map[string]any:
			for k, child := range v {
				if s, ok := child.(string); ok && k == "next" {
					out = append(out, s)
				}
				walk(child)
			}
		case []any:
			for _, child := range v {
				walk(child)
			}
		}
	}
	walk(doc)
	return out
}

// commandLines finds each "rudder-cli dev ..." command in text, up to a
// pipe, a redirect, a background & or a comment.
func commandLines(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		i := strings.Index(line, "rudder-cli dev")
		if i < 0 {
			continue
		}
		cmd := line[i:]
		for _, stop := range []string{" |", " >", " 2>", " &", " #", ")", "`"} {
			if j := strings.Index(cmd, stop); j >= 0 {
				cmd = cmd[:j]
			}
		}
		out = append(out, strings.TrimRight(strings.TrimSpace(cmd), ".,:"))
	}
	return out
}

// checkCommand parses line against the dev tree: the subcommand path, each
// flag, and the positional arguments of the command it lands on.
func checkCommand(root *cobra.Command, line string) error {
	words := shellWords(line)
	if len(words) < 2 || words[0] != "rudder-cli" || words[1] != "dev" {
		return errorf("not a dev command")
	}
	cmd, rest := walkPath(root, words[2:])
	positional, err := checkFlags(cmd, rest)
	if err != nil || cmd.Args == nil {
		return err
	}
	if err := cmd.Args(cmd, positional); err != nil {
		return errorf("%s: %v", cmd.CommandPath(), err)
	}
	return nil
}

// walkPath follows subcommand names and returns the command and the words
// after its path.
func walkPath(cmd *cobra.Command, words []string) (*cobra.Command, []string) {
	for len(words) > 0 {
		sub := findSub(cmd, words[0])
		if sub == nil {
			break
		}
		cmd, words = sub, words[1:]
	}
	return cmd, words
}

// checkFlags checks each flag exists on cmd and returns the positional
// arguments.
func checkFlags(cmd *cobra.Command, words []string) ([]string, error) {
	var positional []string
	for i := 0; i < len(words); i++ {
		w := words[i]
		if !strings.HasPrefix(w, "-") {
			positional = append(positional, w)
			continue
		}
		name, _, hasValue := strings.Cut(strings.TrimLeft(w, "-"), "=")
		f := lookupFlag(cmd, name)
		switch {
		case name == "help":
		case f == nil:
			return nil, errorf("%s has no flag %s", cmd.CommandPath(), w)
		case !hasValue && f.Value.Type() != "bool":
			i++
		}
	}
	return positional, nil
}

func lookupFlag(cmd *cobra.Command, name string) *pflag.Flag {
	if f := cmd.Flags().Lookup(name); f != nil || len(name) != 1 {
		return f
	}
	return cmd.Flags().ShorthandLookup(name)
}

func findSub(cmd *cobra.Command, name string) *cobra.Command {
	for _, c := range cmd.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

// shellWords splits a POSIX shell line on spaces outside single or double
// quotes; the quotes are removed and an empty quoted word is kept.
func shellWords(line string) []string {
	var sw splitter
	for _, r := range line {
		sw.add(r)
	}
	sw.flush()
	return sw.words
}

type splitter struct {
	words []string
	cur   []rune
	quote rune
	word  bool
}

func (s *splitter) add(r rune) {
	switch {
	case s.quote != 0 && r == s.quote:
		s.quote = 0
	case s.quote != 0:
		s.cur = append(s.cur, r)
	case r == ' ':
		s.flush()
	case r == '\'' || r == '"':
		s.quote, s.word = r, true
	default:
		s.cur, s.word = append(s.cur, r), true
	}
}

func (s *splitter) flush() {
	if s.word {
		s.words, s.cur, s.word = append(s.words, string(s.cur)), nil, false
	}
}

func TestHelpAnswersThePlayTestQuestions(t *testing.T) {
	t.Parallel()
	listen := findCommand(t, []string{"listen"})
	require.Contains(t, listen.Flags().Lookup("write-key").Usage, "reject every other key")
	require.Contains(t, findCommand(t, []string{"events", "list"}).Flags().Lookup("write-key").Usage, "Filter by")
	require.Contains(t, findCommand(t, []string{"requests", "show"}).Flags().Lookup("view").Usage, "compact or full")
	require.Equal(t, devlisten.Guide(), NewCmdDev(testDeps(t)).Long, "dev --help is the guide")
}

type pflagFlag = pflag.Flag

func errorf(format string, args ...any) error { return fmt.Errorf(format, args...) }

// The checker must fail on what the help must never say.
func TestCmdDev_CheckerRejectsRemovedForms(t *testing.T) {
	t.Parallel()
	root := NewCmdDev(testDeps(t))
	for _, line := range []string{
		"rudder-cli dev summary --since 0",
		"rudder-cli dev summary",
		"rudder-cli dev events list --expect A=1",
		"rudder-cli dev requests show",
		"rudder-cli dev events list --jq .events",
	} {
		require.Error(t, checkCommand(root, line), line)
	}
}
