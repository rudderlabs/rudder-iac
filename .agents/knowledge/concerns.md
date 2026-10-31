# Concerns

> Technical debt, TODOs, FIXMEs, security concerns, architectural issues.
> Append-only. Agent-authored sections may optionally carry an HTML-comment tag
> (e.g., `<!-- pr:<id> -->`) identifying the writer/PR/run; human-authored
> sections are conventionally left untouched by automated runs.
> Top-5–8 highest-signal items per category, not exhaustive.

## TODO/FIXME/XXX/HACK Density And Clusters
<!-- ticket:RUD-2739 -->
- `api/client/catalog/trackingplans.go:173` TODO indicates `CreateTrackingPlan` is non-idempotent, a high-impact debt item in write-path API behavior.
- `cli/internal/providers/event-stream/source/handler.go:498` FIXME in `(*Handler).Import` fetches all sources then scans for one ID, creating O(n) remote calls/work for a single import.
- `cli/internal/providers/transformations/handlers/library/handler.go:230` and `cli/internal/providers/transformations/handlers/transformation/handler.go:267` TODOs leave export functionality intentionally unimplemented in both handlers.
- `cli/internal/providers/datacatalog/localcatalog/catalog.go:424` TODO in `(*DataCatalog).loadTrackingPlans` defers metadata schema/type handling, risking inconsistent catalog interpretation.
- `cli/internal/providers/datacatalog/localcatalog/tracking_plan.go:140` TODO notes non-recursive include handling, leaving multi-level include graphs partially unsupported.
- `cli/internal/providers/datacatalog/event.go:55` and `cli/internal/providers/datacatalog/event.go:108` duplicate TODOs for categoryID resolution via the new ref mechanism suggest repeated unresolved migration debt.
- `cli/internal/project/migrator/common_migrations.go:24` TODO explicitly defers path-based-to-URN migration, indicating known incomplete migration coverage.

## Security Concerns
<!-- ticket:RUD-2739 -->
- `cli/internal/config/config.go (createConfigFileIfNotExists, updateConfig)` writes config (including `auth.accessToken`) with world-readable defaults (`os.Create`, then `os.WriteFile(..., 0644)`), exposing secrets on multi-user hosts.
- `api/client/client.go (New)` constructs `http.Client{}` without timeout; network hangs can block CLI/API operations indefinitely (availability risk and potential resource exhaustion).
- `cli/internal/cmd/import/retl-source.go (writeSQLToFile)` builds output path using unsanitized `localID` via `fmt.Sprintf("%s/%s.sql", sqlLocation, localID)`, enabling path traversal-style writes if IDs contain separators.
- `cli/internal/cmd/import/retl-source.go (NewCmdRetlSource)` telemetry captures raw `localID`, `remoteID`, `location`, and `sqlLocation`; this can leak workspace structure/resource identifiers to telemetry backends.
- `cli/internal/cmd/telemetry/utils.go (TrackCommand)` blindly copies arbitrary `extras` into telemetry properties, with no allowlist/redaction boundary for future sensitive fields.
- `cli/internal/cmd/root.go (recovery)` prints full panic stack traces in debug mode; stack traces may contain sensitive config/path/token-adjacent context depending on upstream errors.

## Architectural Smells
<!-- ticket:RUD-2739 -->
- `cli/internal/app/dependencies.go (setupProviders, NewDeps)` centralizes provider wiring, feature-flag branching, and client setup in one bootstrap unit; this increases coupling and makes provider extension fragile.
- `cli/internal/provider/composite.go (CompositeProvider)` is a broad orchestrator (dispatch, merge, import/export, remote load, concurrency) with many responsibilities, indicating “god object” drift.
- `cli/internal/provider/provider.go (Provider interface)` is very large (legacy loading, CRUD raw+typed, import/export, validation, graphs), raising implementation burden and violating interface segregation.
- `cli/internal/cmd/root.go (init)` eagerly registers many command trees and hidden-feature toggles in one initializer, making command composition and lifecycle difficult to reason about.
- `cli/internal/project/project.go (LoadLegacySpec flow)` plus provider-specific legacy handlers preserves dual legacy/v1 paths deep in runtime logic, increasing branching complexity and regression surface.
- `api/client/*` service packages repeatedly marshal/unmarshal and perform near-identical request patterns without a strongly typed generic request pipeline, driving duplication and drift risk.

