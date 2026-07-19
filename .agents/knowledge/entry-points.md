# Entry points

> Key entry-point files: read these first to orient in this repo.
> Append-only. Agent-authored sections may optionally carry an HTML-comment tag
> (e.g., `<!-- pr:<id> -->`) identifying the writer/PR/run; human-authored
> sections are conventionally left untouched by automated runs.

## 2026-05-18 Initial Orientation Set
<!-- ticket:RUD-2739 -->
Read these first (3-6 files):
- `README.md`: install, execution modes (binary/docker), and top-level usage expectations for `rudder-cli`.
- `CLAUDE.md`: maintainer architecture guide (provider pattern, apply cycle, directory map, and expected test workflows).
- `cli/cmd/rudder-cli/main.go`: primary executable entrypoint; sets version and calls `cmd.Execute()`.
- `cli/internal/cmd/root.go`: top-level Cobra router; registers command groups (`auth`, `trackingplan`, `workspace`, `apply`, `validate`, `destroy`, `migrate`, `import`, `transformations`, `typer`, `datagraph`) and initialization hooks.
- `Makefile`: canonical developer entrypoints for build/test/lint/docker and typer validation flows.
- `api/client/client.go`: API client root used by CLI/provider layers to communicate with RudderStack services.

## RUD-2752 — Workspace Event Stream Sources CLI Entry
<!-- ticket:RUD-2752 -->
Read these when working on event stream source listing:
- `cli/internal/cmd/workspace/event-stream-sources.go`: workspace command entry for `workspace event-stream-sources list`.
- `cli/internal/providers/event-stream/provider.go`: provider dispatch point for list requests.
- `cli/internal/providers/event-stream/source/handler.go`: source-level list mapping and row shaping.

## INT-6489 — Destination API Contract Entry
<!-- ticket:INT-6489 -->
Read this first when working on destination API contract fields or destination versioning:
- `api/client/destinations.go`: shared destination DTO, exported `VersionInfo` type, and destination CRUD service methods.

## INT-6671 — RETL Connection Sync Behaviour Contract Entry
<!-- ticket:INT-6671 -->
Read this first when working on RETL connection `syncBehaviour` request/response modeling:
- `api/client/retl/connection_types.go`: contains both `CreateRETLConnectionRequest` for create payloads and `RETLConnection` for response payloads; keep the create request's optional `syncBehaviour` distinct from the response model's resolved value.
- `api/client/retl/connections_test.go`: existing create-request tests directly assign `SyncBehaviour`, so pointer optionality changes need matching test fixture updates.

## DEX-456 — Account API Contract Entry
<!-- ticket:DEX-456 -->
Read these first when working on account API contract fields:
- `api/client/accounts.go`: account DTOs and thin account service wrapper.
- `api/client/accounts_test.go`: existing co-located account client tests; extend this file rather than introducing a separate account test harness.

## RUD-2860 — Destination External ID Contract Entry
<!-- ticket:RUD-2860 -->
Read this first when working on destination external IDs or destination ownership metadata:
- `api/client/destinations.go`: centralized destination DTO and CRUD transport, including the dedicated external-ID setter and the update-path scrubbing rule.

## DEX-545 — Named Pattern Validation Entry
<!-- ticket:DEX-545 -->
Read these first when working on named `validate:"pattern=<name>"` behavior:
- `cli/internal/provider/rules/funcs/regex.go`: shared pattern registry and validator function, including allow/reject matching.
- `cli/internal/provider/rules/funcs/regex_test.go`: co-located coverage for `NewPattern`, `NewPatternWithReject`, and validator behavior.
