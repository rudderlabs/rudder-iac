# rudder-iac Go Conventions

This file captures repository-specific rules used during implementation and review.

## Project Context

- Project is a Go CLI (`rudder-cli`) with Terraform-like apply cycle.
- Primary areas:
  - `api/client/`
  - `cli/internal/cmd/`
  - `cli/internal/provider/`
  - `cli/internal/providers/`
  - `cli/internal/resources/`
  - `cli/internal/syncer/`

## Mandatory Testing Expectations

- Unit tests are mandatory for behavior changes.
- Use `testify` (`assert`/`require`) consistently.
- Prefer struct-level equality assertions over field-by-field assertions when reasonable.
- Add/update E2E tests in `cli/tests/` when apply-cycle behavior is affected.

## Error Handling

- Wrap errors with action context and `%w`.
  - Example: `fmt.Errorf("loading workspace: %w", err)`
- Sentinel errors use `Err` prefix (for branching semantics).
- Avoid logging the same error at multiple layers; log near user/command boundary.

## Logging

- Use repository logger wrapper (`logger.New("pkg-name")`).
- Prefer structured context fields (resource identifiers, workspace identifiers).
- Do not log secrets or high-volume noise.

## Naming and Style

- Use Go initialism casing: `ID`, `URL`, `API`.
- Prefer guard clauses (early return/continue) over nested `else`.
- Use `var` blocks for tightly related multi-variable declarations.
- Comments should explain intent/tradeoff (`why`) rather than restating code (`what`).

## Provider/Apply-Cycle Awareness

When touching providers or sync logic:
- Confirm resource lifecycle behavior (create/update/delete/import/export).
- Preserve resource graph and state mapping assumptions.
- Validate changes against apply-cycle expectations, not only unit-level behavior.

## Validation Commands

Run the smallest relevant set before finishing:

```bash
make lint
make test
make test-e2e   # when apply cycle paths are impacted
```

Use `make test-all` for broad changes spanning multiple subsystems.
