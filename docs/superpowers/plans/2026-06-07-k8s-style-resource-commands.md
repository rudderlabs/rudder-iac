# K8s-style Resource Commands Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add top-level `get` / `describe` / `delete` / `set-external-id` verbs plus a scoped, delete-free `apply -f` mode to `rudder-cli`, working uniformly over registered resource types (beachhead: `event-stream-source`, `retl-source-sql-model`).

**Architecture:** Thin Cobra commands over a new testable `cli/internal/resourceops/` package. Reads use the composite provider's remote loaders + `Exporter.FormatForExport`. `delete` calls `LifecycleManager.Delete`. `apply -f` reuses the existing apply pipeline (`project.Load` → `ResourceGraph` → `syncer`) and diverges at exactly one new seam — `syncer.WithScopeToTarget()` filters the source graph to the target's URNs so the planner emits only creates/updates, never deletes. `set-external-id` calls a new optional `ExternalIDSetter` capability.

**Tech Stack:** Go, Cobra, testify. Existing internals: `cli/internal/provider` (CompositeProvider), `cli/internal/resources` (Graph/RemoteResources/state), `cli/internal/syncer` (planner/differ/executePlan), `cli/internal/lister`, `cli/internal/ui`.

---

## Conventions for every task

- **Build env:** `go`/`make` must run in the gvm-enabled shell (`GVM_ROOT` set). Prefer the repo Makefile targets.
- **Per-package test:** `go test ./<pkg>/... -run <TestName> -v`
- **Full gate before each commit:** `make test` and `make lint` (must pass).
- **Tests:** testify `assert`/`require` only. Prefer whole-struct equality (see CLAUDE.md). Follow @apply-go-repo-conventions.
- **Errors:** wrap with `fmt.Errorf("...: %w", err)`; sentinels `ErrUnsupportedType`, `ErrResourceNotFound`, `ErrVerbNotSupported`.
- **IDs:** initialism is `ID` (e.g. `ExternalID`).
- **Commits:** end message with `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`. Do not push unless the user asks.

---

## File Structure

**Create:**
- `cli/internal/resourceops/resolver.go` — type→provider routing, remote lookup by external-id/remote-id, capability assertions, sentinels.
- `cli/internal/resourceops/resolver_test.go`
- `cli/internal/resourceops/reader.go` — list (managed+unmanaged merge), single-get, `-o yaml/json` via `FormatForExport`.
- `cli/internal/resourceops/reader_test.go`
- `cli/internal/resourceops/printer.go` — table + yaml/json rendering.
- `cli/internal/resourceops/printer_test.go`
- `cli/internal/resourceops/externalid.go` — `ExternalIDSetter` interface + claim helper.
- `cli/internal/resourceops/externalid_test.go`
- `cli/internal/resourceops/deleter.go` — managed-resource delete helper.
- `cli/internal/resourceops/deleter_test.go`
- `cli/internal/cmd/get/get.go` — `NewCmdGet()`.
- `cli/internal/cmd/describe/describe.go` — `NewCmdDescribe()`.
- `cli/internal/cmd/delete/delete.go` — `NewCmdDelete()`.
- `cli/internal/cmd/setexternalid/set_external_id.go` — `NewCmdSetExternalID()`.

**Modify:**
- `cli/internal/provider/composite.go` — add public `ProviderForType`.
- `cli/internal/provider/provider.go` — add `ExternalIDSetter` interface doc (optional capability).
- `cli/internal/providers/event-stream/provider.go` — implement `SetExternalID`.
- `cli/internal/providers/retl/provider.go` — implement `SetExternalID`.
- `cli/internal/syncer/syncer.go` — add `scopeToTarget` field, `WithScopeToTarget()` option, source filtering in `apply`.
- `cli/internal/cmd/project/apply/apply.go` — add `-f`/`--file` flag + scoped path.
- `cli/internal/cmd/root.go` — register the 4 new top-level commands.
- `cli/internal/cmd/workspace/*.go` — deprecation notices on the 3 `list` commands.

---

