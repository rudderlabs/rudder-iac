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

## RUD-2963 — Optional Feature API Fallback
<!-- ticket:RUD-2963 -->
- `api/client.APIError.FeatureFlagNotEnabled` classifies an unavailable optional capability only when the response is HTTP 403 and its normalized `APIError.Msg()` contains either the `Flag is not enabled for your account` prefix or the `Feature is not enabled for your account` prefix. Ref: `api/client/common.go` (`APIError.FeatureFlagNotEnabled`).
- Optional-feature list clients degrade this typed condition to a non-nil empty response rather than failing the wider multi-provider operation; DataGraph listing and catalog first-page loading share this behavior, while unrelated errors retain operation-specific wrapping. Ref: `api/client/datagraph/datagraph.go` (`ListDataGraphs`), `api/client/catalog/catalog.go` (`getFirstPage`).

## DEX-545 — Named Pattern Allow/Reject Registry
<!-- ticket:DEX-545 -->
- Named pattern validation is centralized in `cli/internal/provider/rules/funcs/regex.go`; `validate:"pattern=<name>"` consumers should rely on the registry match path rather than calling stored allow regexes directly.
- Pattern registration now supports an optional reject regex: `NewPattern`/`Register` remain allow-only wrappers, while `NewPatternWithReject`/`RegisterWithReject` store the allow regex plus optional reject and make matching fail when the reject regex matches.
- Re-registering a pattern without a reject intentionally clears any stale reject entry for that name, preserving allow-only behavior for existing callers.
- `RegisterWithReject` defensively initializes nil registry maps before writes, so package-local fixtures or future literal registries do not panic when they omit optional maps such as `rejects`.

## DEX-510 — HTTP Destination Definition Conversion
<!-- ticket:DEX-510 -->
- Destination definitions declare `SecretKeys` in local YAML config shape, not upstream API config shape, because `HandlerImpl.ExtractResourcesFromSpec` wraps secrets before API conversion and `MapRemoteToState` wraps after API-to-local conversion; HTTP secrets are `password`, `bearer_token`, and `api_key_value`.
- HTTP event filtering keeps a nested local YAML object (`event_filtering.whitelist` / `event_filtering.blacklist`) and maps it to API `whitelistedEvents` / `blacklistedEvents` plus the `eventFilteringOption` discriminator via `converter.ArrayWithStrings` and `converter.Discriminator`.

## DEX-591 — Destination Apply E2E Catalog Fixtures
<!-- ticket:DEX-591 -->
- Destination apply E2E fixtures use a catalog-style layout: specs live under `cli/tests/testdata/destinations/create/<variation>.yaml` and `cli/tests/testdata/destinations/update/<variation>.yaml`, while shared variables live in `cli/tests/testdata/destinations/destinations.vars.yaml`.
- `TestDestinationsApply` applies the destination create/update directories directly, relying on recursive spec loading so future destination variations can be added without changing test code.
- Expected upstream destination snapshots remain under `cli/tests/testdata/expected/upstream/destinations/{create,update}/destination_<spec.id>` for whole-managed-set comparison.

## DEX-608 — Destination E2E Unverified Fixture Gate
<!-- ticket:DEX-608 -->
- `TestDestinationsApply` still needs `RUDDERSTACK_X_UNVERIFIED_DESTINATIONS=true` while its create/update fixture directories include unverified destination variations such as `attentive_tag`, `http`, and `rs`; S3 is verified/native and does not need that gate.
- Destination apply E2E comments and skip messaging should describe the unverified gate in terms of the current unverified fixture set, not S3 or HTTP alone.

## DEX-661 — Destination Export Empty-Value Pruning
<!-- ticket:DEX-661 -->
- Destination export pruning treats nil, empty strings, and containers holding only empty values as empty, but treats scalar zero values as populated; do not generalize emptiness to scalar zero semantics because `false` and numeric `0` can be meaningful destination config values.
- The destination empty-value helper switches on the concrete types `json.Unmarshal` produces (`nil`, `string`, `[]any`, `map[string]any`) rather than reflecting over arbitrary kinds: local config is always decoded from the API response, so typed slices, maps, and pointers cannot reach it and handling them would be dead code.
- Destination export must prune empty local config values before calling `secret.MaskSecrets`, so API-returned null or empty-string secret keys are dropped before they can become generated variable placeholders in imported YAML.
## DEX-498 — Facebook Conversions E2E Fixture Scope
<!-- ticket:DEX-498 -->
- `facebook_conversions` destination E2E fixtures may document live snapshot capture deferral when live credentials or a disposable destination-enabled workspace are unavailable, but committed fixtures should still make the intended coverage shape visible.
- The initial `facebook_conversions` fixture shape covers a standard cloud-mode variation with dataset/access token, event mappings, PII allow/deny lists, and update mutations for display name, `action_source`, booleans, `test_event_code`, and mappings.

## DEX-505 — Google Pub/Sub E2E Snapshot Deferral
<!-- ticket:DEX-505 -->
- Google Pub/Sub destination E2E coverage may add create/update fixture YAML and dummy credentials without expected upstream snapshot files when no explicitly disposable live destination-enabled stack is available.
- Do not invent hand-written GOOGLEPUBSUB upstream snapshots; defer live `RUN_DESTINATION_E2E=1 TestDestinationsApply` snapshot capture until a safe backend is available, while keeping ungated compile/skip validation as the safe autonomous check.

## DEX-516 — Kinesis E2E Coverage
<!-- ticket:DEX-516 -->
- Never add destination apply fixture YAML under `cli/tests/testdata/destinations/{create,update}` without matching upstream snapshots: `DestinationSnapshotTester` count-checks fetched destinations against the expected file set, so a fixture with no snapshot fails the whole suite before any payload is compared — taking every other destination's coverage down with it.
- Kinesis ships both meaningful auth variations, mirroring the S3 pair: `kinesis` (key-based, `role_based_auth: false`, access keys via `{{ .VAR }}`) and `kinesis-role` (role-based, `role_based_auth: true`, `iam_role_arn`). Both are live-confirmed.
- Snapshots derived as "converter output − `secretKeys` + `schema.json` defaults" matched the live backend exactly on the first gated run, including `useMessageId: false` where the fixture omits it.
