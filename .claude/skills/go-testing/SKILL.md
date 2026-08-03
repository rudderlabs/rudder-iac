---
name: go-testing
description: Opinionated Go testing conventions for writing high-quality, readable, and maintainable tests. Use this skill whenever writing, reviewing, or refactoring Go test files (.go files with _test.go suffix). Triggers on any request involving Go unit tests, table-driven tests, testify usage, test helpers, test structure, or Go testing best practices. Also use when the user asks to "write tests for", "add test coverage", "refactor tests", or "review tests" in a Go codebase.
---

# Opinionated Go Testing Conventions

Every test MUST follow all 9 principles. They are not suggestions — they are the standard.

## Principles

**1. Assert Full Maps, Not Individual Keys**
Build a `map[string]interface{}` of all actual fields and compare in one `assert.Equal`. Never scatter per-field assertions. For slices of structs, convert each element to a map; use `assert.ElementsMatch` when order is non-deterministic.

**2. Parallel by Default**
Call `t.Parallel()` at the top of every test function and every `t.Run` subtest. Each subtest gets its own setup — never share mutable state. Use `t.Cleanup()` for teardown. If a test cannot be parallel, add a comment explaining why.

**3. `require` vs `assert`**
- `require` — stops the test immediately. Use for preconditions/gates where failure makes the rest meaningless or causes a panic (`require.NoError`, `require.NotNil`, `require.Len`).
- `assert` — continues the test. Use for the actual verifications so all failures surface in one run.

**4. Feature-Level Tests, Behavior Subtests**
One `Test` function per feature/unit. One `t.Run` per behavior. Never one `Test` per scenario, never mix unrelated features in one `Test`.
```
1 feature/unit → 1 Test function
1 behavior     → 1 t.Run subtest
```

**5. Table-Driven Tests for 2+ Scenarios**
Express multiple scenarios as a `[]struct{ name string; ... }` slice. Each entry becomes a `t.Run` subtest. For a single scenario, a plain subtest is fine.

**6. Helpers Accept `testing.TB` and Call `t.Helper()`**
Every test helper: (1) accepts `testing.TB`, (2) calls `t.Helper()` first, (3) uses `require` internally — never returns errors, (4) registers cleanup via `t.Cleanup()`.

**7. Test Behavior, Not Implementation**
Assert on return values and observable state changes. Avoid asserting call counts or internal method order. Use simple spies only when verifying an external side-effect is unavoidable.

**8. Golden Files for Complex Outputs**
When expected output exceeds ~10 lines or is structured (JSON/YAML), store it under `testdata/` and use an `-update` flag to regenerate. Normalize non-deterministic fields (timestamps, UUIDs) before comparison. Use `assert.JSONEq` for JSON golden files.

**9. Naming That Reads Like Documentation**
- Top-level: `Test<Type>_<Method>` (e.g. `TestCartService_Checkout`)
- Subtests: `snake_case` behavior descriptions (e.g. `returns_error_when_email_empty`)
- Patterns: `returns_<what>_when_<condition>`, `rejects_<input>_with_<error>`, `defaults_to_<value>_when_<field>_missing`

---

## Quick Reference Checklist

- [ ] Full maps in one assertion, not individual keys
- [ ] `t.Parallel()` on every test and subtest (or justified comment)
- [ ] `require` for gates, `assert` for verifications
- [ ] One `Test` per feature, one `t.Run` per behavior
- [ ] Table-driven for 2+ scenarios
- [ ] Helpers: `testing.TB`, `t.Helper()`, `t.Cleanup()`
- [ ] Assert observable behavior, not internal calls
- [ ] Golden files for complex outputs with `-update` flag
- [ ] Subtest names are `snake_case` behavior descriptions