## Obviously Stale Dependencies Or Commented-Out Code Blocks
<!-- ticket:RUD-2739 -->
- `.github/workflows/release.yml` has commented-out permissions (`# packages: write`, `# issues: write`, `# id-token: write`), signaling stale or unfinished release-hardening decisions.
- `cli/internal/providers/transformations/display/result_displayer.go` contains multiple large block-comment “example” sections (`/* ... */`) embedded in production code, reducing signal-to-noise and maintainability.
- `go.mod` contains `replace github.com/rudderlabs/rudder-data-catalog-provider/sdk => ../rudder-data-catalog-provider/sdk`, a local-path override that is fragile/stale-prone outside dev environments.
- `go.mod` includes both `gopkg.in/yaml.v3` and `go.yaml.in/yaml/v3` module lines, indicating dependency overlap that commonly appears during partial migration and can become stale baggage.
- `cli/internal/project/deprecation.go (LegacySpecDeprecationWarning)` warns that v0.1 will be removed “in a future release,” while many `LoadLegacySpec` paths remain active; this suggests prolonged deprecation without clear retirement point.
- `cli/internal/provider/baseprovider.go:121` comment says legacy loading falls back to `LoadSpec` “for now,” which is explicit temporary behavior that can persist stale semantics.

## RUD-2752 — Baseline Test Failures Affecting Validation Signal
<!-- ticket:RUD-2752 -->
- Repository-wide test runs currently include unrelated baseline failures, so full-suite red status is not a reliable signal for regressions in event stream listing changes.
- Known blockers include `typer/generator/core` atomic write failure expectations and `cli/pkg/exp/project` tests that require `RUDDERSTACK_ACCESS_TOKEN` in the environment.

## DEX-846 — Destination Empty Array Defaults Gap
<!-- ticket:DEX-846 -->
- Destination definition defaults intentionally support scalar default tags only; `cli/internal/providers/destination/definitions/defaults.go` rejects slice defaults, and that refusal is asserted in `defaults_test.go`.
- Slack is currently the only registered destination whose schema declares non-scalar top-level defaults, so its empty-array defaults need fixture/snapshot pinning until DEX-704 teaches the defaults/converter layer to preserve empty arrays.
- `converter.ArrayWithStrings` sends an explicitly empty list as `[]` since DEX-1089, as `ArrayWithObjects` already did, so fixing array defaults now needs only defaults-engine support for `default:"[]"` tags.

## DEX-1079 — Raw Decoder Text For Wrong-Typed Destination Envelope Fields
<!-- ticket:DEX-1079 -->
- `decodeDestinationSpec` (`cli/internal/providers/destination/rule_spec_syntax_valid.go`) returns mapstructure's error verbatim at reference `/spec`, so a wrong-typed envelope field such as `definition_version: "1"` reports a multi-line `decoding spec: decoding failed due to the following error(s): 'definition_version' expected type 'int64', got unconvertible type 'string'` pointing at the `spec:` line instead of a per-field message at `/spec/definition_version`. DEX-1079 reported it; that fix covered only `0`.

## DEX-1081 — Destination Decode Error Follow-Ups
<!-- ticket:DEX-1081 -->
- Resolved: DEX-1079's "Raw Decoder Text For Wrong-Typed Destination Envelope Fields". Wrong-typed envelope and config values now report per field through `definitions.DecodeErrors`, e.g. `'definition_version' must be an integer` at `/spec/definition_version`.
- mapstructure truncates a float into an int field without error, even with `WeaklyTypedInput: false`. So `definition_version: 1.5` validates as version 1, and a `*int` config field truncates the same way. rETL rejects fractions with the `rejectFractionalInts` decode hook in `cli/internal/providers/retl/rules/connection/connection_spec_valid.go`; `decodeDestinationSpec` and `definitions.decodeConfig` have no such hook.
- `extractLineText` (`cli/internal/validation/pathindex/pathindex.go`) renders a scalar from `node.Value`, which drops YAML quotes. So `enabled: "true"` shows as `enabled: true` under `'enabled' must be a boolean`. A fix must render from `node.Style`, which is a bitmask: test it with `&`, not `==`.
- An unsupported `connection_mode` source-type key with a non-string value (`foo: true`) reports both `'foo' must be a string` and `source type 'foo' is not supported …` at the same anchor. `sourceTypeKeyResults` runs in `specSyntaxValidRule.Validate`, outside the owner precedence in `RegisteredDefinition.ValidateConfig`.
- `flattenDecodeErrors` is duplicated in `cli/internal/providers/retl/rules/connection/connection_spec_valid.go` and `cli/internal/providers/destination/definitions/validator.go`.
