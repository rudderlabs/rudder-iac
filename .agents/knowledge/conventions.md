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

## DEX-591 — Destination Onboarding E2E Guidance
<!-- ticket:DEX-591 -->
- Destination onboarding E2E guidance should require live snapshot capture when a destination-enabled stack is available, but may document an explicit deferral reason when live stack credentials are unavailable.
- Destination E2E coverage should focus on each destination's meaningful config variations and complement, not duplicate, exhaustive `definition_test.go` validation/unit coverage.

## DEX-623 — Accounts E2E CI Enforcement
<!-- ticket:DEX-623 -->
- `TestAccountsApply` is expected to run under the normal live-backend `make test-e2e` / `make test-all` CI flow; it should not be guarded by `RUN_ACCOUNT_E2E`.
- `TestAccountsImportWorkspace` remains behind `RUN_ACCOUNT_E2E` until the missing `imported/import-manifest.yaml` failure is fixed by a CI-proven follow-up; do not treat accounts apply gating and import-workspace gating as the same policy.
- The accounts E2E tests set their required account feature flags inside the tests via `t.Setenv`, so local CLI subprocesses inherit account support without workflow-level environment gates.
- Outside the intended disposable CI workspace, prefer compile-only E2E validation such as `go test ./cli/tests -run '^$'` for accounts test changes because the live tests call `destroy` against the configured RudderStack workspace.

## DEX-661 — Destination Export Validation Safety
<!-- ticket:DEX-661 -->
- For destination export/import changes, avoid running full `make test-e2e` in non-disposable autonomous environments because destination/account E2E flows can mutate or destroy the configured RudderStack workspace when live credentials are present.
- Prefer focused unit tests, snapshot fixture inspection, and compile-only E2E validation such as `go test ./cli/tests -run '^$'` unless an explicitly disposable live workspace is available.

## DEX-499 — GCS Destination Source-Type Ownership
<!-- ticket:DEX-499 -->
- GCS destination definitions should retain only the CLI-owned event-stream source types: `android`, `android_kotlin`, `ios`, `ios_swift`, `web`, `unity`, `react_native`, `flutter`, `cordova`, and `cloud`.
- Upstream GCS source types `amp`, `warehouse`, and `shopify` are intentionally excluded from the CLI definition until ownership/mapping is explicit.
- Supported GCS connection modes are cloud-only for all retained source types.
## DEX-498 — Facebook Conversions Destination Validation
<!-- ticket:DEX-498 -->
- For `facebook_conversions`, keep config validation aligned with upstream `schema.json` rather than Terraform-only prose; for example, `test_event_code` should keep schema-derived max-length/pattern validation but not add a `required_if` rule solely because `test_destination` is true.

## DEX-505 — Google Pub/Sub Destination Naming And Validation
<!-- ticket:DEX-505 -->
- Google Pub/Sub uses CLI type and definition directory `googlepubsub`, matching `lowercase(APIType)` and the integrations-config directory. Terraform registers the destination under `google_pubsub`, but the CLI's resource identity follows the API type — as it does for every other definition — so the terraform name is not carried over.
- For Google Pub/Sub `project_id`, prefer `required,dynamic_or_pattern=single_line_100` over adding a near-duplicate single-line 1–100 pattern; `required` supplies the non-empty constraint while the shared named pattern rejects line breaks and over-100-character values.

## DEX-516 — Kinesis Auth Validation Shape
<!-- ticket:DEX-516 -->
- Kinesis uses flat local YAML auth fields instead of Terraform's nested `role_based_authentication` / `key_based_authentication` lists: `role_based_auth` is the required mode selector.
- For Kinesis role-based auth, require `iam_role_arn`; for key-based auth, require `access_key_id` / `access_key`. Both come straight from the `schema.json` `allOf` branches.
- Do **not** reject the other mode's keys. The `allOf` branches only ever *add* requirements — neither schema forbids the opposite mode's fields — so an `excluded_if` pair would be stricter than the source of truth. S3 has the same four auth keys and does not exclude either, so adding it to one destination and not the other makes identical configs validate differently.
- The concrete cost of excluding: `iam_role_arn` is not a secret, so it survives an import round trip. A remote destination holding key-based auth plus a stale `iam_role_arn` would import to a spec the CLI immediately rejects.
