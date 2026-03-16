# PRO-5163: Data Graph Validation Commands — Progress, Discoveries & Decisions

## Implementation Status

All 7 plan steps are **complete**. Lint, build, and tests all pass.

### Files Created
- `cli/internal/providers/datagraph/validations/results.go` — result types and status enums
- `cli/internal/providers/datagraph/validations/planner.go` — builds validation plans (all/modified/single modes)
- `cli/internal/providers/datagraph/validations/planner_test.go`
- `cli/internal/providers/datagraph/validations/runner.go` — orchestrates: remote state → plan → resolve IDs → run tasks
- `cli/internal/providers/datagraph/validations/runner_test.go`
- `cli/internal/providers/datagraph/validations/tasker.go` — concurrent validation execution (concurrency=4)
- `cli/internal/providers/datagraph/validations/tasker_test.go`
- `cli/internal/providers/datagraph/display/validation.go` — terminal + JSON output formatting
- `cli/internal/providers/datagraph/display/validation_test.go`
- `cli/internal/cmd/datagraph/datagraph.go` — parent command group
- `cli/internal/cmd/datagraph/validate/validate.go` — validate command with flag validation
- `cli/internal/cmd/datagraph/validate/validate_test.go`

### Files Modified
- `api/client/datagraph/types.go` — validation request/response types
- `api/client/datagraph/models.go` — `ValidateModel` interface method + impl
- `api/client/datagraph/models_test.go` — API client tests
- `api/client/datagraph/relationships.go` — `ValidateRelationship` interface method + impl
- `api/client/datagraph/relationships_test.go` — API client tests
- `cli/internal/providers/datagraph/provider.go` — added `client` field + `Client()` accessor
- `cli/internal/providers/datagraph/testutils/mocks.go` — mock methods for validate
- `cli/internal/cmd/root.go` — registered `data-graph` command

## Discoveries

### Backend Schema Mismatch (causes HTTP 500 on model validation)

**Root cause:** The Public API (`rudder-api`) validation schemas include a `name` field in `ValidateEntityModelSchema` and `ValidateEventModelSchema`. The Config Backend (`rudder-config-backend`) does NOT accept `name` — its `parseValidateModelBody()` lists known fields as:
- Entity: `['type', 'tableRef', 'primaryId', 'root']`
- Event: `['type', 'tableRef', 'timestamp']`

When `name` is sent, the Config Backend throws `BadRequestError("Unsupported fields: name")`. This error is thrown outside the controller's try-catch block, so it surfaces as a **500 Internal Server Error** instead of a 400.

**CLI-side fix (applied):** Removed `Name` field from `ValidateModelRequest`. The `name` field is resource metadata, not relevant for warehouse validation.

**Backend fix needed (tracked in PRO-5435):** Remove `name` from both validation schemas in `rudder-api/src/modules/data-graph/validation/router.ts`. The Config Backend is correct to reject it.

### Relationship validation works correctly
The relationship validation endpoint returns proper validation issues (e.g., "Table does not exist in the warehouse"). Only model validation was broken due to the `name` field issue.

## Design Decisions

### ValidationDisplayer accepts `io.Writer`
Changed from using `ui.Printf` globally to accepting an `io.Writer` parameter. This makes tests simpler — they pass a `bytes.Buffer` directly instead of using `ui.SetWriter`/`ui.RestoreWriter`. The command passes `cmd.OutOrStdout()`.

### Provider exposes `Client()` method
Added `Client() dgClient.DataGraphClient` to `Provider` so the validate command can access the API client for the runner. The client was already stored internally but wasn't exposed.

### Rule names colored by severity
Issue rule names in terminal output are colored red for errors and yellow for warnings, making it easier to scan results visually.

## PR Review Decisions (2026-03-16)

Decisions made during PR #459 review discussion with fxenik:

### Consolidate into single `validator/` package
All validation components (planner, runner, tasker, results, displayers) move into `cli/internal/providers/datagraph/validator/`. Eliminates cross-package interfaces between the old `validations/` and `display/` packages. Follows the pattern of keeping tightly coupled components together.

### Extract validator orchestrator (importer.go pattern)
Business logic extracted from the cobra command into a standalone `Validate()` function, following the same pattern as `cli/internal/project/importer/importer.go`. The command only collects flags/args and calls the validator. Dependencies injected via interfaces.

### Split displayers into terminal and JSON implementations
Single `ValidationDisplayer` split into `TerminalDisplayer` and `JSONDisplayer` in separate files with a shared `Displayer` interface in `displayer.go`. The validator orchestrator picks the right displayer directly — no factory function.

### Dynamic terminal width for terminal displayer
Uses existing `ui.GetTerminalWidth()` instead of hardcoded `lineWidth = 80`. Minimum width guard of 60. `statusColumn` derived proportionally (75% of width). Scoped to validation displayer only.

### Standalone plan functions with PlannerFunc signature
Replaced `Planner` struct with standalone `PlanAll`, `PlanModified`, `PlanSingle` functions conforming to a common `PlannerFunc` type. Shared `resourcesToUnits` helper eliminates duplicated unit construction. All return `*ValidationPlan` for consistency.

### Mode as sealed interface
Replaced `Mode` int enum + separate `resourceType, targetID string` params with sealed interface: `ModeAll{}`, `ModeModified{}`, `ModeSingle{ResourceType, TargetID}`. Each mode carries only its relevant data, extensible without signature changes.

### Rename `ValidationResults` → `ValidationReport`
Better communicates the purpose of the aggregate result type.

### Add URN to ValidationUnit
`ValidationUnit` gets a `URN` field populated during plan building. Used as task key in tasker (replacing `resourceType:ID` format) and in error messages.

### Per-task spinners using `ui.TaskReporter`
Replaced single spinner with per-validation-task spinners using `ui.TaskReporter` (Bubble Tea), same pattern as syncer's `ProgressSyncReporter`. Reporter callbacks hooked inside the `tasker.RunTasks` closure. No-op reporter for `--json` mode.

### `SilentError` for `--json` exit code behavior
New `SilentError` type in `cli/internal/cmd/` wraps errors that should produce a non-zero exit code but no stderr output. Used when `--json` + validation failures: JSON already written to stdout, exit code 1, no redundant error message. `Execute()` in root.go checks for `SilentError` before calling `ui.PrintError()`. Documented with framework-level comments (not feature-specific).

### Hidden data-graphs command behind experimental flag
`data-graphs` command set to `Hidden: true`, unhidden in `initConfig()` when `data_graph` experimental flag is enabled, matching the existing `debugCmd`/`experimentalCmd` pattern.

### Full-output test assertions for displayers
Tests use `assert.Equal` against complete expected buffer contents instead of `assert.Contains` for individual strings, validating padding/indentation correctness.

### Full-content test assertions for planner
Planner tests compare full `[]*ValidationUnit` struct contents instead of just counting by type.

## What's Left

- **Implement PR review feedback** — all 12 threads, consolidated into validator package refactoring
- **Manual testing** once backend schema fix (PRO-5435) is deployed
- **E2E tests** are not required per plan (validation doesn't affect the apply cycle)
