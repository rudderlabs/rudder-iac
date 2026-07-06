# K8s-style resource command structure — Design

**Status:** Approved design (final decisions A2 + B2 + apply-`-f` flag locked).
Ready for implementation planning.
**Date:** 2026-06-07
**Related (tabled):** `2026-06-07-resource-adoption-facade-TABLED.md`

---

## Summary

Add a uniform, `kubectl`-style verb layer over **all** resources exposed through
the `api` package, operating via the CLI provider/composite layer:

```
rudder-cli get <type> [<id>]          # list, or show one (live remote)
rudder-cli get <type> <id> -o yaml    # re-appliable spec (round-trips with apply -f)
rudder-cli describe <type> <id>       # templated layout of the spec (v1)
rudder-cli delete <type> <id>         # imperative remote delete
rudder-cli set-external-id <type> <remote-id> <external-id>   # primitive claim
rudder-cli apply -f <file|dir>...     # scoped, delete-free apply (new -f mode on existing apply)
```

`get` / `describe` / `delete` / `set-external-id` are **top-level** verbs
(kubectl-style). `apply -f` is a **new flag/mode on the existing top-level
`apply` command** — not a new command — because `rudder-cli apply` already exists
as the whole-project reconcile.

The read/claim/delete verbs share one resolver and one output path. `apply -f`
**reuses the existing apply pipeline** (`project.Load` → `ResourceGraph` →
`syncer`) and diverges only at one point — **source-scoping** — so it produces
creates/updates for exactly the resources you pass and never deletes anything
else.

## Motivation

`apply` produces *unwanted changes* when local spec and remote state drift apart.
This verb layer attacks that by making the spec the literal materialization of
remote state: `get … -o yaml` emits exactly what `apply -f` consumes, so
`get -o yaml | apply -f -` round-trips. Read verbs give `kubectl`'s snappy live
ergonomics; mutating verbs preserve safety via preview + confirmation.

## Goals

- One consistent grammar/UX over all registered resource types.
- Live-remote reads (`get`, `describe`, `get -o yaml`).
- Live-remote mutations: `delete` (imperative, via `LifecycleManager.Delete`) and
  `apply -f` (scoped, delete-free reconcile reusing the syncer) — both with
  preview + confirm.
- A `set-external-id` primitive backed by a new optional `ExternalIDSetter`
  capability — the one additive foundation piece that de-risks the tabled
  adoption facade.
- Maximum reuse of the existing apply pipeline for `apply -f`: diverge only at
  source-scoping, no bespoke imperative executor.

## Non-goals (explicitly deferred / out of scope)

- `edit` — deferred entirely (live-vs-spec semantics decided when picked up).
- Per-resource `import` adoption **facade** — tabled to its own conversation
  (see the TABLED doc). Only the `set-external-id` primitive + `ExternalIDSetter`
  capability land here.
- Evolving `apply` to a **safe-by-default + `--prune`** model (kubectl-style):
  tabled. v1 keeps existing `apply --location` behavior unchanged and adds `-f`
  alongside it (non-breaking). See "Decision 12".
- Nesting the verbs under a `resource` group — **rejected**; verbs are top-level
  (Decision 11). `apply -f` rides the existing `apply` command.
- Short type **aliases** (e.g. `po` for pods) — later nicety.
- Rich, context-aware `describe` — v1 `describe` is a templated layout of the
  spec; the full version is a later phase.

## Locked decisions (with rationale)

