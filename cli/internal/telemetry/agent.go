package telemetry

import (
	"os"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
)

// AgentEnvVar names the coding agent, or the tool driving the CLI, when the
// caller wants to say so itself. It wins over detection.
const AgentEnvVar = "RUDDERSTACK_CLI_AGENT"

// agentEnv lists a variable each coding agent sets in the shells it starts.
// Order matters only when two are set: the first match names the agent.
// Variables a person's own terminal also carries (CURSOR_TRACE_ID in Cursor's
// terminal, REPL_ID in Replit, TERM_PROGRAM) are left out, so a person typing
// a command is not counted as an agent.
var agentEnv = []struct{ env, name string }{
	{"CLAUDECODE", "claude-code"},
	{"CURSOR_AGENT", "cursor"},
	{"CODEX_SANDBOX", "codex"},
	{"CODEX_THREAD_ID", "codex"},
	{"COPILOT_MODEL", "copilot"},
	{"COPILOT_ALLOW_ALL", "copilot"},
	{"GEMINI_CLI", "gemini"},
	{"OPENCODE", "opencode"},
	{"AUGMENT_AGENT", "augment"},
	{"ANTIGRAVITY_AGENT", "antigravity"},
}

// Agent returns the coding agent running the CLI, or "" when a person or a
// script it cannot name is. AI_AGENT is the variable several agents and tools
// have agreed on, so it is read after ours and before the per-agent ones.
func Agent() string {
	for _, env := range []string{AgentEnvVar, "AI_AGENT"} {
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

// agentName keeps a caller-supplied name to lower-case letters, digits, '.',
// '_' and '-', at most 32 of them. The name goes into an HTTP header and a
// warehouse column, and neither should carry whatever the environment held.
func agentName(raw string) string {
	name := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			return r
		}
		return -1
	}, strings.ToLower(raw))

	return name[:min(len(name), 32)]
}

// UserAgent is the User-Agent of every API call. With telemetry on it names
// the agent and carries the telemetry id, which is the only thing that ties a
// command event to the workspace the API saw. With telemetry off it is the
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
	if id := agentName(cfg.Telemetry.AnonymousID); id != "" {
		parts = append(parts, "telemetry-id="+id)
	}
	if len(parts) == 0 {
		return ua
	}

	return ua + " (" + strings.Join(parts, "; ") + ")"
}
