# Resource Adoption Facade — TABLED for a later conversation

**Status:** Parked. Not part of the current core deliverable (the k8s-style
`get` / `get -o yaml` / `describe` / `delete` / `apply -f` command structure).
This document captures the discussion so it can be picked up as its own design.

**Date:** 2026-06-07

---

## Why this is tabled separately

The core work is a uniform, k8s-style **verb layer** over all resources exposed
through the `api` package, operating via the CLI provider/composite layer. The
*adoption facade* (how a user brings an unmanaged remote resource under
management with minimal friction) is an important, related capability — but it
is a distinct design problem with its own UX and policy questions. Solving it
inline would overload the core spec, so it gets its own space.

---

## Motivating problem: IaC drift causing unwanted changes

The reason `apply` produces unwanted changes is that **spec and remote drift
apart**. The adoption facade closes that gap by making the spec the literal
materialization of the remote state, so that the first reconcile has nothing to
revert.

## The core insight (shared understanding reached in discussion)

Today's commands already *are* the primitives we're proposing, but bundled at
workspace granularity:

| Today (workspace-wide)                          | Equivalent primitive (per-resource) |
| ----------------------------------------------- | ----------------------------------- |
| `import workspace` (writes specs + proposed IDs)| `get -o yaml` + **proposed external IDs** |
| `apply` realizing imported metadata on remote   | `set-external-id <type> <remote-id> <external-id>` |

In other words:
- **`import` ≈ `get -o yaml` + proposed external IDs**, over the whole workspace.
- **`apply` on imported metadata ≈ `set-external-id`**, over the whole workspace.

The facade we want to design exposes these as composable per-resource verbs and
a blessed shortcut.

## Verb relationships under discussion

- **Primitives** (foundation): `set-external-id` (pure claim), `get -o yaml`
  (materialize the re-appliable spec via `Exporter.FormatForExport`).
- **Facade**: per-resource `import <type> <remote-id>` = claim + materialize, in
  one command. Reuses the existing importer machinery
  (`LoadImportable` / `FormatForExport` / `writer.Write`), scoped to one ID.

## Decisions captured (to honor when this resumes)

1. **`apply` may also set external IDs** for *intentional* metadata inputs. This
   is acceptable and stays.
2. **`import` as a single command must set the external ID itself (eager)** — it
   should NOT offload the claim to a later `apply`. This is a deliberate
   divergence from today's deferred behavior.
3. **Per-resource `import <type> <remote-id>`**:
   - Accepts `external-id` as an **optional** argument (user choice).
   - When the user does **not** supply one, it **proposes** a value and surfaces
     it for the user to review and accept.
   - Supports a flag to **auto-accept** proposals (non-interactive).
4. **Workspace `import`**:
   - **Always auto-proposes** external IDs.
   - Surfaces proposals to the user for confirmation, OR runs with a flag to
     **auto-accept** proposals.

## Open strategic question (and current lean)

> Should we ship the consistent-but-deferred "Option A" now and fix the strategy
> later, or lay the foundation first so "Option C" (primitives + blessed
> `import` facade) lands cleanly before per-resource `import` is introduced?

**Current lean: foundation-first, scoped tightly.**
- Do **not** ship deferred "A" as an interim — its deferred-claim behavior would
  become load-bearing (habits, tests, existing `import workspace`), making a
  later move to eager-claim a breaking change + churn.
- Instead, build the one additive, optional capability the eager model needs —
  **`ExternalIDSetter`** (wraps SDK setters already used inside each handler's
  `Import`) — as part of the core foundation, and ship `set-external-id` as a
  primitive verb.
- Defer the **facade policy** (eager `import`, proposal/confirm UX, bulk
  convergence). When resumed, it becomes pure composition rather than a refactor.

## Open questions for the resumed conversation

- Proposal algorithm for external IDs (naming/`namer.Namer`) and collision
  handling when proposing across many resources.
- Interactive review/confirm UX vs `--yes`/auto-accept flag ergonomics.
- Should bulk `import workspace` converge onto eager-claim (the "C" end-state),
  and what's the migration story for its current deferred behavior?
- Which resource types lack an SDK external-ID setter (e.g. some datacatalog
  sub-types), and how the facade degrades for them.
- Interaction with the future **declarative apply mode**: a claimed-but-not-yet-
  specified resource is a deletion candidate; eager-`import` that always
  materializes the spec avoids the orphan, but `set-external-id` used alone does
  not — needs a guardrail/warning policy.

## Foundation the core spec WILL provide (so this can resume as composition)

- Uniform remote **read** (`LoadResourcesFromRemote`) and **addressing/lookup**
  (`<type> <external-id|remote-id>` → remote resource + state).
- Uniform **spec materialization** (`get -o yaml` via `Exporter.FormatForExport`).
- The additive optional **`ExternalIDSetter`** capability + `set-external-id`
  primitive verb.
- The **executor seam** (imperative-remote now; declarative-managed later behind
  a flag) that the facade will also ride on.