| # | Decision | Rationale |
|---|----------|-----------|
| 1 | Build over the **CLI provider/composite layer**, not raw `api/client` | The provider layer already normalizes heterogeneous SDK clients into one `Provider` interface with a kind/type registry, state, and apply. |
| 2 | Read verbs = **live remote**; `delete` = imperative remote; `apply -f` = **scoped, delete-free reconcile** reusing the syncer (Decision B2) | Reads stay snappy; `delete` is a direct lifecycle call; `apply -f` reuses the proven planner/executePlan and only changes what differs, never pruning out-of-scope resources. |
| 3 | Address resources by **external-id first, remote-id fallback** | Matches the IaC mental model; remote-id covers unmanaged resources lacking an external id. |
| 4 | **Universal, capability-gated** scope | Read works for all types; mutate/claim are gated by the optional interfaces each provider implements. No type special-cased out by construction. |
| 5 | `get` lists **all remote by default**, `--managed`/`--unmanaged` filter; degrade to managed-only where `LoadImportable` is absent | Discovery is the point of `get`; filtering is cheap; graceful degradation keeps it universal. |
| 6 | `set-external-id` ships as a **primitive verb** + new optional `ExternalIDSetter`; the `import` facade is tabled | Additive foundation now; facade lands later as pure composition, avoiding an A→C refactor. |
| 7 | `describe` v1 = **templated layout of `get -o yaml`** | Minimal, reuses the export renderer; rich describe later. |
| 8 | `apply -f` **prompts by default**; `--confirm` applies blind; `--dry-run` previews only; aligns `delete` on `--confirm` | Preview + confirm is the safety valve before mutating remote. |
| 9 | `-f` accepts **files and directories** (recursive); multiple `-f` allowed | Bulk apply ergonomics. |
| 10 | **Fresh project/provider per `apply -f`** invocation | Handlers accumulate loaded specs with no `Reset`; a fresh instance avoids cross-file/state bleed. |
| 11 | **Top-level verbs** (`get`/`describe`/`delete`/`set-external-id`), not nested under a `resource` group | Kubectl-bare ergonomics; consolidates the scattered `workspace … list` commands into one `get`. Only `apply` collided, and it rides the existing command. |
| 12 | `apply -f` is a **new flag on the existing `apply` command**, alongside (not replacing) `--location` | Non-breaking. `--location` keeps whole-project reconcile; `-f` is scoped/delete-free. Evolving to safe-by-default `+ --prune` is tabled. |
| 13 | Deprecate `workspace accounts list` / `workspace retl-sources list` / `workspace event-stream-sources list` in favor of `get <type>` | One consistent discovery verb replaces three bespoke listers. |

---

## Section A — Command surface & grammar

Four **new top-level verbs** registered on root alongside `apply`/`validate`/
`workspace`/`import`, plus a **new `-f` mode on the existing `apply` command**.
Shared grammar for the new verbs: `rudder-cli <verb> <type> [<id>] [flags]`.

```
get <type> [<id>]                      # top-level
    [--managed | --unmanaged]          # default: all
    [-o, --output table|yaml|json]     # default: table
    [-l, --selector key=value]         # reuses lister.Filters

describe <type> <id>                   # top-level; templated layout of the spec (v1)

delete <type> <id> [--confirm]         # top-level; imperative remote delete

set-external-id <type> <remote-id> <external-id>   # top-level; primitive claim

apply -f <file|dir> [-f <file|dir>]... # NEW mode on existing `apply` command
    [--dry-run]                        # preview only, never applies
    [--confirm]                        # skip the confirmation prompt (blind)
# existing, unchanged:
apply --location <dir|file>            # whole-project reconcile (prunes)
```

Grammar rules:
- `<type>` is the registry type string verbatim (`event-stream-source`,
  `tracking-plan`, `retl-source-sql-model`, …), validated against
  `CompositeProvider.SupportedTypes()`. Unknown → error listing valid types.
- `<id>` resolves external-id first, then remote-id.
- `get … -o yaml|json` emits the re-appliable spec via `Exporter.FormatForExport`
  (not raw API JSON), so it round-trips with `apply -f`.
- Capability-gated: unsupported verb for a type errors clearly (e.g. `account`
  rejects `delete`; types without a setter reject `set-external-id`).
- **`apply` flag mutual-exclusion:** `-f` and `--location` are mutually
  exclusive; passing both errors. `-f` = scoped/delete-free; `--location` =
  whole-project reconcile. Help text states the blast-radius difference loudly.

## Section B — Architecture

**Thin cobra commands over a reusable ops package.** The four new verbs are thin
command files; their logic lives in a testable, non-cobra package so it can be
unit-tested against a mock `Provider`.

- New command packages (top-level, registered in `cmd/root.go`):
  `cli/internal/cmd/get/`, `.../describe/`, `.../delete/`, `.../setexternalid/`
  — each exposes `NewCmdX() *cobra.Command`.
- `apply -f` is **not** a new package: it modifies the existing
  `cli/internal/cmd/project/apply/apply.go` to add the `-f` flag + scoped path.
- New shared logic package `cli/internal/resourceops/`: `resolver.go`,
  `reader.go`, `printer.go`, `deleter.go`, `externalid.go`. Commands call into it.

**Shared building blocks (in `resourceops`):**

1. **Type resolver** — expose a public `ProviderForType(type) (Provider, error)`
   on the composite (today `providerForType` is private, `composite.go:273`) via
   a small `TypeRouter` interface that `CompositeProvider` satisfies; reuse
   `SupportedTypes()` for validation/error messages.
2. **Remote lookup** — given `<type> <id>`:
   - `LoadResourcesFromRemote` → `RemoteResources.GetAll(type)` →
     `map[id]*RemoteResource` (each has `ID`, `ExternalID`, `Data`).
   - Match `<id>` against `ExternalID`, then `ID`.
   - `MapRemoteToState` → URN-keyed `state.State` for `delete`'s state argument.
