package dev

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

// cliOnlyFlags are the client flags with no query parameter behind them.
var cliOnlyFlags = []string{"url", "timeout", "json", "jq", "help"}

// routeCommands maps each query route to the command path that calls it.
var routeCommands = map[string][]string{
	"events":         {"events", "list"},
	"requests":       {"requests", "list"},
	"requests/{seq}": {"requests", "show"},
	"summary":        {"summary"},
}

func kebab(name string) string {
	return strings.ToLower(regexp.MustCompile(`([a-z])([A-Z])`).ReplaceAllString(name, "$1-$2"))
}

func camel(flag string) string {
	parts := strings.Split(flag, "-")
	for i := 1; i < len(parts); i++ {
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, "")
}

func findCommand(t *testing.T, path []string) *cobra.Command {
	t.Helper()
	cmd, _, err := NewCmdDev(testDeps(t)).Find(path)
	require.NoError(t, err)
	require.Equal(t, path[len(path)-1], cmd.Name())
	return cmd
}

// flagDefault is the parameter default as the flag shows it in --help.
func expectedDefValue(p devlisten.Parameter, flag *pflag.Flag) string {
	switch {
	case p.Repeatable:
		return "[]"
	case flag.Value.Type() == "bool":
		return "false"
	}
	return p.Default
}

func TestEveryParameterHasAFlagWithTheSameDefault(t *testing.T) {
	t.Parallel()

	for route, params := range devlisten.Parameters() {
		path, ok := routeCommands[route]
		if !ok {
			continue
		}
		cmd := findCommand(t, path)
		for _, p := range params {
			flag := cmd.Flags().Lookup(kebab(p.Name))
			require.NotNil(t, flag, "%s: parameter %s has no flag --%s", route, p.Name, kebab(p.Name))
			require.Equal(t, expectedDefValue(p, flag), flag.DefValue, "%s: --%s default", route, flag.Name)
			if p.Repeatable {
				require.Equal(t, "stringArray", flag.Value.Type(), "%s: --%s is a repeated flag", route, flag.Name)
			}
		}
	}
}

func TestEveryFlagIsAParameterOrCLIOnly(t *testing.T) {
	t.Parallel()
	params := devlisten.Parameters()

	for route, path := range routeCommands {
		cmd := findCommand(t, path)
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if slices.Contains(cliOnlyFlags, f.Name) {
				return
			}
			found := slices.ContainsFunc(params[route], func(p devlisten.Parameter) bool { return p.Name == camel(f.Name) })
			require.True(t, found, "%s: flag --%s has no parameter %s", route, f.Name, camel(f.Name))
		})
	}
}

func TestFlagsReachTheWireOnlyWhenSet(t *testing.T) {
	t.Parallel()
	var (
		mu    sync.Mutex
		query []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		query = append(query, r.URL.RawQuery)
		mu.Unlock()
		_, _ = w.Write([]byte(`{"events":[],"requests":[]}`))
	}))
	t.Cleanup(srv.Close)

	for _, args := range [][]string{
		{"events", "list", "--since", "3", "--limit", "5", "--view", "summary", "--include", "context", "--status-code", "200"},
		{"events", "list"},
		{"requests", "list", "--failed=false", "--max-bytes", "0", "--route", "/v1/track", "--route", "/v1/page"},
		{"requests", "list", "--failed"},
		{"requests", "list", "--view", "summary"},
	} {
		_, _, err := runDev(t, append(args, "--url", srv.URL, "--json")...)
		require.NoError(t, err, args)
	}

	require.Equal(t, []string{
		"include=context&limit=5&since=3&statusCode=200&view=summary",
		"",
		"failed=false&maxBytes=0&route=%2Fv1%2Ftrack&route=%2Fv1%2Fpage",
		"failed=true",
		"view=summary",
	}, query)
}
