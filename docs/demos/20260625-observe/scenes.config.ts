/**
 * Demo 1 — Observe: read-only tour of the kubectl-style `get` surface.
 * Render gif + mp4: bash docs/demos/record-demos.sh docs/demos/20260625-observe
 *
 * Narration is a typed `#` comment (title) above each command + echo banners,
 * ASCII only. (vhs 0.11.0 mangles the skill's card/note helper escapes/glyphs,
 * so card/note are intentionally unused here.)
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
    { command: 'echo "=== rudder-cli: kubectl-style verbs for RudderStack resources ==="', sleepAfterMs: 2500 },

    { title: "Discover the verbs: get / describe / delete / set-external-id, over every type", command: "rudder-cli get --help", sleepAfterMs: 4500 },

    { title: "List a resource type -- MANAGED tells you what IaC owns vs upstream-only", command: "rudder-cli get tracking-plan", sleepAfterMs: 4500 },

    { title: "Filter the list with a label selector (-l key=value)", command: 'rudder-cli get tracking-plan -l name="Sample ecomm"', sleepAfterMs: 4000 },

    { title: "Split managed from unmanaged with --managed / --unmanaged", command: "rudder-cli get tracking-plan --managed", sleepAfterMs: 4000 },

    { title: "Same grammar, any type: event-stream sources, transformations, ...", command: "rudder-cli get event-stream-source", sleepAfterMs: 4000 },

    { title: "Accounts are a read resource too -- newly wired into the verb layer", command: "rudder-cli get account", sleepAfterMs: 3500 },

    { command: 'echo "=== one CLI, every resource -- github.com/rudderlabs/rudder-iac ==="', sleepAfterMs: 3000 },
  ],
};

export default config;