3. **Capability gating** — type-assert the resolved provider for the optional
   interface (`LifecycleManager`, `ExternalIDSetter`); absent → clear
   `ErrVerbNotSupported` ("verb X not supported for type Y").

**`apply -f` = scoped reconcile (Decision B2).** It reuses the existing pipeline
and diverges at exactly one point:

```
project.Load(-f paths)  →  ResourceGraph()  =  TARGET graph        (reused)
syncer.Sync:
    LoadResourcesFromRemote → MapRemoteToState → StateToGraph  =  SOURCE graph  (reused)
    ── DIVERGENCE ──  scope SOURCE to TARGET's URNs (drop out-of-scope)         (NEW seam)
    planner.Plan(scopedSource, target)  →  only creates/updates, no deletes     (reused)
    executePlan → LifecycleManager.Create/Update                                (reused)
```

The single new seam is a syncer option, e.g. `syncer.WithScopeToTarget()`, that
filters the source graph to URNs present in the target before planning. Because
the differ's delete rule is "URN in source but not in target → delete," an empty
"source-minus-target" set means **no deletes** — by construction. Create/update
classification for target resources is unchanged (a target URN present in remote
→ update; absent → create).

`--dry-run` renders the resulting `plan` and stops; otherwise prompt (unless
`--confirm`) then `executePlan`. The plan/preview rendering reuses the syncer's
existing plan representation — no bespoke diff renderer needed.

**Fresh project/provider per `apply -f`:** each invocation builds a clean
project + provider, loads only the `-f` paths, and builds the target graph from
exactly those resources (handlers accumulate specs with no `Reset`).

## Section C — Read path (`get` / `describe` / `-o yaml`)

- **`get <type>` (list):** merge `LoadResourcesFromRemote` (managed) +
  `LoadImportable` (unmanaged). Table columns:
  `EXTERNAL-ID · REMOTE-ID · NAME · MANAGED`. `--managed`/`--unmanaged` filter the
  merge; `-l key=value` applies `lister.Filters`. Providers lacking
  `LoadImportable` degrade to managed-only with a one-line note.
  - **Merge de-dup rule:** a resource is keyed by **remote-id**. If the same
    remote-id appears in both loads, the **managed** entry wins and `MANAGED =
    yes` (external-id present). Unmanaged-only entries show a blank `EXTERNAL-ID`
    and `MANAGED = no`. So `MANAGED` is exactly "has an external id."
- **`get <type> <id>`:** remote lookup, render one resource.
- **`-o yaml|json`:** `Exporter.FormatForExport` on the selection → re-appliable
  spec; `yaml`/`json` are encodings of the same `FormattableEntity`.
- **`describe <type> <id>` (v1):** a **templated layout** of the same
  `get -o yaml` spec content — readable presentation, no separate formatter. Rich
  describe deferred.

## Section D — Mutate path (`delete`, `apply -f`)

**`apply -f <file|dir>...`** (scoped reconcile — see Section B):
1. Collect spec files recursively from each `-f` path → fresh `project.Load` →
   `ResourceGraph()` = target.
2. `syncer.New(..., syncer.WithScopeToTarget())` → `Sync(ctx, target)`: loads
   remote, scopes source to target's URNs, plans (creates/updates only).
3. Render the plan. If `--dry-run`: stop. Else prompt unless `--confirm`, then
   `executePlan`. **Never deletes** (scoped source ⇒ no delete operations).

