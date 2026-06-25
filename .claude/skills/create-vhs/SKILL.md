---
name: create-vhs
description: Record a rudder-cli terminal demo (GIF or MP4) using charmbracelet/vhs, driven by a typed TypeScript scenes config compiled to a VHS .tape. Use when the user wants to create a video/GIF of the rudder-cli, demo a CLI flow (import, validate, apply, destroy), or produce a recording for the README or a PR. Lean pipeline — no voiceover, no replay capture.
argument-hint: "<scenario-slug> [--description 'what to demo']"
---

# rudder-cli Demo Recording with VHS (lean, TypeScript)

Record `rudder-cli` demos with [charmbracelet/vhs](https://github.com/charmbracelet/vhs).
Each demo lives in its own dated folder under `docs/demos/`. A typed
`scenes.config.ts` is the single source of truth; a small TypeScript compiler
turns it into a VHS `.tape`, and `vhs` renders the recording.

This is a deliberately lean adaptation (no ElevenLabs voiceover, no replay-first
capture, no upload). Commands run **live** during recording.

## Pipeline

```
scenes.config.ts  --(npx tsx compile.ts)-->  tape.tape  --(vhs)-->  recording.gif|mp4
```

`compile.ts` is idempotent and instant. The only slow step is `vhs`, which runs
the commands live (≈ the on-screen duration).

## Files

| Path | Committed? | Purpose |
|---|---|---|
| `.claude/skills/create-vhs/scripts/check-prereqs.sh` | yes | Validates `vhs`, `node`, `ffmpeg`/`ttyd`, `rudder-cli` |
| `.claude/skills/create-vhs/scripts/compile.ts` | yes | `scenes.config.ts → tape.tape` (zero deps, runs via `npx tsx`) |
| `.claude/skills/create-vhs/templates/scenes.config.ts.template` | yes | Starter scenes config |
| `docs/demos/YYYYMMDD-<slug>/scenes.config.ts` | yes | **Source of truth** — output settings + scene list |
| `docs/demos/YYYYMMDD-<slug>/tape.tape` | yes | **Generated** by compile.ts; do not hand-edit |
| `docs/demos/YYYYMMDD-<slug>/recording.gif` or `.mp4` | no (gitignore) | Output of `vhs tape.tape` |

Add `docs/demos/**/recording.*` to `.gitignore` so binaries don't get committed.

## Arguments

- `<scenario-slug>`: kebab/alnum slug (`^[a-zA-Z0-9_-]+$`), e.g. `import-flow`.
  Prefix today's date → `docs/demos/YYYYMMDD-<slug>/`.
- `--description`: optional one-liner of what to demo.

## Required Up-Front Questions

Before scaffolding, ask the user via `AskUserQuestion` (do not assume):

1. **Output format** — **GIF** (README/docs embeds, no ffmpeg needed) or
   **MP4** (sharing/social; needs `ffmpeg` on PATH).
2. **Scene plan** — ordered list of scene titles + commands. Scene 1 is fixed as
   `rudder-cli --version`; ask for Scene 2 onward.
3. **Safety** — if any scene mutates remote state (`apply` without `--dry-run`,
   `destroy`), confirm the user really wants it recorded against a live
   workspace. Prefer `--dry-run` for demos.

## Flow

### Step 1 — Prereqs

```bash
bash .claude/skills/create-vhs/scripts/check-prereqs.sh
```

Stop if `vhs` or `node` is missing. `ffmpeg`/`ttyd` are warnings (ffmpeg is only
needed for `.mp4`). If `rudder-cli` isn't on PATH, build it (`make build`) and
rely on the `setup` PATH export in the config (see below).

### Step 2 — Scaffold the demo

```bash
NAME=$(date +%Y%m%d)-<slug>
mkdir -p docs/demos/$NAME
cp .claude/skills/create-vhs/templates/scenes.config.ts.template \
   docs/demos/$NAME/scenes.config.ts
```

Edit `docs/demos/$NAME/scenes.config.ts`: set `output` (`recording.gif` or
`recording.mp4`), keep Scene 1 (`rudder-cli --version`), and fill in the rest.

**rudder-cli on PATH:** the template's `setup` runs `export PATH="$PWD/bin:$PATH"`
so the repo build at `./bin/rudder-cli` is callable. This assumes you record from
the repo root. If `rudder-cli` is already installed globally, you can drop that
line. Setup commands run inside a VHS `Hide` block — invisible and zero timeline cost.

**Get explicit user approval on the scene list** before recording.

### Step 3 — Compile to a tape

```bash
npx tsx .claude/skills/create-vhs/scripts/compile.ts docs/demos/$NAME
```

Prints the scene count and a duration estimate, and writes `tape.tape`. Sanity-check
the estimate before recording.

### Step 4 — Record

```bash
export PATH="${GOPATH:-$HOME/go}/bin:$PATH"   # if vhs was installed via `go install`
vhs docs/demos/$NAME/tape.tape
```

Output lands at `docs/demos/$NAME/recording.<ext>`. Because commands run live, the
recording takes roughly its on-screen duration. Open the file to review.

### Step 5 — Iterate

| Change | Re-run |
|---|---|
| Timing / sleeps / typing speed | `compile.ts` + `vhs` |
| Add / reorder / edit a scene | edit `scenes.config.ts` → `compile.ts` + `vhs` |
| Output format (gif↔mp4) | change `output` → `compile.ts` + `vhs` |

## Narration: cards & notes (gripping walkthroughs)

To make a demo explain itself, give scenes a `card` or `note`:

- **`card`** — a full-screen intro/section/outro banner: large title + dim subtitle,
  held on screen. Use for the opening "what is this?" and the closing call-to-action.
- **`note`** — a "what's happening" header shown *above* the command: a styled
  heading + dim one-line explanation, held briefly, then the command types below it.

```ts
// Opening card
{ card: { title: "rudder-iac", subtitle: "Infra-as-Code for RudderStack", holdMs: 3500 } }

// Explain, then run
{
  note: { heading: "Step 1 — Authenticate", detail: "Log in once with an access token." },
  command: "rudder-cli auth login",
  replayFile: "01-auth.txt",
}
```

Both clear the screen first, so each step appears clean with no prior clutter.
compile.ts defines the styled `__card`/`__note` shell helpers in hidden setup and
runs each beat inside a `Hide` block (the helper's own `clear` wipes the function
call), so only the rendered card/note is recorded — never the call. A scene may
have a card or a note, a command, or both; a scene with only a `card` is a pure
title beat. `holdMs` controls the on-screen hold (card default 3000, note 2200).

A good gripping structure: **intro card → (note + command) per step → outro card.**

## Replaying captured output

Some commands can't (or shouldn't) run live during a recording:

