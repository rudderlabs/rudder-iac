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
var cliOnlyFlags = []string{"url", "timeout", "json", "help"}

// routeCommands maps each query route to the command path that calls it
// with every parameter.
var routeCommands = map[string][]string{
	"events":         {"events", "list"},
	"requests":       {"requests", "list"},
	"requests/{seq}": {"requests", "show"},
}

// subsetCommands call a route with some of its parameters: dev events
// always asks for view=counts.
var subsetCommands = map[string][]string{
	"events": {"events"},
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

	for _, rc := range allRouteCommands() {
		route, path := rc.route, rc.path
		cmd := findCommand(t, path)
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if slices.Contains(cliOnlyFlags, f.Name) {
				return
			}
			i := slices.IndexFunc(params[route], func(p devlisten.Parameter) bool { return p.Name == camel(f.Name) })
			require.GreaterOrEqual(t, i, 0, "%s: flag --%s has no parameter %s", route, f.Name, camel(f.Name))
			require.Equal(t, expectedDefValue(params[route][i], f), f.DefValue, "%v: --%s default", path, f.Name)
		})
	}
}

type routeCommand struct {
	route string
	path  []string
}

func allRouteCommands() []routeCommand {
	var out []routeCommand
	for _, m := range []map[string][]string{routeCommands, subsetCommands} {
		for route, path := range m {
			out = append(out, routeCommand{route, path})
		}
	}
	return out
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
		{"events", "list", "--since", "3", "--limit", "5", "--view", "compact", "--status-code", "200"},
		{"events", "list"},
		{"events", "--event", "A", "--write-key", "k"},
		{"requests", "list", "--failed=false", "--max-bytes", "0", "--write-key", "k"},
		{"requests", "list", "--failed"},
		{"requests", "list", "--view", "full"},
	} {
		_, _, err := runDev(t, append(args, "--url", srv.URL, "--json")...)
		require.NoError(t, err, args)
	}

	require.Equal(t, []string{
		"limit=5&since=3&statusCode=200&view=compact",
		"",
		"event=A&view=counts&writeKey=k",
		"failed=false&maxBytes=0&writeKey=k",
		"failed=true",
		"view=full",
	}, query)
}
