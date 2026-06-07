# K8s-style Resource Commands — Execution Log (immutable)

> Append-only log of subagent-driven execution of
> `docs/superpowers/plans/2026-06-07-k8s-style-resource-commands.md`.
> Each entry records discovery, decisions, bugs, fixes, and the commit produced.
> **Never edit or delete prior entries** — only append below.

## Execution context

- **Plan:** `docs/superpowers/plans/2026-06-07-k8s-style-resource-commands.md` (13 tasks, 5 phases)
- **Design:** `docs/superpowers/specs/2026-06-07-k8s-style-resource-commands-design.md`
- **Method:** superpowers:subagent-driven-development — fresh implementer subagent
  per task, followed by spec-compliance review then code-quality review.
- **Branch:** work performed on `claude/sweet-mccarthy-8c3eca`, pushed to remote
  `feat/k8s-style-imperative-commands`.
- **Build env:** gvm wrapper — `go` lives at `/usr/local/go/bin/go` (go1.24.0);
  `make` targets prefixed with `PATH="/usr/local/go/bin:$PATH"`.

## Commit-per-task tracker

| Task | Title | Status | Commit(s) |
|------|-------|--------|-----------|
| 1 | Public ProviderForType on composite | ✅ done | `1f10cf8e`, `c81df306` |
| 2 | ExternalIDSetter capability + beachhead | ✅ done | `d6052ca8`, `adb4474c` |
| 3 | syncer.WithScopeToTarget() seam | ✅ done | `81a099ef`, `84d3a070` |
| 4 | resourceops Resolver | ✅ done | `8abfbf7f` |
| 5 | Reader — managed+unmanaged merge | ✅ done | `cdaad663`, `4a5991ed` |
| 6 | Single-resource spec materialization | ✅ done | `50c0bb5d`, `865ab9c4` |
| 7 | get command | ✅ done | `b067ccc0`, `5ea2dfdb`, `f245c9a5` |
| 8 | describe command | ✅ done | `058312d9`, `5a10f332` |
| 9 | set-external-id command | ✅ done | `62f4d44d`, `31712c0a` |
| 10 | delete command | ✅ done | `14b710fb`, `56806eb5` |
| 11 | apply -f scoped mode | pending | — |
| 12 | Deprecate per-noun list commands | pending | — |
| 13 | E2E round-trip + scoped no-delete | pending | — |

---

## Log entries

### 2026-06-07 — Execution kickoff

- Located the plan: branch `feat/k8s-style-imperative-commands` exists locally
  only (checked out in worktree `keen-mendel-df91eb` at `0dae6cad`); no remote
  ref yet. Reset working branch `claude/sweet-mccarthy-8c3eca` to `0dae6cad` so
  the plan/design files are present in this worktree.
- Verified `go build ./...` passes in this worktree (go1.24.0 via gvm).
- Created the 13-task tracker and this immutable log.
- Will push `claude/sweet-mccarthy-8c3eca` → remote `feat/k8s-style-imperative-commands`
  to establish the remote branch, then keep pushing after each task.

### 2026-06-07 — Task 1: Public `ProviderForType` on composite ✅

- **Implementer (sonnet):** Added `TypeRouter` interface + public `ProviderForType`
  on `*CompositeProvider` delegating to the existing private `providerForType`
  (no map-lookup duplication); added a compile-time guard
  `var _ TypeRouter = (*CompositeProvider)(nil)`. Test
  `TestCompositeProvider_ProviderForType` uses the existing
  `testutils.NewMockProvider`. Commit `1f10cf8e`.
- **Discovery:** Plan assumed `NewCompositeProvider` returns `*CompositeProvider`;
  it actually returns `(Provider, error)`, so the test's interface assertion
  `cp.(provider.TypeRouter)` is the right shape. Private `providerForType`
  confirmed at `composite.go:273` as the plan predicted.
- **Spec review (sonnet):** ✅ All 6 requirements met; error chain traced; no
  missing/extra work.
- **Code-quality review (sonnet):** ⚠️ Important issue — the implementer's first
  cut added a global `Unwrap()` to the existing `ErrUnsupportedResourceType` so
  `errors.Is(.., ErrUnsupportedType)` would match. That couples six unrelated
  lifecycle error sites in `baseprovider.go` to the routing sentinel and would
  wrongly match in Task 4's resolver.
