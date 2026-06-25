/**
 * Demo 3 — Guardrails: capability gating, loud errors, blast-radius safety,
 * adoption help, and the deprecation path. Read-only / fail-fast.
 * Render gif + mp4: bash docs/demos/record-demos.sh docs/demos/20260625-guardrails
 *
 * Narration: a typed `#` comment above each command + echo banners (ASCII only).
 */
interface Scene {
  card?: { title: string; subtitle?: string; holdMs?: number };
  note?: { heading: string; detail?: string; holdMs?: number };
  title?: string;
  command?: string;
  typingSpeedMs?: number;
  sleepBeforeMs?: number;
  sleepAfterMs?: number;
  replayFile?: string;
}
interface DemoConfig {
  output: string;
  width?: number;
  height?: number;
  fontSize?: number;
  theme?: string;
  shell?: string;
  typingSpeedMs?: number;
  setup?: string[];
  scenes: Scene[];
}

const config: DemoConfig = {
  output: process.env.DEMO_OUT || "recording.gif",
  width: 1500,
  height: 900,
  fontSize: 18,
  theme: "Dracula",
  typingSpeedMs: 45,
  setup: [
    'export PATH="$(git rev-parse --show-toplevel)/bin:$PATH"',
    'export PS1="$ "',
  ],
  scenes: [
    { command: 'echo "=== Guardrails: capability-gated verbs, loud errors, safe blast radius ==="', sleepAfterMs: 2500 },

    { title: "Verbs are capability-gated -- accounts are read-only, mutating verbs are refused", command: "rudder-cli set-external-id account acct_123 my-account", sleepAfterMs: 4000 },

    { title: "Unknown types fail loud -- the error lists every valid type", command: "rudder-cli get widget", sleepAfterMs: 4500 },

    { title: "Scoped vs full reconcile -- -f never deletes; --location can prune", command: "rudder-cli apply --help", sleepAfterMs: 5500 },

    { title: "Adopt existing resources -- set-external-id brings a remote resource under IaC", command: "rudder-cli set-external-id --help", sleepAfterMs: 4000 },

    { title: "Old per-noun listers now point you at get", command: "rudder-cli workspace accounts list", sleepAfterMs: 4000 },

    { command: 'echo "=== safe by design -- github.com/rudderlabs/rudder-iac ==="', sleepAfterMs: 3000 },
  ],
};

export default config;
