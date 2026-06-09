# Conventions

> Coding conventions and naming schemes — things a linter can't catch.
> Append-only. Agent-authored sections may optionally carry an HTML-comment tag
> (e.g., `<!-- pr:<id> -->`) identifying the writer/PR/run; human-authored
> sections are conventionally left untouched by automated runs.

## RUD-2739 Naming And Layout Conventions
<!-- ticket:RUD-2739 -->
- CLI command packages expose constructors as `NewCmd*` and assemble subcommands in a central root wiring file; command `Use` strings mirror product vocabulary (for example `tp`, `data-graphs`, `retl-sources`). Ref: `cli/internal/cmd/root.go` (`init`, `rootCmd`), `cli/internal/cmd/trackingplan/trackingplan.go` (`NewCmdTrackingPlan`), `cli/internal/cmd/project/apply/apply.go` (`NewCmdApply`).
- Command lifecycle is split intentionally across `PreRunE` (dependency/bootstrap/validation) and `RunE` (execution/side effects), with telemetry deferred at command scope. Ref: `cli/internal/cmd/project/apply/apply.go` (`NewCmdApply`).
- Provider architecture is directory-oriented by domain (`datacatalog`, `retl`, `transformations`, `datagraph`), and each provider owns subfolders for `handlers`, `rules`, `model`, and often domain-specific orchestration utilities. Ref: `cli/internal/providers/retl/provider.go` (`Provider`), `cli/internal/providers/transformations/provider.go` (`Provider`).
- Resource naming differentiates local vs remote identity consistently: local `ID`, remote `RemoteID`, externally stable `ExternalID`, and canonical graph key `URN`. Ref: `cli/internal/providers/transformations/model/library.go` (`LibraryResource`, `LibraryState`), `cli/internal/providers/transformations/model/transformation.go` (`TransformationResource`, `TransformationState`), `cli/internal/resources/state/state.go` (`State.AddResource`).
- Handler contracts rely on exported `HandlerMetadata` carrying `ResourceType`/spec metadata; this keeps handler registration declarative and avoids duplicated string constants at call sites. Ref: `cli/internal/providers/transformations/handlers/library/handler.go` (`HandlerMetadata`), `cli/internal/providers/transformations/handlers/transformation/handler.go` (`HandlerMetadata`).
- API package style uses noun structs + plural service types (`Account`/`accounts`, `AccountsPage`) with shared transport primitives in `service`, keeping endpoint files thin and consistent. Ref: `api/client/accounts.go` (`Account`, `accounts`, `AccountsPage`), `api/client/service.go` (`service`).
- Test placement uses co-located unit tests (`*_test.go`) for package behavior plus dedicated cross-package E2E under `cli/tests`, where `TestMain` builds the binary once and scenarios are snapshot-driven. Ref: `cli/tests/main_test.go` (`TestMain`), `cli/tests/README.md` (scenario and snapshot layout), `cli/tests/helpers/file_manager.go` (`StateFileManager`).
- Snapshot file naming in E2E follows URN-derived filenames and splits expected artifacts by concern (`expected/state` vs `expected/upstream`), enabling deterministic diffing of local-state and API-state regressions separately. Ref: `cli/tests/README.md` ("URN-based filename convention", snapshot sections).
- Error-display convention at process boundary distinguishes normal errors from machine-output flows via `SilentError`, so JSON-producing commands can fail with non-zero exits without extra stderr noise. Ref: `cli/internal/cmd/root.go` (`Execute`), `cli/internal/cmd/cmderrors/errors.go` (`SilentError`).

## RUD-19 — Onboarding Docs Must Match Artifact Names
- Documentation simplification is acceptable only if operational onboarding facts remain unchanged.
- Preserve naming consistency across docs and tooling: local build flow uses `make build` and runs `bin/rudder-cli`, while release install snippets must reference `rudder-cli_<OS>_<arch>.tar.gz` assets.
- Preserve first-run Docker guidance in README for config persistence (`-v ~/.rudder:/.rudder`), token injection (`RUDDERSTACK_ACCESS_TOKEN`), and optional local catalog mounting (`-v ~/my-catalog:/catalog` with `-l /catalog`).