- **Fix (sonnet):** Removed the broad `Unwrap()`; `ProviderForType` now wraps the
  not-found case itself with `fmt.Errorf("%q: %w", resourceType, ErrUnsupportedType)`,
  delegating to `providerForType` for the lookup. `ErrUnsupportedType` is now
  scoped strictly to routing. Commit `c81df306`. `make test` + `make lint` green.

### 2026-06-07 — Task 2: `ExternalIDSetter` capability + beachhead ✅

- **Implementer (sonnet):** Added optional `ExternalIDSetter` interface to
  `provider.go`. event-stream + retl `Provider.SetExternalID` route by
  resourceType through their `handlers` map, wrapping `provider.ErrUnsupportedType`
  on unknown type. Extracted thin `Handler.SetExternalID` on both source +
  sqlmodel handlers (event-stream wraps `client.SetExternalID`, retl wraps
  `client.SetExternalId` — lowercase d preserved); each `Import` now calls the
  thin method at the same position (no behavior/ordering change). Added
  `TestProvider_SetExternalID` to both providers. Commit `d6052ca8`.
- **Discovery:** event-stream `MockSourceClient` already had `SetExternalIDCalled()`.
  retl's in-file `mockRETLStore` lacked a setter — added `SetExternalId` + a
  *Called accessor (test-only). retl mock's `ListRetlSources` uses a variadic
  `...ListRetlSourcesOption` signature.
- **Spec review (sonnet):** ✅ Compliant; Import ordering preserved on both sides
  (retl sets external-id BEFORE diffing state; event-stream AFTER update — both
  unchanged). One minor: event-stream unknown-type test weaker than retl's.
- **Code-quality review (sonnet):** Approve with 2 test-layer fixes — strengthen
  the event-stream unknown-type assertion to `ErrorIs(ErrUnsupportedType)`, and
  rename retl mock's `setExternalIdCalled`/`SetExternalIdCalled()` to the `ID`
  convention (interface method `SetExternalId` left locked to the API client).
  Noted pre-existing `remoteId`/`sourceId` naming in lower layers as out-of-scope.
- **Fix (sonnet):** Both applied. Commit `adb4474c`. `make test` + `make lint` green.

### 2026-06-07 — Task 3: `syncer.WithScopeToTarget()` seam ✅

- **Implementer (sonnet):** Added `scopeToTarget` field + `WithScopeToTarget()`
  option + private `scopeGraphToTarget(source, target)` helper. Guard inserted in
  `apply` immediately after `source := StateToGraph(state)` and before
  `planner.Plan` — filters source to URNs present in target. Left
  `executePlanConcurrently`'s `StateToGraph` untouched (used only for dependency
  ordering; ops come from the already-fixed plan). Two tests:
  `TestSyncer_ScopeToTarget_SuppressesDeletes` (dry-run, asserts no Delete op,
  Update for A present, B absent) and `..._Execution` (non-dry-run, asserts no
  Delete executed against B via `DataCatalogProvider.OperationLog`). Commit `81a099ef`.
- **Discovery:** Existing `DataCatalogProvider` exposes `OperationLog` entries with
  the resource ID at `Args[0]` (from `logOperation`), enabling the execution-path
  assertion without new fakes.
- **Spec review (sonnet):** ✅ Compliant; guard placement correct; helper doesn't
  mutate inputs; concurrent path unchanged; execution test is not a tautology.
- **Code-quality review (sonnet):** Approve with 2 Important test improvements —
  the execution test lacked a positive "Update for A actually ran" assertion, and
  the `Args[0]` index needed a clarifying comment. Minor suggestions (single-loop
  test, `WithScopeToTarget(bool)` signature) declined: the parameterless signature
  is mandated by the plan + Task 11.
- **Fix (sonnet):** Added positive Update assertion + `Args[0]` comment + improved
  helper "why" doc-comment. Commit `84d3a070`.
- **⚠️ Discovery (flaky test, pre-existing):** `make test` intermittently fails
  `TestRunTasks_ErrorWithDependentTask` in `cli/pkg/tasker`. Confirmed unrelated to
  these changes — passes consistently when run in isolation (3×). Tracked as a
  known flaky to watch across later tasks; not caused by this work.

