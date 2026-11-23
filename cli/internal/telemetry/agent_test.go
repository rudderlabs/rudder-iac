package telemetry

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// clearAgentEnv empties every variable Agent and Skill read, so the tests pass
// the same way from a coding agent's shell as from a person's.
func clearAgentEnv(t *testing.T) {
	t.Helper()
	for _, env := range []string{agentEnvVar, aiAgentEnvVar, skillEnvVar} {
		t.Setenv(env, "")
	}
	for _, a := range agentEnv {
		t.Setenv(a.env, "")
	}
}

func TestAgent(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{name: "nothing set", want: ""},
		{
			name: "a person in Cursor's terminal",
			env:  map[string]string{"CURSOR_TRACE_ID": "abc"},
			want: "",
		},
		{
			name: "a person with Copilot settings exported",
			env:  map[string]string{"COPILOT_MODEL": "gpt", "COPILOT_ALLOW_ALL": "1"},
			want: "",
		},
		{
			name: "AI_AGENT names it",
			env:  map[string]string{aiAgentEnvVar: "Amp"},
			want: "amp",
		},
		{
			name: "AI_AGENT is kept as Claude Code writes it, version included",
			env:  map[string]string{"CLAUDECODE": "1", aiAgentEnvVar: "claude-code_2-1-330_agent"},
			want: "claude-code_2-1-330_agent",
		},
		{
			name: "our variable wins",
			env:  map[string]string{agentEnvVar: "rudder-setup", aiAgentEnvVar: "amp", "CLAUDECODE": "1"},
			want: "rudder-setup",
		},
		{
			name: "a name with a space matches the detected one",
			env:  map[string]string{agentEnvVar: " Claude Code "},
			want: "claude-code",
		},
		{
			name: "an unusable name falls through",
			env:  map[string]string{agentEnvVar: ";()", "CLAUDECODE": "1"},
			want: "claude-code",
		},
		{
			name: "a name is cleaned and cut",
			env:  map[string]string{agentEnvVar: "My Agent\r\nX-Evil: 1 aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
			want: "my-agentx-evil-1-aaaaaaaaaaaaaaa",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearAgentEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			assert.Equal(t, tt.want, Agent())
		})
	}
}

// Every row of agentEnv names its agent when its variable is the only one set.
func TestAgentFromEachVariable(t *testing.T) {
	for _, a := range agentEnv {
		t.Run(a.env, func(t *testing.T) {
			clearAgentEnv(t)
			t.Setenv(a.env, "1")
			assert.Equal(t, a.name, Agent())
		})
	}
}

func TestAgentProps(t *testing.T) {
	t.Run("a person sends neither", func(t *testing.T) {
		clearAgentEnv(t)
		assert.Empty(t, AgentProps())
	})

	t.Run("agent and skill", func(t *testing.T) {
		clearAgentEnv(t)
		t.Setenv("CLAUDECODE", "1")
		t.Setenv(skillEnvVar, "Rudder-Setup")
		assert.Equal(t, map[string]any{"agent": "claude-code", "skill": "rudder-setup"}, AgentProps())
	})
}

func TestUserAgent(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		disabled bool
		want     string
	}{
		{
			name: "agent",
			env:  map[string]string{"CLAUDECODE": "1"},
			want: "rudder-cli/1.2.3 (agent=claude-code)",
		},
		{
			name: "agent and skill",
			env:  map[string]string{"CLAUDECODE": "1", skillEnvVar: "rudder-setup"},
			want: "rudder-cli/1.2.3 (agent=claude-code; skill=rudder-setup)",
		},
		{
			name: "a person sends the version alone",
			want: "rudder-cli/1.2.3",
		},
		{
			name:     "telemetry off sends the version alone",
			env:      map[string]string{"CLAUDECODE": "1"},
			disabled: true,
			want:     "rudder-cli/1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearAgentEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			// The id must never reach the header, whatever it holds.
			setTelemetry(t, tt.disabled, uuid.New().String())
			assert.Equal(t, tt.want, UserAgent("1.2.3"))
		})
	}
}
