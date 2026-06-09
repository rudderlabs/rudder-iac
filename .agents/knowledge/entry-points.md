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

## RUD-19 — README Onboarding Anchor
- `README.md` is the primary onboarding entry point and must remain accurate for first-run paths (binary and Docker).
- Keep README install instructions aligned with shipped release assets named `rudder-cli_<OS>_<arch>.tar.gz` and the runtime binary `rudder-cli`.
