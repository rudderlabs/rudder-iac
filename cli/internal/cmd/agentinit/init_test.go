package agentinit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry/telemetrytest"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testWorkspace() *client.Workspace {
	dataPlaneURL := "https://acme-dataplane.rudderstack.com"
	return &client.Workspace{
		ID:           testWorkspaceID,
		Name:         testWorkspaceName,
		Environment:  "PRODUCTION",
		Status:       "ACTIVE",
		Region:       "US",
		DataPlaneURL: &dataPlaneURL,
	}
}

// fakeAPI answers GET /v2/workspace with status and body.
func fakeAPI(t *testing.T, status int, body string) string {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/workspace" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)

	return srv.URL
}

func TestResolveWorkspace(t *testing.T) {
	t.Run("returns the workspace behind the token", func(t *testing.T) {
		url := fakeAPI(t, http.StatusOK, `{"workspace":{"id":"ws-1","name":"Acme","region":"US","dataPlaneURL":"https://acme-dataplane.rudderstack.com"}}`)
		c, err := client.New("token", client.WithBaseURL(url))
		require.NoError(t, err)

		workspace, err := resolveWorkspace(context.Background(), c)
		require.NoError(t, err)

		dataPlaneURL := "https://acme-dataplane.rudderstack.com"
		assert.Equal(t, &client.Workspace{ID: "ws-1", Name: "Acme", Region: "US", DataPlaneURL: &dataPlaneURL}, workspace)
	})

	t.Run("names the fix when the token is rejected", func(t *testing.T) {
		url := fakeAPI(t, http.StatusUnauthorized, `{"message":"Unauthorized"}`)
		c, err := client.New("bad-token", client.WithBaseURL(url))
		require.NoError(t, err)

		_, err = resolveWorkspace(context.Background(), c)
		require.ErrorIs(t, err, ErrTokenRejected)
		assert.EqualError(t, err, "the RudderStack API rejected the access token (HTTP 401): create a new access token and run `rudder-cli auth login`, or set it in RUDDERSTACK_ACCESS_TOKEN")
	})

	t.Run("wraps other API failures", func(t *testing.T) {
		url := fakeAPI(t, http.StatusInternalServerError, `{"message":"boom"}`)
		c, err := client.New("token", client.WithBaseURL(url))
		require.NoError(t, err)

		_, err = resolveWorkspace(context.Background(), c)
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrTokenRejected)
		assert.Contains(t, err.Error(), "fetching workspace information: http status code: 500")
	})
}

func TestRunPrintsOneLinePerAction(t *testing.T) {
	requireGit(t)
	t.Setenv(accessTokenEnv, "set")
	dir := t.TempDir()

	var stdout, stderr bytes.Buffer
	require.NoError(t, run(context.Background(), testWorkspace(), options{location: dir}, &stdout, &stderr))
	assert.Equal(t, strings.Join([]string{
		"Workspace: Acme (ws-1)",
		"created    .git (git init)",
		"created    rudder/",
		"created    rudder/README.md",
		"created    .gitignore",
		"created    .mcp.json",
		"created    AGENTS.md",
		"created    CLAUDE.md",
		"Ready. Open your coding agent in this directory and say: set up RudderStack",
		"",
	}, "\n"), stdout.String())
	assert.Empty(t, stderr.String())

	stdout.Reset()
	require.NoError(t, run(context.Background(), testWorkspace(), options{location: dir}, &stdout, &stderr))
	assert.Equal(t, strings.Join([]string{
		"Workspace: Acme (ws-1)",
		"unchanged  .git (already in a git work tree)",
		"unchanged  rudder/",
		"unchanged  rudder/README.md",
		"unchanged  .gitignore",
		"unchanged  .mcp.json",
		"unchanged  AGENTS.md",
		"unchanged  CLAUDE.md",
		"Ready. Open your coding agent in this directory and say: set up RudderStack",
		"",
	}, "\n"), stdout.String())
}

