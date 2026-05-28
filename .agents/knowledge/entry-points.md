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

## DAW-3454 -- README First, Keep Commands Accurate
- `README.md` is the primary top-level onboarding entry point, so it should be read before other docs when orienting in the repo.
- The install instructions in `README.md` are release-sensitive and include concrete artifact URLs and image references; edits should preserve command correctness while trimming unnecessary prose.
