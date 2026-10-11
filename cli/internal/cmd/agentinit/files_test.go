package agentinit

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testWorkspaceID   = "ws-1"
	testWorkspaceName = "Acme"
)

const freshMCPJSON = `{
  "mcpServers": {
    "rudderstack": {
      "type": "http",
      "url": "https://mcp.rudderstack.com/bearer-auth-mcp",
      "headers": {
        "Authorization": "Bearer ${RUDDERSTACK_ACCESS_TOKEN}",
        "X-Workspace-ID": "ws-1"
      }
    }
  }
}
`

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
}

// stubLookPath reports the named tools as missing, or at the given path, and
// resolves every other tool from PATH.
func stubLookPath(t *testing.T, tools map[string]string) {
	t.Helper()

	original := lookPath
	lookPath = func(name string) (string, error) {
		path, ok := tools[name]
		if !ok {
			return exec.LookPath(name)
		}
		if path == "" {
			return "", exec.ErrNotFound
		}
		return path, nil
	}
	t.Cleanup(func() { lookPath = original })
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
}

func readFile(t *testing.T, dir, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	require.NoError(t, err)
	return string(data)
}

func freshActions() []fileAction {
	return []fileAction{
		{Path: ".git", Action: actionCreated, Note: "git init"},
		{Path: "rudder/", Action: actionCreated},
		{Path: "rudder/README.md", Action: actionCreated},
		{Path: ".gitignore", Action: actionCreated},
		{Path: ".mcp.json", Action: actionCreated},
		{Path: "AGENTS.md", Action: actionCreated},
		{Path: "CLAUDE.md", Action: actionCreated},
	}
}

func unchangedActions() []fileAction {
	actions := freshActions()
	for i := range actions {
		actions[i].Action, actions[i].Note = actionUnchanged, ""
	}
	actions[0].Note = "already in a git work tree"
	return actions
}

// withActions returns the fresh-directory actions with the given ones swapped
// in by path.
func withActions(overrides ...fileAction) []fileAction {
	actions := freshActions()
	for _, o := range overrides {
		for i := range actions {
			if actions[i].Path == o.Path {
				actions[i] = o
			}
		}
	}
	return actions
}

