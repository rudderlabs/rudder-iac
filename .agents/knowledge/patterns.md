# Patterns

> Recurring idioms specific to this repo (error handling, state management,
> retries, logging, DI, request lifecycle).
> Append-only. Agent-authored sections may optionally carry an HTML-comment tag
> (e.g., `<!-- pr:<id> -->`) identifying the writer/PR/run; human-authored
> sections are conventionally left untouched by automated runs.
> Every observed idiom includes a `file:line` reference.

## RUD-2739 Observed Runtime Idioms
<!-- ticket:RUD-2739 -->
- Error context is layered at each boundary with `%w`, so failures preserve causal chains while adding operation intent (for example dependency init, project load, workspace fetch, sync). Ref: `cli/internal/cmd/project/apply/apply.go:49` (`NewCmdApply` PreRunE), `cli/internal/cmd/project/apply/apply.go:80` (`NewCmdApply` RunE), `cli/internal/providers/retl/provider.go:145` (`Provider.ResourceGraph`).
- Domain-specific wrapper errors encode orchestration semantics (`failed` vs `cancelled`) and still participate in `errors.Is/As` via `Unwrap()`. Ref: `cli/pkg/tasker/errors.go:5` (`ErrTaskFailed`), `cli/pkg/tasker/errors.go:18` (`ErrTaskCancelled`), `cli/pkg/tasker/task.go:94` (`job.runTask`).
- Concurrent task execution is dependency-aware: tasks block on dependency completion channels, then short-circuit when upstream failures occur and `continueOnFail` is false. Ref: `cli/pkg/tasker/task.go:36` (`RunTasks`), `cli/pkg/tasker/task.go:94` (`job.runTask`).
- Sync execution follows a stable lifecycle: load remote resources -> map to state -> derive graph -> plan diff -> execute operations -> optional consolidation. Ref: `cli/internal/syncer/syncer.go:121` (`ProjectSyncer.apply`), `cli/internal/syncer/syncer.go:180` (`StateToGraph`), `cli/internal/syncer/syncer.go:484` (`ProjectSyncer.providerOperation`).
- Shared mutable sync state is guarded with `RWMutex` during dereference/mutation so concurrent operations can read references while serializing writes to state updates. Ref: `cli/internal/syncer/syncer.go:22` (`ProjectSyncer.stateMutex`), `cli/internal/syncer/syncer.go:280` (`ProjectSyncer.createOperation`), `cli/internal/syncer/syncer.go:477` (`ProjectSyncer.deleteOperation`).
- Resource state has dual representations: generic map data for serialization plus typed `InputRaw`/`OutputRaw` for provider-specific logic and reflection dereferencing. Ref: `cli/internal/resources/state/state.go:19` (`ResourceState`), `cli/internal/resources/state/state.go:24` (`InputRaw`), `cli/internal/resources/state/state.go:25` (`OutputRaw`).
- Request lifecycle in the API client is centralized (`Do` + `service` helpers): build request with auth headers, execute, decode success payload, and normalize non-2xx into structured `APIError`. Ref: `api/client/client.go:72` (`Client.Do`), `api/client/client.go:94` (`APIError` construction), `api/client/service.go:13` (`service.next`).
- Pagination follows a repeated “first page + `Paging.Next` loop” idiom in service clients, avoiding duplicated transport code by reusing `service.next`. Ref: `api/client/accounts.go:40` (`accounts.List`), `api/client/accounts.go:49` (`accounts.ListAll`), `api/client/service.go:13` (`service.next`).
- Dependency injection uses functional options to swap infrastructure (HTTP client/base URL/user agent, sync reporter/concurrency) without changing constructors. Ref: `api/client/options.go:3` (`Option`), `api/client/options.go:15` (`WithHTTPClient`), `cli/internal/syncer/syncer.go:60` (`WithReporter`), `cli/internal/syncer/syncer.go:69` (`WithConcurrency`).
- Polling-style async workflows use timeout + ticker loops with explicit pending/failed/completed states rather than sleep-based retries. Ref: `cli/internal/providers/retl/sqlmodel/preview.go:13` (`DefaultTimeout`), `cli/internal/providers/retl/sqlmodel/preview.go:25` (`Handler.Preview`), `cli/internal/providers/retl/sqlmodel/preview.go:55` (`time.NewTicker`).
- Logging is package-scoped and structured (`slog` wrapper with fixed attrs), allowing runners/providers to stamp component identity into every record. Ref: `cli/internal/logger/log.go:49` (`logger.New`), `cli/internal/providers/datagraph/validator/runner.go:16` (`validationLog`), `cli/internal/providers/transformations/testorchestrator/runner.go:22` (`testLogger`).

## RUD-2752 — Workspace List Command Construction Pattern
<!-- ticket:RUD-2752 -->
- Workspace list commands follow a consistent orchestration shape: parse flags, defer telemetry tracking, build dependencies via app bootstrap, select a `lister.ListProvider`, choose table/JSON format from `--json`, then execute `List` with resource type and filters.
- Provider-side list behavior is centralized behind provider `List(ctx, resourceType, filters)` methods, creating a stable extension seam for adding new listable workspace resources.
- List result rows are expected to map into common resource keys (`id`, `name`, `type`, `enabled`) with optional domain-specific keys such as `externalId`, aligning with shared lister table/JSON output expectations.

## INT-6489 — Destination CRUD Struct Passthrough
<!-- ticket:INT-6489 -->
- Destination Create and Update copy the input `Destination`, clear only `ID`, and marshal the full struct through the shared service helper.
- Destination Get unmarshals `response.destination` into the same `Destination` type used by write paths.
- Because CRUD uses whole-struct passthrough, new optional public contract fields with `json:",omitempty"` can often be added to the DTO without changing individual service methods.

## DEX-456 — Account Client Thin-Service Placement
<!-- ticket:DEX-456 -->
- `api/client/accounts.go` follows the repo's thin-service pattern: the concrete `accounts` wrapper delegates list/get/create/update/delete behavior to shared `service` helpers. Additive account contract work should stay in that account-specific client surface and its co-located tests, not in `client.go` or the shared transport layer.

## RUD-2860 — Shared Destination DTO Write Scrubbing
<!-- ticket:RUD-2860 -->
- Destination support is centralized in `api/client/destinations.go`; the same `Destination` struct is used as the request and response contract for create, update, get, and list.
- Because destination create/update marshal copied whole structs, adding response-oriented or ownership-metadata fields to `Destination` can affect write payloads unless each write method explicitly clears fields that do not belong on that endpoint.
- `Destinations.Update` is expected to copy the input destination and clear `ExternalID` before marshaling, while `Destinations.SetExternalID` owns the dedicated external-ID write endpoint.
