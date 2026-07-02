/**
 * Demo 4 — Adopt: set-external-id associates a remote source with a local
 * external id, bringing it under IaC management.
 *
 * The hidden setup seeds a genuinely UNMANAGED throwaway source via the api
 * (docs/demos/scripts/seed-unmanaged — created with no external id) and captures
 * its remote id in $RID; the demo adopts it with set-external-id, inspects it,
 * and deletes it. MUTATES a live workspace but self-cleans.
 * Render gif + mp4: bash docs/demos/record-demos.sh docs/demos/20260625-adopt
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
    'export RUDDERSTACK_CLI_EXPERIMENTAL=true',
    'export RUDDERSTACK_X_RESOURCE_COMMANDS=true', // resource verbs are experimental
    // Seed an UNMANAGED throwaway source via the api (no external id) and
    // capture its remote id in $RID.
    'export RID=$(seed-unmanaged -name "Legacy Orders Source" | sed -n "s/.*id=\\([^ ]*\\).*/\\1/p")',
  ],
  scenes: [
    { command: 'echo "=== Adopt a remote source with set-external-id ==="', sleepAfterMs: 2500 },

    { title: "An UNMANAGED source (no external id) -- created outside IaC, MANAGED=no", command: 'rudder-cli get event-stream-source -l name="Legacy Orders Source"', sleepAfterMs: 4000 },

    { title: "Adopt it: set-external-id assigns the local external id ($RID = its remote id)", command: "rudder-cli set-external-id event-stream-source $RID orders-pipeline", sleepAfterMs: 4000 },

    { title: "Now it is MANAGED -- IaC tracks it by that external id", command: 'rudder-cli get event-stream-source -l name="Legacy Orders Source"', sleepAfterMs: 4000 },

    { title: "Inspect the adopted source", command: "rudder-cli describe event-stream-source orders-pipeline", sleepAfterMs: 4500 },

    { title: "Clean up the throwaway", command: "rudder-cli delete event-stream-source orders-pipeline --confirm", sleepAfterMs: 3500 },

    { command: 'echo "=== set-external-id -- adopt remote resources into IaC ==="', sleepAfterMs: 3000 },
  ],
};

export default config;