func TestPrepare(t *testing.T) {
	requireGit(t)

	block := agentsBlock(testWorkspaceID, testWorkspaceName)

	tests := []struct {
		name        string
		existing    map[string]string
		wantActions []fileAction
		wantFiles   map[string]string
	}{
		{
			name:        "fresh directory",
			wantActions: freshActions(),
			wantFiles: map[string]string{
				"rudder/README.md": specReadmeContent,
				".gitignore":       "rudder/*.vars.yaml\n",
				".mcp.json":        freshMCPJSON,
				"AGENTS.md":        block,
				"CLAUDE.md":        "@AGENTS.md\n",
			},
		},
		{
			name: "existing rudder/README.md is kept",
			existing: map[string]string{
				"rudder/README.md": "my notes\n",
			},
			wantActions: withActions(
				fileAction{Path: "rudder/", Action: actionUnchanged},
				fileAction{Path: "rudder/README.md", Action: actionUnchanged},
			),
			wantFiles: map[string]string{
				"rudder/README.md": "my notes\n",
			},
		},
		{
			name: ".gitignore without the line gets it appended",
			existing: map[string]string{
				".gitignore": "node_modules",
			},
			wantActions: withActions(
				fileAction{Path: ".gitignore", Action: actionUpdated, Note: "added rudder/*.vars.yaml"},
			),
			wantFiles: map[string]string{
				".gitignore": "node_modules\nrudder/*.vars.yaml\n",
			},
		},
		{
			name: ".gitignore with the line is left alone",
			existing: map[string]string{
				".gitignore": "node_modules\nrudder/*.vars.yaml\n",
			},
			wantActions: withActions(
				fileAction{Path: ".gitignore", Action: actionUnchanged},
			),
			wantFiles: map[string]string{
				".gitignore": "node_modules\nrudder/*.vars.yaml\n",
			},
		},
		{
			name: ".mcp.json with another server is merged in order",
			existing: map[string]string{
				".mcp.json": `{"mcpServers":{"other":{"command":"npx","args":["-y","other-mcp"]}},"someKey":"a&b"}`,
			},
			wantActions: withActions(
				fileAction{Path: ".mcp.json", Action: actionUpdated, Note: "added the rudderstack server"},
			),
			wantFiles: map[string]string{
				".mcp.json": `{
  "mcpServers": {
    "other": {
      "command": "npx",
      "args": ["-y", "other-mcp"]
    },
    "rudderstack": {
      "type": "http",
      "url": "https://mcp.rudderstack.com/bearer-auth-mcp",
      "headers": {
        "Authorization": "Bearer ${RUDDERSTACK_ACCESS_TOKEN}",
        "X-Workspace-ID": "ws-1"
      }
    }
  },
  "someKey": "a&b"
}
`,
			},
		},
		{
			name: ".mcp.json with a different rudderstack entry is replaced",
			existing: map[string]string{
				".mcp.json": `{"mcpServers":{"rudderstack":{"type":"http","url":"https://mcp.rudderstack.com/mcp"}}}`,
			},
			wantActions: withActions(
				fileAction{Path: ".mcp.json", Action: actionUpdated, Note: "replaced a different rudderstack server entry"},
			),
			wantFiles: map[string]string{
				".mcp.json": freshMCPJSON,
			},
		},
		{
			name: ".mcp.json with the same entry in another layout is left alone",
			existing: map[string]string{
				".mcp.json": `{"mcpServers": {"rudderstack": {"url": "https://mcp.rudderstack.com/bearer-auth-mcp", "type": "http",` +
					` "headers": {"X-Workspace-ID": "ws-1", "Authorization": "Bearer ${RUDDERSTACK_ACCESS_TOKEN}"}}}}`,
			},
			wantActions: withActions(
				fileAction{Path: ".mcp.json", Action: actionUnchanged},
			),
		},
		{
			name: "AGENTS.md without the block gets it appended",
			existing: map[string]string{
				"AGENTS.md": "# My app\n\nBuild with make.",
			},
			wantActions: withActions(
				fileAction{Path: "AGENTS.md", Action: actionUpdated, Note: "added the RudderStack block"},
			),
			wantFiles: map[string]string{
				"AGENTS.md": "# My app\n\nBuild with make.\n\n" + block,
			},
		},
		{
			name: "AGENTS.md with an old block has it replaced",
			existing: map[string]string{
				"AGENTS.md": "# My app\n\n<!-- rudderstack:begin -->\nold text\n<!-- rudderstack:end -->\n\n## Testing\n\nRun make test.\n",
			},
			wantActions: withActions(
				fileAction{Path: "AGENTS.md", Action: actionUpdated, Note: "replaced the RudderStack block"},
			),
			wantFiles: map[string]string{
				"AGENTS.md": "# My app\n\n" + block + "\n## Testing\n\nRun make test.\n",
			},
		},
		{
			name: "CLAUDE.md that does not mention AGENTS.md gets the import",
			existing: map[string]string{
				"CLAUDE.md": "# Notes for Claude\n",
			},
			wantActions: withActions(
				fileAction{Path: "CLAUDE.md", Action: actionUpdated, Note: "added @AGENTS.md"},
			),
			wantFiles: map[string]string{
				"CLAUDE.md": "# Notes for Claude\n@AGENTS.md\n",
			},
		},
		{
			name: "CLAUDE.md that mentions AGENTS.md is left alone",
			existing: map[string]string{
				"CLAUDE.md": "Read AGENTS.md first.\n",
			},
			wantActions: withActions(
				fileAction{Path: "CLAUDE.md", Action: actionUnchanged},
			),
			wantFiles: map[string]string{
				"CLAUDE.md": "Read AGENTS.md first.\n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, tt.existing)

			actions, err := prepare(context.Background(), dir, testWorkspaceID, testWorkspaceName)
			require.NoError(t, err)
			assert.Equal(t, tt.wantActions, actions)
			assert.DirExists(t, filepath.Join(dir, ".git"))
			for rel, want := range tt.wantFiles {
				assert.Equal(t, want, readFile(t, dir, rel), rel)
			}

			again, err := prepare(context.Background(), dir, testWorkspaceID, testWorkspaceName)
			require.NoError(t, err)
			assert.Equal(t, unchangedActions(), again, "a second run changes nothing")
		})
	}
}

