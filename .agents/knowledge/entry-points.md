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

## DOCS-2434 — README is the install surface
<!-- ticket:DOCS-2434 -->
- `README.md` remains the authoritative installation entry point: it covers macOS, Linux, Docker, and build-from-source setup, and there are no nested installation docs under `docs/` to consult instead.
