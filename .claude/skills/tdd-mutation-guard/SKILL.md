---
name: tdd-mutation-guard
description: Enforce a disciplined TDD workflow with CRAP score monitoring. Use after writing or modifying Go code to ensure tests are written first (Red-Green-Refactor) and complexity stays low (CRAP < 8). Triggers on every code change cycle.
---

# TDD Mutation Guard

Enforce a rigorous test-driven development cycle augmented with complexity monitoring for Go code in this repository.

## When to Use This Skill

- Writing new functions or methods (tests first)
- Modifying existing code (ensure test coverage is meaningful)
- Refactoring (verify code remains clean and testable)
- Reviewing code quality metrics after changes
- When the user asks to "guard" or "tdd" their code

## The Workflow: Red-Green-Refactor

Every code change follows this cycle. **One test case at a time — never batch multiple tests or write a full test table before implementing.**

### Phase 1: RED — Write Exactly ONE Failing Test

1. Write **one test case** (or one table row) that expresses a single desired behavior.
   - For table-driven tests: add only the next row, not the full table.
   - **STOP. Do not write more tests yet.**
2. Run the test and **confirm it fails for the right reason** — a behavior mismatch, not a compilation error.
   ```bash
   go test ./path/to/package/... -run TestFunctionName -v
   ```
3. If the failure reason is wrong (e.g. panics instead of a wrong value), fix the test until the failure is correct. Do not proceed until RED is confirmed.

> **FORBIDDEN**: Writing multiple test cases or an entire test file before any implementation. Each RED step covers exactly one behavior.

### Phase 2: GREEN — Minimum Code to Pass That One Test

1. Write the **minimum code** necessary to make the one failing test pass.
   - Do not implement logic for cases not yet tested.
   - Do not optimize or generalize yet.
2. Run `go test ./path/to/package/... -run TestFunctionName -v` to confirm green.
   - **STOP. Do not move to the next test until this test is green.**

### Phase 3: REFACTOR — Clean Up, Then Check Complexity

1. Improve code structure while keeping the test green.
2. **Mandatory: run gocyclo on the touched package and report the output.**
   ```bash
   gocyclo -over 8 ./path/to/package/
   ```
   - The threshold is 8, not 5 — it aligns with the CRAP target. A function with complexity ≤ 8 and full coverage has CRAP ≤ 8. Only split when CRAP would breach 8, not merely when complexity exceeds an arbitrary lower bound.
3. **Mandatory: generate a coverage profile and check CRAP for every touched function.**
   ```bash
   go test -coverprofile=coverage.out ./path/to/package/
   go tool cover -func=coverage.out
   ```
   - Calculate CRAP for each function with complexity > 1. If CRAP ≥ 8, refactor before adding the next test.
4. Run `go test ./path/to/package/...` to confirm everything is still green after refactoring.

**Only after a passing REFACTOR phase may you begin the next RED cycle.**

## Quick Reference

| Topic | Reference |
|-------|-----------|
| CRAP score calculation and Go tooling | [references/crap-score.md](references/crap-score.md) |
| TDD patterns for Go | [references/tdd-patterns.md](references/tdd-patterns.md) |

## CRAP Score Target

**Keep CRAP below 8 for every function.**

```
CRAP(m) = complexity(m)^2 * (1 - coverage(m))^3 + complexity(m)
```

| Complexity | Coverage Needed | CRAP Score |
|-----------|----------------|------------|
| 1         | 0%             | 2          |
| 5         | 0%             | 30         |
| 5         | 100%           | 5          |
| 10        | 80%            | 8.0        |
| 10        | 100%           | 10         |
| 20        | 95%            | 20.5       |

**Takeaway**: High complexity demands near-100% coverage. Better to split functions and keep complexity low.

## Implementation Checklist (Per RED-GREEN-REFACTOR Cycle)

Each bullet is a gate — do not advance until it is ticked.

**RED**
- [ ] Exactly one new test case written (one behavior, not a batch)
- [ ] Test confirmed failing for the right behavioral reason (not compile error)

**GREEN**
- [ ] Minimum code written to pass that one test — nothing more
- [ ] `go test ./pkg/... -run TestName -v` output shows PASS

**REFACTOR**
- [ ] `gocyclo -over 8 ./pkg/` run and output reviewed — no function > 8 complexity
- [ ] `go test -coverprofile=coverage.out ./pkg/ && go tool cover -func=coverage.out` run
- [ ] CRAP < 8 calculated for every touched function with complexity > 1
- [ ] `go test ./pkg/...` still green after refactoring

**Before Committing (after all cycles done)**
- [ ] `make lint` passes
- [ ] `make test` passes

## Anti-Patterns to Avoid

1. **Batching tests before implementation** — Writing a full test table or multiple test functions before writing any implementation code. Each cycle covers exactly one behavior.
2. **Skipping the gocyclo check** — The complexity check is mandatory after every REFACTOR, not optional. Report the command output explicitly.
3. **Writing implementation first** — Always test first, even for "obvious" code.
4. **Testing implementation details** — Test behavior, not internals.
5. **Chasing 100% line coverage** — Focus on meaningful assertions over coverage numbers.
6. **Large functions with high coverage** — Split them; CRAP penalizes complexity exponentially.
7. **Mocking everything** — Per repo conventions, prefer real dependencies where feasible.
8. **Stub too generous to produce RED** — When starting from a stub (e.g. `return nil, nil`), the stub must be minimal enough that the first test actually fails. If the test passes on the stub, the stub is doing too much — strip it back further until RED is confirmed before writing any real logic.

## Integration with Repo Workflow

```bash
# Per-cycle commands (run at each phase, not just once at the end)
go test ./path/to/package/... -run TestName -v   # RED: confirm fail; GREEN: confirm pass
gocyclo -over 8 ./path/to/package/              # REFACTOR: mandatory complexity check
go test -coverprofile=coverage.out ./path/to/package/
go tool cover -func=coverage.out                 # REFACTOR: mandatory CRAP check

# Final gate before commit
make lint
make test
make test-e2e   # only if apply cycle is affected
```