### 2026-06-07 — Task 4: resourceops Resolver ✅

- **Implementer (sonnet):** New package `cli/internal/resourceops` with `Resolver`
  (`New`, `ProviderFor`, `FindRemote` external-id-first/remote-id-fallback,
  `ExternalIDSetterFor` capability gate, private `loadAll` single managed-load
  path) + sentinels `ErrResourceNotFound`/`ErrVerbNotSupported`. 7 black-box
  tests (96% coverage) including capability gate BOTH directions (a local
  `mockExternalIDSetter` embeds `testutils.MockProvider` + adds `SetExternalID`).
  Commit `8abfbf7f`.
- **Discovery:** Reused `testutils.MockProvider` (field `LoadResourcesFromRemoteVal`)
  instead of a hand-rolled `fakeProvider` — reviewer called this a justified
  positive deviation. Noted pre-existing `RemoteId` (lowercase d) field in
  `testutils.MockProvider` as a convention follow-up (out of scope).
- **Spec review (sonnet):** ✅ Compliant; external-id-first ordering verified;
  `loadAll` is the single load path; no premature Task 5 unmanaged merge.
- **Code-quality review (sonnet):** ✅ Approve, no blocking issues. Minor: `loadAll`
  doc says "managed" (will be updated in Task 5 when merge lands); `New` lacks a
  doc comment. `loadAll` seam confirmed well-shaped for Task 5 (body-only change).

### 2026-06-07 — Task 5: Reader — managed+unmanaged merge ✅

- **Implementer (sonnet):** Added `reader.go` — `Scope` (All/Managed/Unmanaged),
  `Row{ExternalID,RemoteID,Name,Managed}`, `ListRows(ctx, prov, type, scope)`,
  `SupportsUnmanaged(prov)`, private `mergedRemote` (managed+unmanaged, managed
  wins on RemoteID dup), `extractName`. Extended resolver `loadAll` to delegate to
  the shared `mergedRemote` so `FindRemote` resolves unmanaged-by-remote-id.
  Commit `cdaad663`.
- **Design decisions / discovery:**
  - `ListRows` takes the narrower `provider.ManagedRemoteResourceLoader` (not full
    `Provider`) and asserts `UnmanagedRemoteResourceLoader` at runtime.
  - **Degraded surfacing:** plan wanted a `Degraded` flag, but the binding test
    pins `ListRows(...) ([]Row, error)`. Resolved by keeping the signature and
    providing a companion `SupportsUnmanaged(prov) bool` probe (two-call protocol).
    Task 7's `get` must call `SupportsUnmanaged` and print a one-line note when it
    is false and scope includes unmanaged.
- **Spec review (sonnet):** ✅ Compliant; merge/de-dup/scope correct; `loadAll`
  uses the shared primitive; nil-safe. Minor: no test for a provider lacking the
  unmanaged interface entirely.
- **Code-quality review (sonnet):** Approve with minor rework — document the
  two-call protocol + namer choice, exhaustive scope switch, rename a misleading
  test, add the missing no-interface-branch test.
- **Fix (sonnet):** All applied. New `TestReader_List_ProviderWithoutUnmanagedInterface`
  uses a `managedOnlyProvider` (no `LoadImportable`), exercising the
  `if !ok { return result, nil }` branch. Commit `4a5991ed`.

### 2026-06-07 — Task 6: Single-resource spec materialization (-o yaml/json) ✅

- **Implementer (sonnet):** Added `SpecYAML`/`SpecJSON` to `reader.go` (find single
  resource → single-entry collection → `Exporter.FormatForExport` → encode) and
  `printer.go` (`EncodeYAML` via project formatter; `EncodeJSON` via
  YAML→map→JSON so keys are lowercase & a top-level `"kind"` exists). Real
  round-trip test using `eventstream.New(mockClient)` + `SetGetSourcesFunc`.
  Commit `50c0bb5d`.
