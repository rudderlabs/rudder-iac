# CLI documentation generation — Design

**Status:** Approved design. Ready for implementation planning.
**Date:** 2026-06-25
**Repos:** `rudder-iac` (generator) + `rudder-hugo` (rendering)

---

## Summary

Auto-generate **structured** documentation for the `rudder-cli` command surface,
grounded in the live cobra command tree, and render it as **rich,
developer-friendly** pages in the `rudder-hugo` docs site.

Hard boundary (the core requirement): **the CLI emits only structured data
(YAML); rudder-hugo owns all templating and rendering.** The CLI never produces
markdown or HTML.

The design mirrors the existing **validation-rules** pipeline in rudder-hugo
(`data/rudder-cli-rules.yaml` → `validation-rules-explorer` shortcode + partials),
so it slots into conventions the docs team already maintains.

First iteration ships two things:

1. `rudder-iac`: a `make gen-cli-docs` target that produces
   `docs/generated/cli-commands.yaml` from the assembled cobra tree.
2. `rudder-hugo`: a `cli-commands-explorer` shortcode + partials + a content page
   that render a (manually copied) `data/rudder-cli-commands.yaml`, styled to fit
   the 2026 docs rebrand.

## Goals

- Command reference is **auto-generated from code** — descriptions, flags,
  args, examples, aliases, deprecation, and experimental gating all come from the
  cobra definitions, not hand-maintained prose.
- **Structured, versionable artifact** the docs framework can consume, in the
  same shape/style as the existing `rudder-cli-rules.yaml`.
- **Rich, developer-friendly rendering**: per-command sections, usage synopsis,
  flags tables, copy-friendly examples, experimental/deprecated badges,
  hierarchy (subcommands), and stable anchors.
- Clean separation: regenerating docs is a rudder-iac concern; presentation is a
  rudder-hugo concern.

## Non-goals (deferred)

- **CI automation.** The eventual target is a release-time job in rudder-iac that
  commits the artifact and opens a PR on rudder-hugo. Not built now — the
  artifact is **manually copied** for this exercise. `make gen-cli-docs` is the
  entry point.
- **No fetch script.** We do **not** mirror `fetch-rudder-cli-rules.sh` (believed
  unused in rudder-hugo). No npm fetch/update scripts.
- **No raw/overrides/merge split for now.** The generated file is copied directly
  to `data/rudder-cli-commands.yaml`. An editorial overrides+merge layer (as the
  rules pipeline has) is an easy later addition.
- Converging/redirecting the existing hand-written
  `content/dev-tools/rudder-cli/commands.md` — left untouched this iteration.
- Search-index integration beyond standard heading/anchor discovery.

## Architecture & data flow

```
rudder-iac                                             rudder-hugo (branch: docs-rebrand-2026-v2)
──────────                                             ────────────────────────────────────────
make gen-cli-docs
 └─ cli/cmd/gen-cli-docs        (standalone, mirrors gen-rule-docs)
      └─ cmd.Root()  ── walks cobra tree ──▶ cli/internal/clidocs
                                              Build(root) → Document
                                              Serialize(doc, dir)
      → docs/generated/cli-commands.yaml
                   │  (manual copy for this exercise;
                   │   later: release-time job commits + opens PR on rudder-hugo)
                   ▼
             data/rudder-cli-commands.yaml          (the copied CLI export)
             data/cli-commands-nav.yaml             (hand-authored grouping/order)
             layouts/shortcodes/cli-commands-explorer.html
               + layouts/partials/cli-commands/*.html
             content/dev-tools/rudder-cli/commands-reference.md  →  {{< cli-commands-explorer >}}
```

Each unit has one job: `clidocs.Build` turns a cobra tree into a plain data
model; `Serialize` writes it; `gen-cli-docs` is thin glue; the Hugo shortcode
turns the data into pages. Each is independently understandable and testable.

---

## Part A — rudder-iac (generator)

### A1. Expose the root command

`cli/internal/cmd/root.go` — add:

```go
// Root returns the fully-assembled root command (all subcommands are wired in
// init()). Used by tooling such as gen-cli-docs to introspect the command tree.
func Root() *cobra.Command { return rootCmd }
```

The tree is already assembled at package init, so importing the package yields a
complete tree including hidden/experimental commands.

### A2. `clidocs` package

`cli/internal/clidocs/` — pure model + serialization, no cobra-tool coupling
(mirrors `cli/internal/validation/docs`):

- `Build(root *cobra.Command) Document` — walks the tree recursively into a
  serializable model.
- `Serialize(doc Document, outputDir string) error` — writes
  `cli-commands.yaml`.
- Deterministic: subcommands and flags **sorted by name**; each command's flag
  list includes its own flags plus inherited/persistent flags.

### A3. `gen-cli-docs` tool + make target

`cli/cmd/gen-cli-docs/main.go` — standalone dev tool (mirrors
`cli/cmd/gen-rule-docs`): `--output-dir` (default `docs/generated/`), no auth, no
network. `Makefile`:

```make
gen-cli-docs: ## Generate the structured CLI command documentation artifact
	$(GO) run ./cli/cmd/gen-cli-docs --output-dir docs/generated
```

### A4. Experimental grounding via annotations

So the export carries the experimental gating **from code**, add a cobra
annotation on the gated commands:

```go
Annotations: map[string]string{"docs:experimentalFlag": "resourceCommands"},
```

on `get` / `describe` / `delete` / `set-external-id`. For the `apply` command
(which is not itself experimental — only its `-f` mode is), the `-f` **flag**
carries a flag annotation `docs:experimentalFlag=resourceCommands`.
`clidocs.Build` reads the command annotation into the command's
`experimental_flag` and the flag annotation into that flag's `experimental_flag`.