- **Interactive / secret** — `auth login` prompts for an access token; you don't
  want to type a real token on camera.
- **One-shot** — `import workspace` succeeds once, then errors `import not
  allowed as project has changes to be synced` on re-run.
- **Non-deterministic / slow** — anything whose output drifts between runs.

For these, give the scene a `replayFile`: the command is still **typed verbatim**
on screen, but its output is printed from a captured file instead of running live.

1. Save the expected output (everything that appears *after* you press Enter —
   not the prompt or the command itself) to `docs/demos/<demo>/outputs/NN-name.txt`.
   Mask any secrets there (e.g. render a token as `***`).
2. Reference it from the scene:

```ts
{ command: "rudder-cli auth login", replayFile: "01-auth.txt", sleepAfterMs: 1800 }
```

compile.ts wires a hidden shell function that shadows the demo's binary and emits
each capture in command order; scenes of the same binary **without** a replayFile
still run live (via `command`). Nothing about the live workspace is touched.

To match a custom prompt, set it in `setup`, e.g.
`setup: ["export PS1='➜  myrepo git:(main) ✗ '"]` — compile.ts appends a trailing
`clear` so the hidden setup never shows.

## scenes.config.ts shape

```ts
interface Scene {
  card?: { title: string; subtitle?: string; holdMs?: number };   // full-screen banner
  note?: { heading: string; detail?: string; holdMs?: number };   // "what's happening" header
  title?: string;          // optional on-screen header (typed as a shell comment)
  command?: string;        // command to type and run (optional for card/note-only beats)
  typingSpeedMs?: number;  // per-scene override
  sleepBeforeMs?: number;  // pause before typing
  sleepAfterMs?: number;   // read time after output; default 3000
  replayFile?: string;     // print outputs/<file> instead of running live
}
interface DemoConfig {
  output: string;          // recording.gif | recording.mp4
  width?: number;          // default 1400
  height?: number;         // default 900
  fontSize?: number;
  theme?: string;          // any VHS theme, e.g. "Dracula"
  shell?: string;          // "bash" (default) | "zsh"
  typingSpeedMs?: number;  // default 50
  setup?: string[];        // hidden pre-roll (PATH export, cd, clear)
  scenes: Scene[];
}
export default config;
```

