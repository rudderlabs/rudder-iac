/**
 * Demo 2 — Build: the full managed lifecycle of one resource, all scoped.
 * MUTATES a live workspace: creates, updates, then DELETES a throwaway
 * `demo-orders-source` (and only that). Self-cleaning.
 * Render gif + mp4: bash docs/demos/record-demos.sh docs/demos/20260625-build
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
  height: 950,
  fontSize: 18,
  theme: "Dracula",
  typingSpeedMs: 45,
  setup: [
    'export PATH="$(git rev-parse --show-toplevel)/bin:$PATH"',
    'export PS1="$ "',
    'cd "$(git rev-parse --show-toplevel)/docs/demos/20260625-build/catalog"',
  ],
  scenes: [
    { command: 'echo "=== Build: create -> inspect -> update -> delete, always scoped ==="', sleepAfterMs: 2500 },

    { title: "Author a spec -- one declarative file describes the source", command: "cat orders.yaml", sleepAfterMs: 3500 },

    { title: "Preview first (safe) -- --dry-run shows the plan, applies nothing", command: "rudder-cli apply -f orders.yaml --dry-run --confirm=false", sleepAfterMs: 4000 },

    { title: "Apply -- scoped: -f applies ONLY this file, never deletes anything else", command: "rudder-cli apply -f orders.yaml --confirm=false", sleepAfterMs: 4500 },

    { title: "Now it is managed (MANAGED=yes); upstream-only sources are untouched", command: "rudder-cli get event-stream-source", sleepAfterMs: 4000 },

    { title: "Round-trip: get -o yaml emits a re-appliable spec (apply -f of it is a no-op)", command: "rudder-cli get event-stream-source demo-orders-source -o yaml", sleepAfterMs: 5000 },

    { title: "Describe -- a human-readable layout of the same spec", command: "rudder-cli describe event-stream-source demo-orders-source", sleepAfterMs: 4500 },

    { title: "Update one field -- re-apply the edited spec for a precise, scoped diff", command: "rudder-cli apply -f orders-v2.yaml --confirm=false", sleepAfterMs: 4500 },

    { title: "Delete -- imperative remote delete of a managed resource", command: "rudder-cli delete event-stream-source demo-orders-source --confirm", sleepAfterMs: 3500 },

    { title: "Workspace restored -- back to exactly what we started with", command: "rudder-cli get event-stream-source", sleepAfterMs: 3500 },

    { command: 'echo "=== create . get . describe . apply -f . delete -- all scoped ==="', sleepAfterMs: 3000 },
  ],
};

export default config;