func TestRunJSON(t *testing.T) {
	requireGit(t)
	t.Setenv(accessTokenEnv, "set")
	dir := t.TempDir()

	var stdout, stderr bytes.Buffer
	require.NoError(t, run(context.Background(), testWorkspace(), options{location: dir, json: true}, &stdout, &stderr))

	var got map[string]any
	dec := json.NewDecoder(&stdout)
	require.NoError(t, dec.Decode(&got))
	assert.False(t, dec.More(), "stdout holds one JSON object and nothing else")

	assert.Equal(t, map[string]any{
		"workspace": map[string]any{
			"id":           "ws-1",
			"name":         "Acme",
			"dataPlaneURL": "https://acme-dataplane.rudderstack.com",
			"region":       "US",
		},
		"files": []any{
			map[string]any{"path": ".git", "action": "created", "note": "git init"},
			map[string]any{"path": "rudder/", "action": "created"},
			map[string]any{"path": "rudder/README.md", "action": "created"},
			map[string]any{"path": ".gitignore", "action": "created"},
			map[string]any{"path": ".mcp.json", "action": "created"},
			map[string]any{"path": "AGENTS.md", "action": "created"},
			map[string]any{"path": "CLAUDE.md", "action": "created"},
		},
		"skills": map[string]any{"ran": false, "exitCode": nil},
		"next":   "Open your coding agent in this directory and say: set up RudderStack",
	}, got)
}

func TestRunWarnsWhenTheTokenIsNotInTheEnvironment(t *testing.T) {
	stubLookPath(t, map[string]string{"git": ""})
	t.Setenv(accessTokenEnv, "")

	var stdout, stderr bytes.Buffer
	require.NoError(t, run(context.Background(), testWorkspace(), options{location: t.TempDir()}, &stdout, &stderr))

	assert.Contains(t, stderr.String(), ".mcp.json reads the access token from RUDDERSTACK_ACCESS_TOKEN, which is not set in this shell")
	assert.Contains(t, stderr.String(), "git is not installed, so init did not run git init")
	assert.Contains(t, stdout.String(), "skipped    .git (git is not installed)")
}

func TestRunWarnsWhenTheGitCheckFails(t *testing.T) {
	stubLookPath(t, map[string]string{"git": fakeGit(t)})
	t.Setenv("FAKE_GIT_STDOUT", "")
	t.Setenv("FAKE_GIT_STDERR", "fatal: detected dubious ownership in repository at '/srv/app'\n")
	t.Setenv("FAKE_GIT_EXIT", "128")
	t.Setenv(accessTokenEnv, "set")

	var stdout, stderr bytes.Buffer
	require.NoError(t, run(context.Background(), testWorkspace(), options{location: t.TempDir()}, &stdout, &stderr))

	assert.Contains(t, stdout.String(), "skipped    .git (git rev-parse failed: fatal: detected dubious ownership in repository at '/srv/app')")
	assert.NotContains(t, stdout.String(), "created    .git\n")
	assert.Contains(t, stderr.String(), "init did not run git init (git rev-parse failed: fatal: detected dubious ownership in repository at '/srv/app')")
}

// fakeNpx writes a script that prints its arguments and working directory and
// exits with $FAKE_NPX_EXIT, so the skills step runs without npm or a network.
func fakeNpx(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake npx is a shell script")
	}

	path := filepath.Join(t.TempDir(), "npx")
	script := "#!/bin/sh\necho \"args: $*\"\necho \"dir: $(pwd -P)\"\nexit \"$FAKE_NPX_EXIT\"\n"
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	return path
}

