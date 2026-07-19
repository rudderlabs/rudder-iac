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

## INT-6489 — Destination Version Field Naming
<!-- ticket:INT-6489 -->
- Destination versioning fields follow the public API client convention of exported Go fields with camelCase JSON tags: `version`, `versionInfo`, `status`, `action`, `retirementDate`, and `migrationDocsURL`.
- Optional destination version metadata uses pointer fields for optional date/URL values, matching `omitempty` semantics instead of inventing sentinel zero values.
- `VersionInfo` is an exported `api/client` package type so callers can directly consume destination version status, action, retirement date, and migration docs URL metadata.

## RUD-2899 — DataGraph Rule Documentation Coverage
<!-- ticket:RUD-2899 -->
- When maintaining project-level gatekeeper rule docs/tests, include the DataGraph V1 match pattern (`kind: data-graph`, `version: rudder/v1`) now that DataGraph is always available.
- Rule documentation coverage should stay aligned for rules such as `project/duplicate-urn`, `project/manifest-inline-conflict`, and `project/metadata-syntax-valid`; the ruledoc gatekeeper test helper patterns are the companion surface to keep in sync.
## DEX-456 — SDK ID Initialism Style
<!-- ticket:DEX-456 -->
- The Go SDK uses fully capitalized `ID` initialisms in identifiers and JSON field helpers. New account-related SDK fields and helpers should follow that existing `ID` spelling rather than mixed-case variants.

## DEX-543 — Destination Secret Path Syntax
<!-- ticket:DEX-543 -->
- Destination `SecretKeys` use dot-separated map-key paths for nested secrets; slice-indexed paths are not supported in this convention.
- Empty secret path segments, empty paths, and all-digit segments such as `items.0.key` are invalid destination definition configuration and should be propagated as helper errors rather than silently ignored.
- Export variable names for dotted destination secret paths replace `.` with `_`, so `s3.access_key_id` becomes a suffix like `S3_ACCESS_KEY_ID`; this can collide with a literal top-level key that already contains underscores.
