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

## RUD-17 — README As Install Entry Surface
<!-- ticket:RUD-17 -->
- `README.md` is the top-level user onboarding entrypoint and is expected to contain all primary install paths in one place: macOS release tarballs (Apple Silicon and Intel), Linux release tarball, Docker image usage, and local source builds.
- Documentation rewrites should preserve concrete install path fidelity (artifact names/commands) or explicitly replace it with equivalent pointers, because users rely on copy-paste-safe setup instructions from this page.