# Phase 0 — Foundation

## Task 1: Public `ProviderForType` on the composite

**Files:**
- Modify: `cli/internal/provider/composite.go`
- Test: `cli/internal/provider/composite_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestCompositeProvider_ProviderForType(t *testing.T) {
	p := newStubProvider([]string{"event-stream-source"}, nil) // kinds nil, types set
	cp, err := provider.NewCompositeProvider(map[string]provider.Provider{"es": p})
	require.NoError(t, err)

	router, ok := cp.(provider.TypeRouter)
	require.True(t, ok, "CompositeProvider must satisfy TypeRouter")

	got, err := router.ProviderForType("event-stream-source")
	require.NoError(t, err)
	assert.Same(t, p, got)

	_, err = router.ProviderForType("nope")
	assert.ErrorIs(t, err, provider.ErrUnsupportedType)
}
```

(If a stub provider helper doesn't exist in this package, add a minimal one in the test file implementing `provider.Provider` with configurable `SupportedTypes()`/`SupportedKinds()`.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cli/internal/provider/... -run TestCompositeProvider_ProviderForType -v`
Expected: FAIL — `TypeRouter`/`ErrUnsupportedType`/`ProviderForType` undefined.

- [ ] **Step 3: Implement**

In `composite.go` add:

```go
// ErrUnsupportedType is returned when no provider handles the requested type.
var ErrUnsupportedType = errors.New("unsupported resource type")

// TypeRouter resolves the provider responsible for a resource type.
// CompositeProvider satisfies it; individual providers need not.
type TypeRouter interface {
	ProviderForType(resourceType string) (Provider, error)
}

// ProviderForType returns the provider registered for the given resource type.
func (p *CompositeProvider) ProviderForType(resourceType string) (Provider, error) {
	prov, ok := p.registeredTypes[resourceType]
	if !ok {
		return nil, fmt.Errorf("%q: %w", resourceType, ErrUnsupportedType)
	}
	return prov, nil
}
```

(If a private `providerForType` already exists, keep it and have the public method delegate, or replace internal callers — do not duplicate the map lookup.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cli/internal/provider/... -run TestCompositeProvider_ProviderForType -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
make test && make lint
git add cli/internal/provider/composite.go cli/internal/provider/composite_test.go
git commit -m "feat(provider): expose ProviderForType + TypeRouter on composite"
```

---

## Task 2: `ExternalIDSetter` capability + beachhead implementations

**Files:**
- Modify: `cli/internal/provider/provider.go` (interface + doc)
- Modify: `cli/internal/providers/event-stream/provider.go`
- Modify: `cli/internal/providers/retl/provider.go`
- Test: `cli/internal/providers/event-stream/provider_test.go`, `cli/internal/providers/retl/provider_test.go`

**Expected small refactor (scope into this task's commit):** `SetExternalID` is
currently inlined inside each handler's `Import` (`event-stream/source/handler.go:561`,
`retl/sqlmodel/handler.go:294`). Extract a thin `SetExternalID(ctx, remoteID, externalID) error`
method on each handler, have `Import` call it, and have the provider route to it.
Do this for **both** event-stream and retl handlers here.

- [ ] **Step 1: Write the failing test (event-stream)**

```go
func TestProvider_SetExternalID(t *testing.T) {
	mockClient := source.NewMockSourceClient() // existing mock in source pkg
	p := eventstream.New(mockClient)           // real constructor (see provider_test.go:26)

	err := p.SetExternalID(context.Background(), source.ResourceType, "src_remote_123", "my-source")
	require.NoError(t, err)
	assert.True(t, mockClient.SetExternalIDCalled())

	err = p.SetExternalID(context.Background(), "not-a-type", "x", "y")
	assert.Error(t, err) // unsupported type for this provider
}
```

(`event-stream/source/mock_client.go` already exposes `SetExternalID` and `SetExternalIDCalled()`. The real constructor is `eventstream.New(<store>)` — existing `provider_test.go:26` already calls `eventstream.New(source.NewMockSourceClient())`.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cli/internal/providers/event-stream/... -run TestProvider_SetExternalID -v`
Expected: FAIL — `SetExternalID` undefined on provider.

- [ ] **Step 3: Implement**

In `provider.go` (provider package) add the optional capability with documentation:

```go
// ExternalIDSetter is an OPTIONAL provider capability: it associates an existing
// remote resource (remoteID) with a local external ID, making it managed.
// Providers whose backing SDK exposes a setter implement it; others do not, and
// the verb layer rejects `set-external-id` for their types via type assertion.
type ExternalIDSetter interface {
	SetExternalID(ctx context.Context, resourceType, remoteID, externalID string) error
}
```

In `event-stream/provider.go` add (route to the source handler via the `handlers` map the provider already holds; the handler's new thin method wraps `h.client.SetExternalID(ctx, remoteId, id)` from `source/handler.go:561`):

```go
func (p *Provider) SetExternalID(ctx context.Context, resourceType, remoteID, externalID string) error {
	if resourceType != source.ResourceType {
		return fmt.Errorf("%q: %w", resourceType, provider.ErrUnsupportedType)
	}
	return p.handlerFor(resourceType).SetExternalID(ctx, remoteID, externalID)
}
```

(Use whatever handler-access the provider already has — providers hold a `handlers` map, not a named `sourceHandler` field. Add the thin `Handler.SetExternalID` method described above.)

In `retl/provider.go` add the analogous method for `sqlmodel.ResourceType`, delegating to the handler whose new thin method wraps the SDK's `SetExternalId(ctx, remoteID, externalID)` (note lowercase-d, `retl/sqlmodel/handler.go:294`).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cli/internal/providers/event-stream/... ./cli/internal/providers/retl/... -run TestProvider_SetExternalID -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
make test && make lint
git add cli/internal/provider/provider.go cli/internal/providers/event-stream/ cli/internal/providers/retl/
git commit -m "feat(provider): add optional ExternalIDSetter capability (event-stream, retl)"
```

---

## Task 3: `syncer.WithScopeToTarget()` — the scoped-apply seam

**Files:**
- Modify: `cli/internal/syncer/syncer.go`
- Test: `cli/internal/syncer/syncer_test.go`

- [ ] **Step 1: Write the failing test**

Drive it through behavior: with scoping on, a resource present in remote (source) but absent from target must NOT produce a Delete operation.

```go
func TestSyncer_ScopeToTarget_SuppressesDeletes(t *testing.T) {
	// remote has A and B; target has only A (modified).
	prov := newFakeSyncProvider(/* remote: A,B */)
	target := graphWith(/* A' */)

	s, err := syncer.New(prov, testWorkspace, syncer.WithDryRun(true), syncer.WithScopeToTarget())
	require.NoError(t, err)

	plan := captureReportedPlan(t, s, target) // via a test reporter
	for _, op := range plan.Operations {
		assert.NotEqual(t, planner.Delete, op.Type, "scoped apply must not delete B")
	}
}
```

(Use the existing syncer test helpers/fakes if present; otherwise add a minimal fake `SyncProvider` and a capturing `SyncReporter`.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cli/internal/syncer/... -run TestSyncer_ScopeToTarget_SuppressesDeletes -v`
Expected: FAIL — `WithScopeToTarget` undefined (and/or delete present).

- [ ] **Step 3: Implement**

Add field + option + filter in `syncer.go`:

```go
// in ProjectSyncer struct:
scopeToTarget bool

func WithScopeToTarget() Option {
	return func(s *ProjectSyncer) error {
		s.scopeToTarget = true
		return nil
	}
}

// scopeGraphToTarget returns a copy of source containing only resources whose
// URN is also in target. Used by `apply -f` so the planner never emits deletes
// for resources outside the applied set.
func scopeGraphToTarget(source, target *resources.Graph) *resources.Graph {
	scoped := resources.NewGraph()
	for urn, r := range source.Resources() {
		if _, ok := target.GetResource(urn); ok {
			scoped.AddResource(r)
		}
	}
	return scoped
}
```

In `apply`, right after `source := StateToGraph(state)` (syncer.go:134):

```go
source := StateToGraph(state)
if s.scopeToTarget {
	source = scopeGraphToTarget(source, target)
}
```

Note (verified safe): `executePlanConcurrently` rebuilds `StateToGraph(state)` (syncer.go:246), but that graph is used only for `GetDependents()` task-ordering — executed operations are strictly `plan.Operations`, already fixed by `planner.Plan` at line 137. So scoping `source` at line 134 fully suppresses out-of-scope deletes; the concurrent path needs **no** separate scoping. Still add a test asserting execution (not just dry-run) performs no Delete when scoped, to lock this invariant against future refactors.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cli/internal/syncer/... -run TestSyncer_ScopeToTarget -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
make test && make lint
git add cli/internal/syncer/syncer.go cli/internal/syncer/syncer_test.go
git commit -m "feat(syncer): add WithScopeToTarget to suppress out-of-scope deletes"
```

---

# Phase 1 — resourceops shared building blocks

## Task 4: Resolver — type routing, remote lookup, capability assertion

**Files:**
- Create: `cli/internal/resourceops/resolver.go`, `cli/internal/resourceops/resolver_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestResolver_FindRemote_ByExternalIDThenRemoteID(t *testing.T) {
	rc := resources.NewRemoteResources()
	rc.Set("event-stream-source", map[string]*resources.RemoteResource{
		"src_1": {ID: "src_1", ExternalID: "my-source", Data: map[string]any{"name": "S"}},
	})
	prov := &fakeProvider{remote: rc}
	r := resourceops.New(&fakeRouter{prov: prov})

	got, err := r.FindRemote(context.Background(), "event-stream-source", "my-source")
	require.NoError(t, err)
	assert.Equal(t, "src_1", got.ID)

	got2, err := r.FindRemote(context.Background(), "event-stream-source", "src_1")
	require.NoError(t, err)
	assert.Equal(t, "src_1", got2.ID)

	_, err = r.FindRemote(context.Background(), "event-stream-source", "ghost")
	assert.ErrorIs(t, err, resourceops.ErrResourceNotFound)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cli/internal/resourceops/... -run TestResolver_FindRemote -v`
Expected: FAIL — package/symbols undefined.

- [ ] **Step 3: Implement**

```go
package resourceops

import (
	"context"
	"errors"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

var (
	ErrResourceNotFound = errors.New("resource not found")
	ErrVerbNotSupported = errors.New("operation not supported for resource type")
)

type Resolver struct{ router provider.TypeRouter }

func New(router provider.TypeRouter) *Resolver { return &Resolver{router: router} }

func (r *Resolver) ProviderFor(resourceType string) (provider.Provider, error) {
	return r.router.ProviderForType(resourceType)
}

// FindRemote loads managed+unmanaged remote resources of resourceType and
// returns the one matching id (external-id first, then remote-id).
func (r *Resolver) FindRemote(ctx context.Context, resourceType, id string) (*resources.RemoteResource, error) {
	all, err := r.loadAll(ctx, resourceType) // map[remoteID]*RemoteResource (Task 5 reuses this)
	if err != nil {
		return nil, err
	}
	for _, res := range all {
		if res.ExternalID == id {
			return res, nil
		}
	}
	if res, ok := all[id]; ok {
		return res, nil
	}
	return nil, fmt.Errorf("%s %q: %w", resourceType, id, ErrResourceNotFound)
}

// ExternalIDSetterFor asserts the optional capability or returns ErrVerbNotSupported.
func (r *Resolver) ExternalIDSetterFor(resourceType string) (provider.ExternalIDSetter, error) {
	p, err := r.ProviderFor(resourceType)
	if err != nil {
		return nil, err
	}
	setter, ok := p.(provider.ExternalIDSetter)
	if !ok {
		return nil, fmt.Errorf("set-external-id on %q: %w", resourceType, ErrVerbNotSupported)
	}
	return setter, nil
}
```

(`loadAll` is implemented in Task 5; for this task stub it to call `LoadResourcesFromRemote` + `GetAll` so the test compiles, then Task 5 extends it with the unmanaged merge.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cli/internal/resourceops/... -run TestResolver_FindRemote -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
make test && make lint
git add cli/internal/resourceops/resolver.go cli/internal/resourceops/resolver_test.go
git commit -m "feat(resourceops): add resolver (type routing, remote lookup, capability gate)"
```

---

## Task 5: Reader — managed+unmanaged merge with de-dup

**Files:**
- Create: `cli/internal/resourceops/reader.go`, `cli/internal/resourceops/reader_test.go`

- [ ] **Step 1: Write the failing test** (assert merge + de-dup rule: managed wins, MANAGED == has external id)

```go
func TestReader_List_MergesAndDedupes(t *testing.T) {
	managed := resources.NewRemoteResources()
	managed.Set("event-stream-source", map[string]*resources.RemoteResource{
		"src_1": {ID: "src_1", ExternalID: "a"},
	})
	importable := resources.NewRemoteResources()
	importable.Set("event-stream-source", map[string]*resources.RemoteResource{
		"src_1": {ID: "src_1"},               // dup of managed → managed wins
		"src_2": {ID: "src_2"},               // unmanaged only
	})
	prov := &fakeProvider{remote: managed, importable: importable}
	rows, err := resourceops.ListRows(context.Background(), prov, "event-stream-source", resourceops.ScopeAll)
	require.NoError(t, err)

	assert.ElementsMatch(t, []resourceops.Row{
		{ExternalID: "a", RemoteID: "src_1", Managed: true},
		{ExternalID: "",  RemoteID: "src_2", Managed: false},
	}, rows)
}
```

- [ ] **Step 2: Run** `go test ./cli/internal/resourceops/... -run TestReader_List_MergesAndDedupes -v` → FAIL.

- [ ] **Step 3: Implement** `reader.go`: `type Scope int` (`ScopeAll`, `ScopeManaged`, `ScopeUnmanaged`); `type Row struct{ ExternalID, RemoteID, Name string; Managed bool }`; `ListRows(ctx, prov, type, scope)` that calls `LoadResourcesFromRemote().GetAll(type)` (managed), and — if the provider implements `UnmanagedRemoteResourceLoader` — `LoadImportable(ctx, namer)` then `GetAll(type)` (unmanaged); merge keyed by RemoteID with managed winning; apply `scope` filter; if `LoadImportable` is absent and scope needs unmanaged, return managed-only and set a `Degraded` flag (surface a one-line note in the command). Name comes from `Data` when present.

- [ ] **Step 4: Run** the test → PASS.

- [ ] **Step 5: Commit**

```bash
make test && make lint
git add cli/internal/resourceops/reader.go cli/internal/resourceops/reader_test.go
git commit -m "feat(resourceops): list rows merging managed+unmanaged with de-dup"
```

---

## Task 6: Single-resource spec materialization (`-o yaml/json`)

**Files:**
- Create/extend: `cli/internal/resourceops/reader.go`, `cli/internal/resourceops/printer.go` (+ tests)

- [ ] **Step 1: Write the failing test** — `SpecYAML(ctx, prov, type, id)` returns YAML that re-parses to the same kind/metadata via `Exporter.FormatForExport`.

```go
func TestReader_SpecYAML_RoundTrips(t *testing.T) {
	prov := newExportableFakeProvider(/* one managed source */)
	out, err := resourceops.SpecYAML(context.Background(), prov, "event-stream-source", "my-source")
	require.NoError(t, err)

	spec, err := specs.New([]byte(out))
	require.NoError(t, err)
	assert.Equal(t, "event-stream-source", spec.Kind) // or the kind the exporter emits
}
```

- [ ] **Step 2: Run** → FAIL.

- [ ] **Step 3: Implement** `SpecYAML`/`SpecJSON`: build a single-entry `RemoteResources`, call `prov.FormatForExport(collection, idNamer, resolver)`, then encode the returned `FormattableEntity` to YAML/JSON in `printer.go`. Reuse `namer` + `resolver` constructors the importer uses (`cli/internal/project/importer/importer.go`). Gate on the provider implementing `Exporter`.

- [ ] **Step 4: Run** → PASS.

- [ ] **Step 5: Commit**

```bash
make test && make lint
git add cli/internal/resourceops/reader.go cli/internal/resourceops/printer.go cli/internal/resourceops/*_test.go
git commit -m "feat(resourceops): materialize re-appliable spec for -o yaml/json"
```

---

# Phase 2 — Read verbs

## Task 7: `get` command (list + single + flags)

**Files:**
- Create: `cli/internal/cmd/get/get.go`
- Modify: `cli/internal/cmd/root.go`
- Test: `cli/internal/cmd/get/get_test.go`

- [ ] **Step 1: Write the failing test** — table-driven over arg parsing/format selection (use a fake deps/resolver injected via a package seam). Assert: `get <type>` → list path; `get <type> <id> -o yaml` → SpecYAML path; unknown type → error mentions valid types; `--managed`+`--unmanaged` together → error.

- [ ] **Step 2: Run** `go test ./cli/internal/cmd/get/... -v` → FAIL.

- [ ] **Step 3: Implement** `NewCmdGet()` mirroring the structure of `cli/internal/cmd/workspace/event-stream-sources.go` (PreRunE builds `app.NewDeps()`, asserts `CompositeProvider()` to `provider.TypeRouter`, constructs `resourceops.Resolver`). Flags: `-o/--output` (default `table`), `--managed`, `--unmanaged` (mutually exclusive), `-l/--selector` (`map[string]string`). Dispatch: 0 extra args → `ListRows` + `printer` (table/json); 1 arg with `-o yaml|json` → `SpecYAML/SpecJSON`; 1 arg table → render single row. Telemetry via `telemetry.TrackCommand("get", err, ...)`.

Register in `root.go`: `rootCmd.AddCommand(get.NewCmdGet())`.

- [ ] **Step 4: Run** `go test ./cli/internal/cmd/get/... -v` → PASS. Then smoke-build: `make build`.

- [ ] **Step 5: Commit**

```bash
make test && make lint
git add cli/internal/cmd/get/ cli/internal/cmd/root.go
git commit -m "feat(cmd): add top-level get verb"
```

---

## Task 8: `describe` command (templated layout of the spec)

**Files:**
- Create: `cli/internal/cmd/describe/describe.go`, `cli/internal/cmd/describe/describe_test.go`
- Modify: `cli/internal/cmd/root.go`

- [ ] **Step 1: Write the failing test** — `describe <type> <id>` renders the `SpecYAML` content through a templated layout helper (assert output contains the spec's key fields under a header). Keep v1 minimal: it's `SpecYAML` piped to `ui` formatting, not a new data path.

- [ ] **Step 2: Run** → FAIL.

- [ ] **Step 3: Implement** `NewCmdDescribe()` (`Args: cobra.ExactArgs(2)`): resolve, call `resourceops.SpecYAML`, render via a small templated layout (reuse `ui.FormattedMap` over the decoded spec map, with a `Managed: yes/no` line). Register in `root.go`.

- [ ] **Step 4: Run** → PASS; `make build`.

- [ ] **Step 5: Commit**

```bash
make test && make lint
git add cli/internal/cmd/describe/ cli/internal/cmd/root.go
git commit -m "feat(cmd): add describe verb (templated spec layout)"
```

---

# Phase 3 — Mutate & claim verbs

## Task 9: `set-external-id` command

**Files:**
- Create: `cli/internal/cmd/setexternalid/set_external_id.go` (+ test)
- Modify: `cli/internal/cmd/root.go`

- [ ] **Step 1: Write the failing test** — `set-external-id <type> <remote-id> <external-id>`: success path calls the resolver's `ExternalIDSetterFor(type).SetExternalID(...)`; a type whose provider lacks the capability returns `ErrVerbNotSupported`.

- [ ] **Step 2: Run** → FAIL.

- [ ] **Step 3: Implement** `NewCmdSetExternalID()` (`Args: cobra.ExactArgs(3)`): resolve setter via `Resolver.ExternalIDSetterFor`, call it, print success. Register in `root.go`.

- [ ] **Step 4: Run** → PASS; `make build`.

- [ ] **Step 5: Commit**

```bash
make test && make lint
git add cli/internal/cmd/setexternalid/ cli/internal/cmd/root.go
git commit -m "feat(cmd): add set-external-id verb"
```

---

## Task 10: `delete` command (managed resources)

**Files:**
- Create: `cli/internal/resourceops/deleter.go` (+ test), `cli/internal/cmd/delete/delete.go` (+ test)
- Modify: `cli/internal/cmd/root.go`

- [ ] **Step 1: Write the failing test (deleter)** — `Delete(ctx, prov, type, id, confirm=true)`:
  - resolves the remote resource (external-id/remote-id),
  - rejects with a clear error if the resolved resource has no external id (unmanaged — nothing to delete via IaC),
  - for a managed resource: builds state via `MapRemoteToState`, calls `LifecycleManager.Delete(ctx, externalID, type, stateOutput)` (assert called with correct args via a mock provider).

- [ ] **Step 2: Run** `go test ./cli/internal/resourceops/... -run TestDeleter -v` → FAIL.

- [ ] **Step 3: Implement** `deleter.go`: resolve → if `ExternalID == ""` return a descriptive error (`%w` = `ErrVerbNotSupported` or a dedicated `ErrUnmanaged`); else `state, _ := prov.MapRemoteToState(singleColl)`, look up `state.GetResource(type+":"+externalID)`, call `prov.Delete(ctx, externalID, type, sr.Output)`. Gate `prov` on `LifecycleManager` (the `Provider` interface already includes it; `account`'s implementation should error — verify and surface cleanly).

- [ ] **Step 4: Implement the command** `NewCmdDelete()` (`Args: cobra.ExactArgs(2)`, `--confirm`): preview the single deletion, prompt unless `--confirm` (reuse `ui` confirm prompt as the existing apply does), call deleter. Register in `root.go`.

- [ ] **Step 5: Run** both tests → PASS; `make build`.

- [ ] **Step 6: Commit**

```bash
make test && make lint
git add cli/internal/resourceops/deleter.go cli/internal/cmd/delete/ cli/internal/cmd/root.go cli/internal/resourceops/deleter_test.go
git commit -m "feat(cmd): add delete verb for managed resources"
```

---

## Task 11: `apply -f` scoped mode on the existing apply command

**Files:**
- Modify: `cli/internal/cmd/project/apply/apply.go`
- Test: `cli/internal/cmd/project/apply/apply_test.go` (add) or an E2E (Task 13)

- [ ] **Step 1: Write the failing test** — flag wiring + mutual exclusion:
  - `-f` and `--location` both set → error before any work.
  - `-f` set → the syncer is constructed with `WithScopeToTarget()` (assert via a seam: factor the syncer-option assembly into a small exported-for-test function `buildSyncOptions(scoped, dryRun, confirm)` returning `[]syncer.Option` and unit-test that the scoped variant includes the scope option). Use a marker/len check or a typed sentinel option list.

- [ ] **Step 2: Run** `go test ./cli/internal/cmd/project/apply/... -v` → FAIL.

- [ ] **Step 3: Implement**
  - Add `var files []string` + `cmd.Flags().StringArrayVarP(&files, "file", "f", nil, "Apply only the resources in these files/dirs (scoped, never deletes). Mutually exclusive with --location.")`.
  - In `PreRunE`/`RunE`: if both `files` and `location` set → `return fmt.Errorf("--file/-f and --location are mutually exclusive")`.
  - When `files` set: collect spec files recursively from each path (reuse the project loader's directory handling; load each path into the fresh project), build the target graph, and assemble syncer options **including** `syncer.WithScopeToTarget()`. Keep `--dry-run`/`--confirm` behavior identical to today.
  - When `location` set: unchanged path.
  - Update the command `Example` and `Long` to document the blast-radius difference (`-f` scoped vs `--location` whole-project reconcile).

- [ ] **Step 4: Run** the test → PASS; `make build`. Manually verify help: `./<built-binary> apply --help` shows `-f` and the warning.

- [ ] **Step 5: Commit**

```bash
make test && make lint
git add cli/internal/cmd/project/apply/apply.go cli/internal/cmd/project/apply/apply_test.go
git commit -m "feat(apply): add scoped, delete-free -f mode alongside --location"
```

---

# Phase 4 — Deprecation & end-to-end

## Task 12: Deprecate the per-noun `list` commands

**Files:**
- Modify: `cli/internal/cmd/workspace/event-stream-sources.go`, `.../accounts.go` (or equivalent), `.../retl-sources.go`

- [ ] **Step 1: Write the failing test** — assert each `list` command sets `Deprecated` (cobra prints it automatically) pointing to `get <type>`.

```go
func TestEventStreamSourcesList_Deprecated(t *testing.T) {
	cmd := findSubcommand(workspace.NewCmdEventStreamSources(), "list")
	require.NotNil(t, cmd)
	assert.Contains(t, cmd.Deprecated, "get event-stream-source")
}
```

- [ ] **Step 2: Run** → FAIL.

- [ ] **Step 3: Implement** — set `cmd.Deprecated = "use 'rudder-cli get <type>' instead"` on each of the three list subcommands. Behavior otherwise unchanged. Note the retl workspace command constructor is `NewCmdRetlSource()` (singular), not `NewCmdRetlSources()`.

- [ ] **Step 4: Run** → PASS.

- [ ] **Step 5: Commit**

```bash
make test && make lint
git add cli/internal/cmd/workspace/
git commit -m "chore(cmd): deprecate per-noun list commands in favor of get"
```

---

## Task 13: E2E — `get -o yaml | apply -f -` round-trip + scoped no-delete

**Files:**
- Create/extend: `cli/tests/` (follow existing E2E harness that drives the built binary via `os/exec`)

- [ ] **Step 1: Write the failing E2E** (against the test workspace / fixture the existing E2E suite uses):
  1. `get event-stream-source <id> -o yaml` → capture stdout.
  2. `apply -f -` (stdin) or `apply -f <tmpfile>` with that YAML → expect **no changes** (round-trip is a no-op when remote already matches).
  3. Apply a modified field via `-f` → expect exactly one update, and assert an **unrelated** managed resource is untouched (scoped no-delete).

- [ ] **Step 2: Run** `make test-e2e` → FAIL (commands/flags not yet wired end-to-end if any gap remains).

- [ ] **Step 3: Fix** any integration gaps surfaced.

- [ ] **Step 4: Run** `make test-e2e` → PASS.

- [ ] **Step 5: Commit**

```bash
make test && make lint && make test-e2e
git add cli/tests/
git commit -m "test(e2e): get -o yaml | apply -f round-trip and scoped no-delete"
```

---

## Final verification

- [ ] `make test-all` passes.
- [ ] `./<binary> get --help`, `describe --help`, `delete --help`, `set-external-id --help`, `apply --help` all render with correct flags and the `-f`/`--location` warning.
- [ ] Manual: `get event-stream-source` lists managed+unmanaged with a MANAGED column; `--managed`/`--unmanaged` filter; `-o yaml` round-trips through `apply -f`.
- [ ] Confirm `account` rejects `delete`/`set-external-id` with a clear `ErrVerbNotSupported` message.

## Notes / known follow-ups (out of scope here)

- Widen beyond beachhead (datagraph, transformations, datacatalog) by implementing/verifying `ExternalIDSetter` per type.
- Evolve `apply` to safe-by-default `+ --prune` (tabled Decision 12).
- Adoption facade `import <type> <id>` (see TABLED doc).
- Short type aliases; rich context-aware `describe`.