Example plan output (reuses the syncer's plan representation):
```
~ update event-stream-source/my-source  (remote: src_abc)
+ create event-stream-source/new-source
Plan: 1 create, 1 update   (apply -f never deletes)
```

**`delete <type> <id>`** (imperative): remote lookup → `MapRemoteToState` for the
`state` output that `Delete` needs → preview single deletion → confirm unless
`--confirm` → `LifecycleManager.Delete(ctx, id, type, state)`.

**Capability gate:** `delete` on a provider without real `LifecycleManager`
fails fast (`account`). `apply -f` on an unsupported type surfaces the provider's
own create/update error.

## Section E — `set-external-id` + `ExternalIDSetter` capability

The SDK setters are buried inside handlers' `Import` today
(`event-stream/source/handler.go:561`, `datagraph/.../handler.go:152`,
`retl/sqlmodel/handler.go:294`). Surface a uniform **optional** capability:

```go
type ExternalIDSetter interface {
    SetExternalID(ctx context.Context, resourceType, remoteID, externalID string) error
}
```

Each capable handler implements it by delegating to the SDK setter it already
calls. The verb type-asserts for it; types without a setter reject the verb. This
is the single additive foundation piece that lets the tabled `import` facade land
later as composition.

## Section F — Cross-cutting

- **Output:** reuse `lister` (table) and `ui` (formatted layout); add a small
  `printer` for `-o yaml|json` over `FormattableEntity`.
- **Errors:** wrap with context (`fmt.Errorf("...: %w", err)`); sentinels
  `ErrUnsupportedType`, `ErrResourceNotFound`, `ErrVerbNotSupported`.
- **Testing:** unit tests per verb against a mock `Provider`; E2E in `cli/tests/`
  for the apply cycle, with `get -o yaml | apply -f -` round-trip as an anchor.
- **Risk/Impact:** Medium — new top-level verbs + a new `-f` mode on `apply`;
  `delete`/`apply -f` mutate remote (mitigated by dry-run + prompt-by-default).
  Additive only — existing `apply --location` behavior is unchanged. The one
  ergonomic hazard is `apply -f file` (scoped) vs `apply --location file`
  (prunes); mitigated by mutual-exclusion + loud help (Decision 12).
- **Deprecation:** `workspace accounts list`, `workspace retl-sources list`,
  `workspace event-stream-sources list` gain a deprecation notice pointing at
  `get <type>` (Decision 13). They keep working in v1; removal is a later step.
- **Rollout:** universal capability-gated design; implement beachhead first
  (`event-stream-source`, `retl-source-sql-model` — full CRUD + setter), widen
  after.

---

## Capability matrix (read = universal; mutate/claim = gated)

| Type | get/describe/-o yaml | delete / apply | set-external-id |
|------|:--:|:--:|:--:|
| `event-stream-source` | ✅ | ✅ | ✅ |
| `retl-source-sql-model` | ✅ | ✅ | ✅ |
| `data-graph` (+ model, relationship) | ✅ | ✅ | ✅ (experimental flag) |
| `transformation`, `library` | ✅ | ✅ | ✅ |
| `property`, `event`, `tracking-plan`, `custom-type`, `category` | ✅ | ✅ | gated per setter availability |
| `account` | ✅ | ⚠️ SDK CUD exists (origin/main #617); workspace provider not yet wired to `LifecycleManager` | ❌ no SDK setter |

> **Beachhead is verified; other rows are aspirational.** Only
> `event-stream-source` and `retl-source-sql-model` (full CRUD + setter) are
> confirmed v1 contracts. Non-beachhead rows reflect *expected* capability and
> must be verified per type during implementation (which datacatalog sub-types
> actually expose an SDK setter is an open question in the TABLED doc).
> Capability gating means an unverified row simply doesn't offer the gated verb —
> it never silently does the wrong thing.

## Key code seams

- `cli/internal/provider/composite.go:265,273` — `providerForKind`/`providerForType`
  (expose `ProviderForType`); `:68,72` — `SupportedKinds`/`SupportedTypes`.
- `cli/internal/resources/collection.go:11,20,51` — `RemoteResource`,
  `RemoteResources`, `GetAll(type)`.
- `cli/internal/provider/provider.go` — `LifecycleManager` (Create/Update/Delete),
  `ManagedRemoteResourceLoader`, `UnmanagedRemoteResourceLoader`, `StateLoader`,
  `Exporter`.
- `cli/internal/syncer/...` — `differ`/`planner`/`executePlan` + `syncer.New`
  options; **add `WithScopeToTarget()`** (the one new seam for `apply -f`).
- `cli/internal/cmd/project/apply/apply.go:25` — existing `apply` command (add
  the `-f` flag + scoped path; `--location` unchanged).
- `cli/internal/cmd/root.go:83` — `rootCmd.AddCommand(...)` registration site for
  the new top-level verbs.
- `cli/internal/lister/lister.go:47` — `List(ctx, type, filters)` side interface
  (table rendering reuse).
- `cli/internal/project/importer/importer.go` — `FormatForExport` + `writer`
  (reused by `-o yaml`).

## Open questions / future

- **Accounts CUD (origin/main #617):** SDK now has `Create`/`Update`/`Delete`
  but no `SetExternalID`. To expose `delete`/`apply -f` for `account`, wire the
  workspace provider to `LifecycleManager` (widen-beyond-beachhead). `delete`
  works via remote-id; **`apply -f` needs an identity story** — with no external
  ID, accounts never appear as managed in `MapRemoteToState`, so scoped reconcile
  always classifies them as create. Resolve (external-id support for accounts, or
  name/remote-id matching in the provider) before enabling account `apply -f`.
- Evolve `apply` to safe-by-default `+ --prune` (Decision-12 follow-up).
- Short type aliases.
- Rich, context-aware `describe`.
- Adoption facade (`import <type> <id>`) — see TABLED doc; lands as composition
  on this foundation.