- **🐞 Discovery (real pre-existing bug, FIXED):** event-stream
  `source/handler.go` `LoadResourcesFromRemote` stored `Data` as a VALUE
  `EventStreamSource`, but `FormatForExport` and `MapRemoteToState` type-assert a
  POINTER `*EventStreamSource`. This would fail at runtime for `get -o yaml` and
  import. Fixed to store `&s` and updated `MapRemoteToState` + handler/provider
  tests. Spec review confirmed the loopvar `&s` is SAFE (go.mod = go 1.24,
  per-iteration scoping) — explicit `s := source` copy added belt-and-suspenders.
- **⚠️ Known limitation (documented follow-up):** single-resource materialization
  loads only the target provider's remote, so CROSS-PROVIDER references (e.g. a
  source pointing at a tracking plan owned by another provider) can't be resolved.
  Mitigated: resolver `Remote` is now populated with the provider's full managed
  collection (same-provider refs resolve), and `FormatForExport` errors are
  wrapped with a clear message naming the limitation. Full fix (composite-level
  remote) deferred — aligns with the design's beachhead scope.
- **Spec review (sonnet):** ✅ Compliant; bug fix confirmed real + safe.
- **Code-quality review (sonnet):** Changes needed — populate the export resolver
  (correctness), share the external-id-first find rule between `FindRemote` and
  the materializer (dedup), rename a misleading test, fix a stale loopvar comment,
  add `printer_test.go`, add a nil guard.
- **Fix (sonnet):** All applied — extracted shared `findInMap` helper (used by both
  `FindRemote` and `specContent`); populated resolver Remote + wrapped error;
  removed the redundant test; added `printer_test.go` (4 tests). Commit `865ab9c4`.

### 2026-06-07 — Task 7: top-level `get` command ✅ (Phase 2)

- **Implementer (sonnet):** Added `cli/internal/cmd/get/` with `NewCmdGet()` and a
  testable `RunGet(ctx, out, cp, args, opts)` core behind a package `Composite`
  seam (`ProviderForType` + `SupportedTypes`). Flags `-o/--output`,
  `--managed`/`--unmanaged` (mutually exclusive), `-l/--selector`. Dispatch:
  1 arg→`ListRows` (table via `text/tabwriter` for testability, or json);
  2 args+yaml/json→`SpecYAML`/`SpecJSON`; 2 args+table→single row. Registered in
  root.go. Commit `b067ccc0`.
- **Spec review (sonnet):** Found `-l/--selector` parsed but SILENTLY DROPPED — a
  no-op. Everything else compliant.
- **Fix 1 (sonnet):** Made `--selector` functional via `filterRows` over the Row
  columns (`external-id`/`remote-id`/`name`/`managed`, AND semantics, unknown key
  → error). To make the degraded-note path testable, narrowed the seam so
  `Composite.ProviderForType` returns `GetProvider` (=`ManagedRemoteResourceLoader`)
  with a `compositeShim` wrapping the real composite; `runSingle` re-asserts
  `provider.Provider` for yaml/json. Added selector + degraded-note tests.
  Commit `5ea2dfdb`.
- **Code-quality review (sonnet):** Seam inversion judged JUSTIFIED (a full
  `provider.Provider` always satisfies `SupportsUnmanaged`, so the degraded path is
  untestable without a narrower return). Found 3 Important UX-correctness bugs:
  (1) `parseSelector` silently drops malformed `-l` entries; (2) degraded note
  fired on the single-resource path too; (3) `-o yaml` on the list path silently
  rendered a table.
- **Fix 2 (sonnet):** (1) `parseSelector` now errors on non-`key=value`; (2) note
  moved into `runList` only; (3) `--output` validated up front, list+yaml returns a
  clear "single resource only" error; aligned telemetry to the `[]KV` slice form;
  added not-found + malformed-selector + list-yaml + single-no-note tests (21 tests
  total). Commit `f245c9a5`. Deferred minors: adopt `MarkFlagsMutuallyExclusive`,
  `validateType`/`ProviderForType` dedup, `GetProvider` naming.

### 2026-06-07 — Task 8: `describe` command ✅

- **Implementer (sonnet):** Added `cli/internal/cmd/describe/` with `NewCmdDescribe()`
  (`ExactArgs(2)`) and testable `RunDescribe(ctx, out, router, type, id)` —
  resolves provider, `SpecYAML` → decode YAML → `ui.FormattedMap` + a
  `Managed: yes/no` line, all to `out`. Registered in root.go. 5 tests using a
  real event-stream provider. Commit `058312d9`.