### A5. Emitted schema (`docs/generated/cli-commands.yaml`)

Same envelope style as `rules.yaml`:

```yaml
schema_version: 1
tool_metadata:
  cli_version: 0.0.0        # NOTE: no generated_at — keep committed diffs clean
commands:
  - name: get
    path: [get]             # stable id / anchor source
    short: "Get or list resources by type"
    long: "Get or list remote resources of a given type. …"   # cobra Long, verbatim
    usage: "rudder-cli get <type> [<id>] [flags]"
    aliases: []
    args: "1 to 2 args"     # human hint derived from cobra Args
    deprecated: ""          # cobra Deprecated message if any
    hidden: true            # from cobra
    experimental_flag: resourceCommands   # from annotation, if present
    examples:               # parsed from cobra Example
      - "rudder-cli get event-stream-source"
      - "rudder-cli get event-stream-source my-source -o yaml"
    flags:
      - { name: output, shorthand: o, type: string, default: table, usage: "Output format: table, yaml, or json", hidden: false }
      - { name: managed, shorthand: "",  type: bool,   default: "false", usage: "Show only managed resources", hidden: false }
      # a flag may also carry experimental_flag (e.g. apply's -f):
      # - { name: file, shorthand: f, type: stringArray, usage: "…", experimental_flag: resourceCommands }
    subcommands: []         # recursive; same shape
```

- **Tree-shaped** (nested `subcommands`) so hierarchy renders naturally.
- Carries everything the rich rendering needs (below): synopsis, long
  description, usage, args, per-flag detail, examples, aliases, deprecation,
  experimental gating, and nesting.

### A6. Testing (rudder-iac)

- Unit test `clidocs.Build` against a small fixture cobra tree: asserts
  names/flags/hidden/`experimental_flag`/nesting and deterministic ordering.
- Golden-file test on the serialized YAML for the fixture tree.
- `make gen-cli-docs` runs clean; `make lint` green.

---

## Part B — rudder-hugo (rendering)

**Base branch:** stack on `docs-rebrand-2026-v2` (PR #2161, the 2026 docs
rebrand), so the command reference inherits the new design system and components.

### B1. Data

- `data/rudder-cli-commands.yaml` — the generated artifact, copied in manually
  for this exercise. Read by the shortcode via `.Site.Data`.
- `data/cli-commands-nav.yaml` — hand-authored grouping/order (mirrors the
  existing `validation-rules-nav.yaml`). Defines the sections and their order,
  e.g. the **verb suite first**, then core, then resource-specific. This is a
  rudder-hugo editorial artifact (not part of the CLI export, so it does not
  cross the boundary). The shortcode falls back to experimental-first then
  alphabetical if a command isn't listed.

### B2. Templates (all rendering lives here)

- `layouts/shortcodes/cli-commands-explorer.html` — entry point; optional params
  to scope/order (e.g. `group="verbs"`), mirroring `validation-rules-explorer`.
- `layouts/partials/cli-commands/`:
  - `sidebar-nav.html` — command tree navigation.
  - `command-card.html` — one command: synopsis, usage, description, badges.
  - `flags-table.html` — flags (name, shorthand, type, default, description).
  - `examples.html` — copy-friendly example blocks (rebrand code component /
    tabs where useful).
  - `badges.html` — experimental (+ "enable with `resourceCommands`") and
    deprecated badges.

Templates use the **rebrand's components** (callouts/notes, tabs, code blocks
with copy, cards, drilldown sidebar) so the output matches the 2026 look.

### B3. Content page

- `content/dev-tools/rudder-cli/commands-reference.md` — front matter + intro,
  then `{{< cli-commands-explorer >}}`. **Verb suite rendered first**, then core
  commands, then resource-specific. Existing `commands.md` untouched.

### B4. "Rich, developer-friendly" — concrete requirements

The rendered reference must include, per command:

- **Synopsis + full description** (from `short` + `long`).
- **Usage line** and **args** expectation.
- **Flags table** with shorthand, type, default, and description.
- **Copy-able examples** (from cobra `Example`), in the rebrand code component.
- **Badges**: experimental (with the enable hint) and deprecated.
- **Aliases** and **"see also"** links to subcommands / related commands.
- **Stable anchors** from `path` for deep-linking and search.
- **Hierarchy**: subcommands nested and navigable via the sidebar.

### B5. Testing (rudder-hugo)

- `hugo` builds without template errors on `docs-rebrand-2026-v2` + these files.
- Manual visual check of the rendered `commands-reference` page (verb suite
  first, flags/examples/badges correct). No template unit-test framework — matches
  how the rules explorer is validated.

---

## Open decisions (resolved)

- **Generation mechanism:** a `make gen-cli-docs` target running a standalone
  tool (mirrors `gen-rule-docs`), not a public subcommand. CI deferred.
- **Cross-repo transfer:** manual copy now; release-time commit+PR bot later.
- **Timestamp:** omitted from the committed artifact (only `cli_version`) to keep
  diffs clean — a deliberate divergence from `rules.yaml`.
- **Overrides layer:** deferred; render the generated file directly for now.

## Risks / impact

- **Low, additive.** New generator + package + make target in rudder-iac (no
  runtime behavior change); new templates/content in rudder-hugo on a feature
  branch. The one small production edit is exporting `cmd.Root()` and adding
  doc annotations to already-existing commands.
- Keeping the artifact fresh is manual until the release-time CI lands (noted).
