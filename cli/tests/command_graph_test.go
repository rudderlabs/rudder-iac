package tests

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// graphPayload mirrors the fields of projectgraph.Payload that consumers rely
// on. It is redeclared here rather than imported so the test fails if the
// published JSON contract changes shape, which is the thing being protected.
type graphPayload struct {
	SchemaVersion int    `json:"schemaVersion"`
	CLIVersion    string `json:"cliVersion"`
	Location      string `json:"location"`
	Nodes         []struct {
		URN         string `json:"urn"`
		ID          string `json:"id"`
		Type        string `json:"type"`
		DisplayName string `json:"displayName"`
		File        string `json:"file"`
		Line        int    `json:"line"`
		Column      int    `json:"column"`
	} `json:"nodes"`
	Edges []struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"edges"`
	Cycle       []string `json:"cycle"`
	Diagnostics []struct {
		RuleID   string `json:"ruleId"`
		Severity string `json:"severity"`
		Message  string `json:"message"`
		File     string `json:"file"`
		Line     int    `json:"line"`
		Column   int    `json:"column"`
	} `json:"diagnostics"`
}

// runGraph invokes the graph command with stdout and stderr kept apart —
// consumers pipe stdout straight into a JSON parser, so anything leaking into
// it is a break. The environment is stripped of credentials to prove the
// command needs none.
func runGraph(t *testing.T, args ...string) (graphPayload, string) {
	t.Helper()

	var stdout, stderr bytes.Buffer

	cmd := exec.Command(cliBinPath, append([]string{"graph"}, args...)...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(os.Environ(),
		"RUDDERSTACK_ACCESS_TOKEN=",
		"HOME="+t.TempDir(), // no ~/.rudder config to fall back on either
	)

	require.NoError(t, cmd.Run(), "graph must exit zero; stderr:\n%s", stderr.String())

	var payload graphPayload
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload),
		"stdout must be parseable JSON, got:\n%s", stdout.String())

	return payload, stderr.String()
}

func TestGraph_EmitsProjectWithoutCredentials(t *testing.T) {
	payload, _ := runGraph(t,
		"--location", "testdata/project/create",
		"--var-file", "testdata/project/substitution.vars.yaml",
	)

	assert.Equal(t, 1, payload.SchemaVersion)
	assert.Empty(t, payload.Diagnostics, "the fixture project is valid")
	assert.Empty(t, payload.Cycle)
	assert.NotEmpty(t, payload.Nodes)
	assert.NotEmpty(t, payload.Edges)

	byURN := map[string]bool{}
	for _, n := range payload.Nodes {
		byURN[n.URN] = true

		assert.NotEmpty(t, n.ID, "node %s has no id", n.URN)
		assert.NotEmpty(t, n.Type, "node %s has no type", n.URN)
		assert.NotEmpty(t, n.DisplayName, "node %s has no display name", n.URN)

		// Every node must be navigable, which is the whole point of the payload.
		assert.NotEmpty(t, n.File, "node %s has no source file", n.URN)
		assert.Positive(t, n.Line, "node %s has no line", n.URN)
		require.FileExists(t, n.File, "node %s points at a missing file", n.URN)
	}

	// Edges must only reference nodes the payload actually declares, or a
	// renderer ends up with dangling edges.
	for _, e := range payload.Edges {
		assert.True(t, byURN[e.From], "edge source %s is not a declared node", e.From)
		assert.True(t, byURN[e.To], "edge target %s is not a declared node", e.To)
	}
}

func TestGraph_IsDeterministicAcrossRuns(t *testing.T) {
	args := []string{"--location", "testdata/project/create", "--var-file", "testdata/project/substitution.vars.yaml"}

	first, _ := runGraph(t, args...)
	second, _ := runGraph(t, args...)

	assert.Equal(t, first, second, "repeated runs over an unchanged project must agree")
}

// The editor case: a project mid-edit still has to produce a graph, because
// that is exactly when a reader needs to see what the broken reference points
// away from.
func TestGraph_ReportsDiagnosticsAndStillReturnsGraph(t *testing.T) {
	broken := t.TempDir()

	entries, err := os.ReadDir("testdata/project/create")
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		content, err := os.ReadFile(filepath.Join("testdata/project/create", entry.Name()))
		require.NoError(t, err)

		if entry.Name() == "events.yaml" {
			content = bytes.ReplaceAll(content,
				[]byte("#/categories/app_categories/user_actions"),
				[]byte("#/categories/app_categories/ghost_category"))
		}

		require.NoError(t, os.WriteFile(filepath.Join(broken, entry.Name()), content, 0o644))
	}

	payload, _ := runGraph(t, "--location", broken, "--var-file", "testdata/project/substitution.vars.yaml")

	require.Len(t, payload.Diagnostics, 1)
	diagnostic := payload.Diagnostics[0]
	assert.Equal(t, "error", diagnostic.Severity)
	assert.Contains(t, diagnostic.Message, "ghost_category")
	assert.Equal(t, filepath.Join(broken, "events.yaml"), diagnostic.File)
	assert.Positive(t, diagnostic.Line, "a diagnostic without a position cannot be placed in an editor")

	assert.NotEmpty(t, payload.Nodes, "a failing project must still yield its graph")
}
