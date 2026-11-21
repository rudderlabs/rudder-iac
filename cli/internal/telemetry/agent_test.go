package telemetry

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// clearAgentEnv empties every variable Agent reads, so the tests pass the same
// way from a coding agent's shell as from a person's.
func clearAgentEnv(t *testing.T) {
	t.Helper()
	t.Setenv(AgentEnvVar, "")
	t.Setenv("AI_AGENT", "")
	t.Setenv(SkillEnvVar, "")
	for _, a := range agentEnv {
		t.Setenv(a.env, "")
	}
}

func TestAgent(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"nothing set", nil, ""},
		{"claude code", map[string]string{"CLAUDECODE": "1"}, "claude-code"},
		{"cursor agent", map[string]string{"CURSOR_AGENT": "1"}, "cursor"},
		{"codex", map[string]string{"CODEX_THREAD_ID": "abc"}, "codex"},
		{"copilot", map[string]string{"COPILOT_MODEL": "gpt"}, "copilot"},
		{"a person in Cursor's terminal", map[string]string{"CURSOR_TRACE_ID": "abc"}, ""},
		{"AI_AGENT names it", map[string]string{"AI_AGENT": "Amp"}, "amp"},
		{"our variable wins", map[string]string{AgentEnvVar: "rudder-setup", "AI_AGENT": "amp", "CLAUDECODE": "1"}, "rudder-setup"},
		{"an unusable name falls through", map[string]string{AgentEnvVar: " ;()", "CLAUDECODE": "1"}, "claude-code"},
		{"a name is cleaned and cut", map[string]string{AgentEnvVar: "My Agent\r\nX-Evil: 1 aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, "myagentx-evil1aaaaaaaaaaaaaaaaaa"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearAgentEnv(t)
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			assert.Equal(t, tc.want, Agent())
		})
	}
}

func TestUserAgent(t *testing.T) {
	set := func(t *testing.T, disabled bool, id string) {
		t.Helper()
		viper.Set("telemetry.disabled", disabled)
		viper.Set("telemetry.anonymousId", id)
		t.Cleanup(func() {
			viper.Set("telemetry.disabled", nil)
			viper.Set("telemetry.anonymousId", nil)
		})
	}

	t.Run("agent and telemetry id", func(t *testing.T) {
		clearAgentEnv(t)
		t.Setenv("CLAUDECODE", "1")
		set(t, false, "0b0e7a1c-id")
		assert.Equal(t, "rudder-cli/1.2.3 (agent=claude-code; telemetry-id=0b0e7a1c-id)", UserAgent("1.2.3"))
	})

	t.Run("agent, skill and telemetry id", func(t *testing.T) {
		clearAgentEnv(t)
		t.Setenv("CLAUDECODE", "1")
		t.Setenv(SkillEnvVar, "Rudder-Setup")
		set(t, false, "0b0e7a1c-id")
		assert.Equal(t, "rudder-setup", Skill())
		assert.Equal(t, "rudder-cli/1.2.3 (agent=claude-code; skill=rudder-setup; telemetry-id=0b0e7a1c-id)", UserAgent("1.2.3"))
	})

	t.Run("a person, telemetry on", func(t *testing.T) {
		clearAgentEnv(t)
		set(t, false, "0b0e7a1c-id")
		assert.Equal(t, "rudder-cli/1.2.3 (telemetry-id=0b0e7a1c-id)", UserAgent("1.2.3"))
	})

	t.Run("telemetry off sends the version alone", func(t *testing.T) {
		clearAgentEnv(t)
		t.Setenv("CLAUDECODE", "1")
		set(t, true, "0b0e7a1c-id")
		assert.Equal(t, "rudder-cli/1.2.3", UserAgent("1.2.3"))
	})

	t.Run("no id yet and no agent", func(t *testing.T) {
		clearAgentEnv(t)
		set(t, false, "")
		assert.Equal(t, "rudder-cli/1.2.3", UserAgent("1.2.3"))
	})
}