- **Spec review (sonnet):** ✅ Compliant; renders to `out`, reuses `SpecYAML`,
  sentinels propagate.
- **Code-quality review (sonnet):** Changes needed — (Important) describe did a
  redundant double remote load (`FindRemote` + `SpecYAML`); (Important) `RunE`
  used a non-idiomatic named return `(err error)`. Minors: a duplicate test and a
  weak `Args` assertion.
- **Fix (sonnet):** Added `resourceops.SpecYAMLWithManaged` (returns yaml + managed
  in one find; `SpecYAML`/`SpecJSON` signatures unchanged, delegate to the shared
  internal path); describe now calls it once (one load instead of two). Switched to
  `var err error`. Strengthened the arity test; removed the duplicate. Commit
  `5a10f332`.
- **Controller verification (no regression):** Confirmed `specContent` (hence all
  `Spec*` funcs) is MANAGED-only — already true since Task 6. The old describe would
  have failed at `SpecYAML` for an unmanaged resource too, so the single-load
  refactor narrows nothing in practice; describe renders managed resources, which is
  the v1 contract.

### 2026-06-07 — Task 9: `set-external-id` command ✅ (Phase 3)

- **Implementer (sonnet):** Added `cli/internal/cmd/setexternalid/` —
  `NewCmdSetExternalID()` (`ExactArgs(3)`) + testable
  `RunSetExternalID(ctx, out, router, type, remoteID, externalID)` that resolves the
  optional setter via `Resolver.ExternalIDSetterFor` and calls
  `SetExternalID(ctx, type, remoteID, externalID)`, printing a confirmation.
  Registered in root.go. 4 tests (success, unsupported-capability→ErrVerbNotSupported,
  unknown-type→ErrUnsupportedType, arity). Commit `62f4d44d`.
- **Spec review (sonnet):** ✅ Compliant; arg→param order verified correct (no
  remoteID/externalID swap).
- **Code-quality review (sonnet):** Approve with minors — import path-ordering in
  root.go; success test only checked the external id; no error-path test; the mock
  discarded its args (an arg transposition would be invisible).
- **Fix (sonnet):** Path-sorted imports; enhanced `MockSourceClient.SetExternalID` to
  capture args + a `SetExternalIDErr` injector; strengthened the success test to
  assert all three fields AND the correct forwarded arg order; added a client-error
  propagation test. Commit `31712c0a`.

### 2026-06-07 — Task 10: `delete` command ✅

- **Implementer (sonnet):** Added `resourceops.Delete(ctx, prov, type, id)` (reuses
  `mergedRemote`+`findInMap`; rejects unmanaged via new `ErrUnmanaged`; builds
  state via `MapRemoteToState`, calls `prov.Delete(ctx, externalID, type,
  sr.Data())` — same path as `syncer.deleteOperation`) and a
  `cli/internal/cmd/delete/` command (`ExactArgs(2)`, `--confirm` default false =
  prompt, true = skip) with a testable `RunDelete(ctx, out, prov, type, id,
  skipConfirm, confirmFn)` doing preview→confirm→delete. Commit `14b710fb`.
- **Discovery:** `mergedRemote` returns `(map, error)` (degraded flag dropped in
  Task 5 for `SupportsUnmanaged`) — implementer adapted.
- **Spec review (sonnet):** ✅ Compliant; critically verified `prov.Delete` receives
  the resolved EXTERNAL id (not the remote id) even when looked up by remote-id —
  no transposition bug. Unmanaged/abort paths confirmed to NOT call Delete.
- **Code-quality review (sonnet):** Changes needed — import path-order in root.go;
  unknown type didn't list valid types (inconsistent with `get`); no test for
  `prov.Delete` returning an error. (Reviewer questioned `--confirm` semantics vs
  apply; kept as-is — the plan mandates `--confirm`=skip for delete.)
- **Fix (sonnet):** Path-sorted imports; clarified `--confirm` help; added
  `validateType` listing valid types (+ tests); added delete-error wrap test.
  Commit `56806eb5`. Deferred minors: show resource name in preview, shared test
  fixtures, pre-existing `ImportArgs.RemoteId` naming.