func TestInstallSkills(t *testing.T) {
	t.Run("prints the command when npx is missing", func(t *testing.T) {
		stubLookPath(t, map[string]string{"npx": ""})
		t.Setenv(skillsSourceEnv, "")

		var out, stderr bytes.Buffer
		got := installSkills(context.Background(), t.TempDir(), defaultSkillsAgent, &out, &stderr)

		assert.Equal(t, skillsResult{}, got)
		assert.Empty(t, out.String())
		assert.Contains(t, stderr.String(), "npx -y skills add rudderlabs/rudder-agent-skills -y -a claude-code -s '*'")
	})

	for _, exitCode := range []int{0, 3} {
		t.Run(fmt.Sprintf("runs npx in the directory and reports exit code %d", exitCode), func(t *testing.T) {
			stubLookPath(t, map[string]string{"npx": fakeNpx(t)})
			t.Setenv("FAKE_NPX_EXIT", fmt.Sprint(exitCode))
			t.Setenv(skillsSourceEnv, "./unreleased-skills")
			dir := t.TempDir()
			realDir, err := filepath.EvalSymlinks(dir)
			require.NoError(t, err)

			var out, stderr bytes.Buffer
			got := installSkills(context.Background(), dir, "cursor", &out, &stderr)

			assert.Equal(t, skillsResult{Ran: true, ExitCode: &exitCode}, got)
			assert.Equal(t, "args: -y skills add ./unreleased-skills -y -a cursor -s *\ndir: "+realDir+"\n", out.String())
			if exitCode != 0 {
				assert.Contains(t, stderr.String(), "exited with code 3")
			}
		})
	}
}

func TestRunJSONWithSkillsKeepsStdoutToJSON(t *testing.T) {
	stubLookPath(t, map[string]string{"git": "", "npx": fakeNpx(t)})
	t.Setenv("FAKE_NPX_EXIT", "0")
	t.Setenv(accessTokenEnv, "set")

	var stdout, stderr bytes.Buffer
	opts := options{location: t.TempDir(), json: true, skills: true, skillsAgent: defaultSkillsAgent}
	require.NoError(t, run(context.Background(), testWorkspace(), opts, &stdout, &stderr))

	var got result
	dec := json.NewDecoder(&stdout)
	dec.DisallowUnknownFields()
	require.NoError(t, dec.Decode(&got))
	assert.False(t, dec.More())

	zero := 0
	assert.Equal(t, skillsResult{Ran: true, ExitCode: &zero}, got.Skills)
	assert.Contains(t, stderr.String(), "args: -y skills add rudderlabs/rudder-agent-skills -y -a claude-code -s *")
}

func TestInitCommand(t *testing.T) {
	t.Run("fails with the missing-token error and writes nothing", func(t *testing.T) {
		calls := telemetrytest.Record(t)
		testutils.SetConfig(t, "auth.accessToken", "")
		dir := t.TempDir()

		err := telemetrytest.Execute(NewCmdInit(), nil, []string{"-l", dir})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "RUDDERSTACK_ACCESS_TOKEN")
		assert.Equal(t, []telemetrytest.Call{{
			Command: "init",
			Errored: true,
			Extras:  []telemetry.KV{{K: "skills", V: false}, {K: "json", V: false}},
		}}, *calls)

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})

	t.Run("fails clearly on a rejected token and writes nothing", func(t *testing.T) {
		telemetrytest.Record(t)
		testutils.UseFakeAPI(t)
		testutils.SetConfig(t, "apiURL", fakeAPI(t, http.StatusUnauthorized, `{"message":"Unauthorized"}`))
		dir := t.TempDir()

		err := telemetrytest.Execute(NewCmdInit(), nil, []string{"-l", dir})
		require.ErrorIs(t, err, ErrTokenRejected)

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})

	t.Run("writes the token's workspace into .mcp.json", func(t *testing.T) {
		calls := telemetrytest.Record(t)
		testutils.UseFakeAPI(t)
		stubLookPath(t, map[string]string{"git": ""})
		dir := t.TempDir()

		cmd := NewCmdInit()
		var stdout bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&bytes.Buffer{})

		require.NoError(t, telemetrytest.Execute(cmd, nil, []string{"-l", dir, "--json"}))

		var got result
		require.NoError(t, json.Unmarshal(stdout.Bytes(), &got))
		assert.Equal(t, testutils.FakeWorkspaceID, got.Workspace.ID)
		assert.Contains(t, readFile(t, dir, ".mcp.json"), `"X-Workspace-ID": "`+testutils.FakeWorkspaceID+`"`)
		assert.NotContains(t, readFile(t, dir, ".mcp.json"), "fake-token")
		assert.Equal(t, []telemetrytest.Call{{
			Command: "init",
			Extras:  []telemetry.KV{{K: "skills", V: false}, {K: "json", V: true}},
		}}, *calls)
	})
}
