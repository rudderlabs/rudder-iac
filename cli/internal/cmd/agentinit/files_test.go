package agentinit

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
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

	sameFileAgain := unchangedActions()
	sameFileAgain[len(sameFileAgain)-1].Note = "same file as AGENTS.md"

	tests := []struct {
		name     string
		existing map[string]string
		// symlinks maps a link to its target, both relative to the directory.
		symlinks    map[string]string
		wantActions []fileAction
		wantFiles   map[string]string
		// wantAgain is the second run's actions, unchangedActions() when nil.
		wantAgain []fileAction
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
			name: `.mcp.json with "mcpServers": null gets the server`,
			existing: map[string]string{
				".mcp.json": `{"mcpServers": null}`,
			},
			wantActions: withActions(
				fileAction{Path: ".mcp.json", Action: actionUpdated, Note: "added the rudderstack server"},
			),
			wantFiles: map[string]string{
				".mcp.json": freshMCPJSON,
			},
		},
		{
			name: "empty .mcp.json gets the server",
			existing: map[string]string{
				".mcp.json": "",
			},
			wantActions: withActions(
				fileAction{Path: ".mcp.json", Action: actionUpdated, Note: "added the rudderstack server"},
			),
			wantFiles: map[string]string{
				".mcp.json": freshMCPJSON,
			},
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
		{
			name: "CLAUDE.md that is a symlink to AGENTS.md gets no self-import",
			existing: map[string]string{
				"AGENTS.md": "# My app\n",
			},
			symlinks: map[string]string{
				"CLAUDE.md": "AGENTS.md",
			},
			wantActions: withActions(
				fileAction{Path: "AGENTS.md", Action: actionUpdated, Note: "added the RudderStack block"},
				fileAction{Path: "CLAUDE.md", Action: actionUnchanged, Note: "same file as AGENTS.md"},
			),
			wantFiles: map[string]string{
				"AGENTS.md": "# My app\n\n" + block,
				"CLAUDE.md": "# My app\n\n" + block,
			},
			wantAgain: sameFileAgain,
		},
		{
			name: "AGENTS.md that is a symlink to CLAUDE.md gets no self-import",
			existing: map[string]string{
				"CLAUDE.md": "# Notes for Claude\n",
			},
			symlinks: map[string]string{
				"AGENTS.md": "CLAUDE.md",
			},
			wantActions: withActions(
				fileAction{Path: "AGENTS.md", Action: actionUpdated, Note: "added the RudderStack block"},
				fileAction{Path: "CLAUDE.md", Action: actionUnchanged, Note: "same file as AGENTS.md"},
			),
			wantFiles: map[string]string{
				"AGENTS.md": "# Notes for Claude\n\n" + block,
				"CLAUDE.md": "# Notes for Claude\n\n" + block,
			},
			wantAgain: sameFileAgain,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, tt.existing)
			for link, target := range tt.symlinks {
				require.NoError(t, os.Symlink(target, filepath.Join(dir, link)))
			}

			actions, err := prepare(context.Background(), dir, testWorkspaceID, testWorkspaceName)
			require.NoError(t, err)
			assert.Equal(t, tt.wantActions, actions)
			assert.DirExists(t, filepath.Join(dir, ".git"))
			for rel, want := range tt.wantFiles {
				assert.Equal(t, want, readFile(t, dir, rel), rel)
			}
			for link := range tt.symlinks {
				info, err := os.Lstat(filepath.Join(dir, link))
				require.NoError(t, err)
				assert.Equal(t, os.ModeSymlink, info.Mode().Type(), "%s is still a symlink", link)
			}

			wantAgain := tt.wantAgain
			if wantAgain == nil {
				wantAgain = unchangedActions()
			}
			again, err := prepare(context.Background(), dir, testWorkspaceID, testWorkspaceName)
			require.NoError(t, err)
			assert.Equal(t, wantAgain, again, "a second run changes nothing")
		})
	}
}

// fakeGit writes a git script for ensureGitRepo. rev-parse prints
// $FAKE_GIT_STDOUT and $FAKE_GIT_STDERR and exits with $FAKE_GIT_EXIT. init
// creates the .git directory.
func fakeGit(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake git is a shell script")
	}

	path := filepath.Join(t.TempDir(), "git")
	script := "#!/bin/sh\n" +
		"# Called as: git -C <dir> <command> ...\n" +
		"if [ \"$3\" = init ]; then mkdir \"$2/.git\"; exit 0; fi\n" +
		"printf '%s' \"$FAKE_GIT_STDOUT\"\n" +
		"printf '%s' \"$FAKE_GIT_STDERR\" >&2\n" +
		"exit \"$FAKE_GIT_EXIT\"\n"
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	return path
}

func TestEnsureGitRepo(t *testing.T) {
	tests := []struct {
		name     string
		stdout   string
		stderr   string
		exitCode int
		want     fileAction
		wantInit bool
	}{
		{
			name:     "not a git repository runs git init",
			stderr:   "fatal: not a git repository (or any of the parent directories): .git\n",
			exitCode: 128,
			want:     fileAction{Path: ".git", Action: actionCreated, Note: "git init"},
			wantInit: true,
		},
		{
			name:   "inside a work tree is left alone",
			stdout: "true\n",
			want:   fileAction{Path: ".git", Action: actionUnchanged, Note: "already in a git work tree"},
		},
		{
			name:     "a repository with dubious ownership is skipped",
			stderr:   "fatal: detected dubious ownership in repository at '/srv/app'\nTo add an exception for this directory, call:\n",
			exitCode: 128,
			want: fileAction{Path: ".git", Action: actionSkipped,
				Note: "git rev-parse failed: fatal: detected dubious ownership in repository at '/srv/app'"},
		},
		{
			name:     "another exit code is skipped",
			stderr:   "fatal: not a git repository\n",
			exitCode: 1,
			want:     fileAction{Path: ".git", Action: actionSkipped, Note: "git rev-parse failed: fatal: not a git repository"},
		},
		{
			name:     "a failure with no message is skipped",
			exitCode: 128,
			want:     fileAction{Path: ".git", Action: actionSkipped, Note: "git rev-parse failed: exit status 128"},
		},
		{
			name:   "inside a git directory but not a work tree is skipped",
			stdout: "false\n",
			want:   fileAction{Path: ".git", Action: actionSkipped, Note: "in a git directory but not in a work tree"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubLookPath(t, map[string]string{"git": fakeGit(t)})
			t.Setenv("FAKE_GIT_STDOUT", tt.stdout)
			t.Setenv("FAKE_GIT_STDERR", tt.stderr)
			t.Setenv("FAKE_GIT_EXIT", strconv.Itoa(tt.exitCode))
			dir := t.TempDir()

			got, err := ensureGitRepo(context.Background(), dir)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			if tt.wantInit {
				assert.DirExists(t, filepath.Join(dir, ".git"))
			} else {
				assert.NoDirExists(t, filepath.Join(dir, ".git"), "git init did not run")
			}
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