func TestPrepareInsideGitWorkTree(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	require.NoError(t, exec.Command("git", "-C", root, "init").Run())
	dir := filepath.Join(root, "app")
	require.NoError(t, os.Mkdir(dir, 0o755))

	actions, err := prepare(context.Background(), dir, testWorkspaceID, testWorkspaceName)
	require.NoError(t, err)

	assert.Equal(t, fileAction{Path: ".git", Action: actionUnchanged, Note: "already in a git work tree"}, actions[0])
	assert.NoDirExists(t, filepath.Join(dir, ".git"))
}

func TestPrepareWithoutGit(t *testing.T) {
	stubLookPath(t, map[string]string{"git": ""})
	dir := t.TempDir()

	actions, err := prepare(context.Background(), dir, testWorkspaceID, testWorkspaceName)
	require.NoError(t, err)

	assert.Equal(t, withActions(fileAction{Path: ".git", Action: actionSkipped, Note: "git is not installed"}), actions)
	assert.NoDirExists(t, filepath.Join(dir, ".git"))
}

func TestPrepareRefusesFilesItCannotMerge(t *testing.T) {
	tests := []struct {
		name     string
		existing map[string]string
		wantErr  string
	}{
		{
			name:     ".mcp.json that is not JSON",
			existing: map[string]string{".mcp.json": "{not json"},
			wantErr:  "updating .mcp.json: it is not a JSON object",
		},
		{
			name:     ".mcp.json whose mcpServers is not an object",
			existing: map[string]string{".mcp.json": `{"mcpServers":[]}`},
			wantErr:  "updating .mcp.json: mcpServers is not a JSON object",
		},
		{
			name:     "AGENTS.md with a begin marker and no end marker",
			existing: map[string]string{"AGENTS.md": "<!-- rudderstack:begin -->\nhalf a block\n"},
			wantErr:  `updating AGENTS.md: found "<!-- rudderstack:begin -->" without a "<!-- rudderstack:end -->" after it`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubLookPath(t, map[string]string{"git": ""})
			dir := t.TempDir()
			writeFiles(t, dir, tt.existing)

			_, err := prepare(context.Background(), dir, testWorkspaceID, testWorkspaceName)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			for rel, content := range tt.existing {
				assert.Equal(t, content, readFile(t, dir, rel), "the file is left as it was")
			}
		})
	}
}

func TestAgentsBlock(t *testing.T) {
	block := agentsBlock(testWorkspaceID, testWorkspaceName)

	for _, want := range []string{
		"<!-- rudderstack:begin -->\n",
		"workspace 'Acme' (ws-1)",
		"`npx skills add rudderlabs/rudder-agent-skills -y -s rudder-onboarding`",
		"https://github.com/rudderlabs/rudder-agent-skills/blob/main/plugins/rudder-core/skills/rudder-onboarding/SKILL.md",
		"Pass `-l rudder` to every `rudder-cli` command",
		"`rudder-cli apply -l rudder --dry-run`",
		"`rudder-cli apply -l rudder --confirm=false`",
		"`Removed resources:`",
		"`rudder/secrets.vars.yaml`, never in specs or chat",
		"`.mcp.json`",
		"\n<!-- rudderstack:end -->\n",
	} {
		assert.Contains(t, block, want)
	}
}