## VHS tape commands (in the generated tape)

| Command | Notes |
|---|---|
| `Output` | First line; recording path |
| `Set` | Tape-level config; must be at the top (mid-tape `Set` is ignored with a warning), except `Set TypingSpeed` which can change between scenes |
| `Type "text"` | Types text (backslashes and quotes are escaped by compile.ts) |
| `Enter` | Presses Enter |
| `Sleep Ns` / `Sleep Nms` | Pause. Compound durations like `4m30s` are NOT supported — compile.ts always emits plain ms |
| `Hide` / `Show` | Bracket invisible setup. **Hide blocks do not advance the recording timeline.** |

## Troubleshooting

| Problem | Fix |
|---|---|
| `could not open ttyd: navigation failed: net::ERR_CONNECTION_REFUSED` | Transient ttyd/browser startup race. **Re-run `vhs tape.tape` once** — it almost always succeeds on the second try. |
| `Expected file path after output` / `Invalid command: Users` | An unquoted `Output` path with slashes. compile.ts already quotes it; if you hand-edited the tape, wrap the path in `"..."`. |
| `vhs: command not found` | `go install github.com/charmbracelet/vhs@latest` then `export PATH="${GOPATH:-$HOME/go}/bin:$PATH"`. |
| Help/output line wraps | Raise `width` (the longest rudder-cli help line is ~130 chars; 1500 @ fontSize 18 fits). |
| Hidden `setup`/replay lines show up in the recording | `clear` must be the **last** hidden command — Hide stops frame capture but commands are still typed into the terminal. compile.ts appends the trailing `clear` automatically; don't add your own. |
| `Invalid command:` errors on a `Type` line | VHS `Type` has **no** `\"` escape. compile.ts wraps each typed string in `"`, `'`, or backticks (whichever the text lacks); don't hand-edit a tape to add `\"`. |

## Tips

- **Demos should be safe.** Use `--dry-run` for `apply`; avoid `destroy` against a
  live workspace unless the user explicitly wants it on camera.
- **Wide tables**: raise `width` (≥ 1400). **Long output**: raise `height`.
- **GIF too large**: trim `sleepAfterMs`, lower `width/height`, or switch to `.mp4`.
- **Never bake secrets into the tape.** Don't put access tokens in `setup` or
  commands; rely on `~/.rudder/config.json` or a pre-exported env var instead.

## References

- [charmbracelet/vhs README](https://github.com/charmbracelet/vhs)
- [VHS canonical demo.tape](https://github.com/charmbracelet/vhs/blob/main/examples/demo.tape)
- Adapted from the clawrium `create-vhs` skill (Python pipeline), reduced to a
  lean TypeScript recorder for rudder-cli.
