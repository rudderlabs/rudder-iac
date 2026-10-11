package agent

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry/telemetrytest"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type runRecorder struct {
	calls [][]string
	err   error
}

func (r *runRecorder) Run(_ context.Context, name string, args ...string) error {
	r.calls = append(r.calls, append([]string{name}, args...))
	return r.err
}

func workspaceDeps(rec *runRecorder, err error) InitDeps {
	return InitDeps{
		Workspace: func(context.Context) (*client.Workspace, error) {
			if err != nil {
				return nil, err
			}
			return &client.Workspace{ID: "ws_1", Name: "Acme prod"}, nil
		},
		Run: rec.Run,
	}
}

func TestRunInit(t *testing.T) {
	const skillsCmd = "npx skills add rudderlabs/rudder-agent-skills"

	tests := []struct {
		name          string
		opts          InitOptions
		files         map[string]string
		workspaceErr  error
		runErr        error
		expectedOut   []string
		unexpectedOut []string
		expectedRuns  [][]string
		expectedErr   string
		expectAgents  bool
		agentsContent string
	}{
		{
			name:  "writes AGENTS.md and reports workspace and project",
			opts:  InitOptions{Version: "1.2.3"},
			files: map[string]string{"package.json": `{"devDependencies":{"vite":"5","typescript":"5"}}`},
			expectedOut: []string{
				"Workspace: Acme prod",
				"Project: Node.js (package.json); frameworks: Vite; TypeScript: yes",
				"Wrote AGENTS.md",
				"Ready. Open your coding agent here and say: set up RudderStack",
			},
			unexpectedOut: []string{skillsCmd},
			expectAgents:  true,
		},
		{
			name:         "unknown ecosystem still succeeds",
			opts:         InitOptions{Version: "1.2.3"},
			expectedOut:  []string{"Project: unknown", "Ready."},
			expectAgents: true,
		},
		{
			name:         "invalid token fails before writing anything",
			opts:         InitOptions{Version: "1.2.3"},
			workspaceErr: errors.New("401 unauthorized"),
			expectedErr:  "verifying access token: 401 unauthorized",
		},
		{
			name:          "existing AGENTS.md is refused",
			opts:          InitOptions{Version: "1.2.3"},
			files:         map[string]string{"AGENTS.md": "mine"},
			expectedErr:   "AGENTS.md already exists",
			expectAgents:  true,
			agentsContent: "mine",
		},
		{
			name:         "force overwrites AGENTS.md",
			opts:         InitOptions{Version: "1.2.3", Force: true},
			files:        map[string]string{"AGENTS.md": "mine"},
			expectedOut:  []string{"Wrote AGENTS.md", "Ready."},
			expectAgents: true,
		},
		{
			name:         "skills flag prints the command and does not run it",
			opts:         InitOptions{Version: "1.2.3", PrintSkills: true},
			expectedOut:  []string{skillsCmd, "Ready."},
			expectedRuns: nil,
			expectAgents: true,
		},
		{
			name:         "install-skills flag runs the command",
			opts:         InitOptions{Version: "1.2.3", InstallSkills: true},
			expectedOut:  []string{"Ready."},
			expectedRuns: [][]string{{"npx", "skills", "add", "rudderlabs/rudder-agent-skills"}},
			expectAgents: true,
		},
		{
			name:          "failed skills install fails the command without the ready line",
			opts:          InitOptions{Version: "1.2.3", InstallSkills: true},
			runErr:        errors.New("exit status 1"),
			expectedRuns:  [][]string{{"npx", "skills", "add", "rudderlabs/rudder-agent-skills"}},
			expectedErr:   "installing skills (" + skillsCmd + "): exit status 1",
			unexpectedOut: []string{"Ready."},
			expectAgents:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tt.files {
				require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
			}
			tt.opts.Dir = dir

			rec := &runRecorder{err: tt.runErr}
			var out bytes.Buffer
			err := RunInit(t.Context(), &out, tt.opts, workspaceDeps(rec, tt.workspaceErr))

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
			for _, s := range tt.expectedOut {
				assert.Contains(t, out.String(), s)
			}
			for _, s := range tt.unexpectedOut {
				assert.NotContains(t, out.String(), s)
			}
			assert.Equal(t, tt.expectedRuns, rec.calls)

			content, readErr := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
			if !tt.expectAgents {
				assert.True(t, os.IsNotExist(readErr), "AGENTS.md must not be written")
				return
			}
			require.NoError(t, readErr)
			if tt.agentsContent != "" {
				assert.Equal(t, tt.agentsContent, string(content))
				return
			}
			assert.Contains(t, string(content), "rudder-cli "+tt.opts.Version)
			assert.NotContains(t, string(content), versionPlaceholder)
		})
	}
}

func TestRunInitWritesNothingElse(t *testing.T) {
	dir := t.TempDir()
	rec := &runRecorder{}

	require.NoError(t, RunInit(t.Context(), &bytes.Buffer{}, InitOptions{Dir: dir, Version: "dev"}, workspaceDeps(rec, nil)))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	assert.Equal(t, []string{"AGENTS.md"}, names)
}

func TestRunInitRejectsMissingDirectory(t *testing.T) {
	err := RunInit(t.Context(), &bytes.Buffer{}, InitOptions{Dir: filepath.Join(t.TempDir(), "nope")}, workspaceDeps(&runRecorder{}, nil))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading project directory")
}

// The template is the contract with the agent. These guard the kinds and
// layout it names, which the CLI rejects when they are wrong.
func TestAgentsTemplateNamesTheSpecKinds(t *testing.T) {
	for _, want := range []string{
		"kind: event-stream-source",
		"kind: destination",
		"event-stream-connections",
		"sources/<id>.yaml",
		"destinations/<id>.yaml",
		"connections.yaml",
		"--confirm=false",
		"npx skills add rudderlabs/rudder-agent-skills",
	} {
		assert.Contains(t, agentsTemplate, want)
	}

	assert.NotContains(t, agentsTemplate, "rudder-cli init")
	assert.False(t, strings.Contains(agentsTemplate, "—"), "no em dashes")
}

func TestNewCmdAgentHasOnlyInit(t *testing.T) {
	cmd := NewCmdAgent()

	var names []string
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	assert.Equal(t, []string{"init"}, names)
}

// A missing token fails through the real command wiring, reports "agent init"
// to telemetry as errored, and writes nothing.
func TestInitCommandWithoutTokenFailsAndTracks(t *testing.T) {
	testutils.SetConfig(t, "auth.accessToken", "")
	calls := telemetrytest.Record(t)

	dir := t.TempDir()
	err := telemetrytest.Execute(NewCmdInit(), []string{"agent"}, []string{"-l", dir})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "access token is required")
	assert.Equal(t, []telemetrytest.Call{{Command: "agent init", Errored: true}}, *calls)

	_, statErr := os.Stat(filepath.Join(dir, "AGENTS.md"))
	assert.True(t, os.IsNotExist(statErr))
}
