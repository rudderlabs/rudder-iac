package telemetry

import (
	"os"
	"strings"

	"github.com/google/uuid"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
)

const (
	// agentEnvVar names the coding agent, or the tool driving the CLI, when
	// the caller wants to say so itself. It wins over detection.
	agentEnvVar = "RUDDERSTACK_CLI_AGENT"

	// skillEnvVar names the agent skill whose instructions run the CLI, such
	// as rudder-setup. Only the skill knows this, so it is never detected.
	skillEnvVar = "RUDDERSTACK_CLI_SKILL"

	// aiAgentEnvVar is the variable several agents and tools have agreed on
	// to name themselves.
	aiAgentEnvVar = "AI_AGENT"
)

// Skill returns the skill the caller named, or "".
func Skill() string { return agentName(os.Getenv(skillEnvVar)) }

// agentEnv lists a variable each coding agent sets in the shells it starts.
// Order matters only when two are set: the first match names the agent.
// Variables a person's own terminal also carries (CURSOR_TRACE_ID in Cursor's
// terminal, REPL_ID in Replit, TERM_PROGRAM) are left out, so a person typing
// a command is not counted as an agent. Copilot CLI is left out for the same
// reason: COPILOT_MODEL and COPILOT_ALLOW_ALL are settings a person exports,
// and no variable is known that only Copilot sets.
var agentEnv = []struct{ env, name string }{
	{"CLAUDECODE", "claude-code"},
	{"CURSOR_AGENT", "cursor"},
	{"CODEX_SANDBOX", "codex"},
	{"CODEX_THREAD_ID", "codex"},
	{"GEMINI_CLI", "gemini"},
	{"OPENCODE", "opencode"},
	{"AUGMENT_AGENT", "augment"},
	{"ANTIGRAVITY_AGENT", "antigravity"},
}

// Agent returns the coding agent running the CLI, or "" when a person or a
// script it cannot name is. AI_AGENT is read after ours and before the
// per-agent variables, and its value is sent as the agent wrote it. Claude
// Code sets it to a name with its version, such as claude-code_2-1-330_agent,
// so one agent has several values. Reports group them by prefix, which keeps
// the version for whoever wants it; a name cut down here could not be put back.
func Agent() string {
	for _, env := range []string{agentEnvVar, aiAgentEnvVar} {
		if name := agentName(os.Getenv(env)); name != "" {
			return name
		}
	}

	for _, a := range agentEnv {
		if os.Getenv(a.env) != "" {
			return a.name
		}
	}

	return ""
}

// AgentProps returns the agent and the skill as properties of a command
// event, each only when it has a value.
func AgentProps() map[string]any {
	props := map[string]any{}
	if agent := Agent(); agent != "" {
		props["agent"] = agent
	}
	if skill := Skill(); skill != "" {
		props["skill"] = skill
	}
	return props
}

// agentName keeps a caller-supplied name to lower-case letters, digits, '.',
// '_' and '-', at most 32 of them, with each space turned into '-' so that
// "Claude Code" reads the same as the detected claude-code. The name goes into
// an HTTP header and a warehouse column, and neither should carry whatever the
// environment held.
func agentName(raw string) string {
	name := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			return r
		case r == ' ':
			return '-'
		}
		return -1
	}, strings.ToLower(strings.TrimSpace(raw)))

	return name[:min(len(name), 32)]
}

// UserAgent is the User-Agent of every API call. With telemetry on it names
// the agent and the skill and carries the telemetry id, which is the only thing
// that ties a command event to the workspace the API saw. With telemetry off it is the
// version alone.
func UserAgent(version string) string {
	ua := "rudder-cli/" + version

	cfg := config.GetConfig()
	if cfg.Telemetry.Disabled {
		return ua
	}

	var parts []string
	if agent := Agent(); agent != "" {
		parts = append(parts, "agent="+agent)
	}
	if skill := Skill(); skill != "" {
		parts = append(parts, "skill="+skill)
	}
	// The id is sent whole, as command events send it, or the two never
	// match. Initialise writes a UUID; anything else in the config is dropped
	// instead of being put into a header.
	if id, err := uuid.Parse(cfg.Telemetry.AnonymousID); err == nil {
		parts = append(parts, "telemetry-id="+id.String())
	}
	if len(parts) == 0 {
		return ua
	}

	return ua + " (" + strings.Join(parts, "; ") + ")"
}
