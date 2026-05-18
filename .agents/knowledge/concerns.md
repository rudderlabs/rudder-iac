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
