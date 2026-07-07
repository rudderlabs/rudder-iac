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

## DEX-456 — Account API Contract Entry
<!-- ticket:DEX-456 -->
Read these first when working on account API contract fields:
- `api/client/accounts.go`: account DTOs and thin account service wrapper.
- `api/client/accounts_test.go`: existing co-located account client tests; extend this file rather than introducing a separate account test harness.

## RUD-2860 — Destination External ID Contract Entry
<!-- ticket:RUD-2860 -->
Read this first when working on destination external IDs or destination ownership metadata:
- `api/client/destinations.go`: centralized destination DTO and CRUD transport, including the dedicated external-ID setter and the update-path scrubbing rule.
