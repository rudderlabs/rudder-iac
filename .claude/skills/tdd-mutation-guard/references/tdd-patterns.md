# TDD Patterns for Go

## Core Principle: Red-Green-Refactor

```
RED    -> Write a test that fails (expresses desired behavior)
GREEN  -> Write minimum code to pass
REFACTOR -> Improve structure, keep tests green
```

Never skip RED. Even for "trivial" code, the test documents intent.

## Pattern 1: Table-Driven TDD

Write the test table first, then implement:

```go
// RED: Define all cases upfront
func TestParseResourceType(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected ResourceType
        wantErr  bool
    }{
        {name: "valid source", input: "source", expected: TypeSource},
        {name: "valid destination", input: "destination", expected: TypeDestination},
        {name: "empty string", input: "", wantErr: true},
        {name: "unknown type", input: "foobar", wantErr: true},
        {name: "case insensitive", input: "SOURCE", expected: TypeSource},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseResourceType(tt.input)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.expected, got)
        })
    }
}

// GREEN: Implement ParseResourceType to pass all cases
// REFACTOR: Simplify if needed, ensure CRAP < 8
```

## Pattern 2: Behavior-Driven Test Names

Name tests by behavior, not implementation. This applies equally to top-level function names and `t.Run` subtest names.

```go
// Good: describes observable behavior
func TestCatalogValidate_StructuralValidation(t *testing.T) {}
func TestCatalogValidate_RegisteredCompleteness(t *testing.T) {}

// Bad: copies spec rule numbers or internal labels
func TestCatalogValidate_Rule1(t *testing.T) {}
func TestCatalogValidate_Rule4(t *testing.T) {}
```

The same rule applies to `t.Run` subtest names — they must read as plain-English sentences describing what the system does:

```go
// Good
{name: "empty rule_id is rejected"}
{name: "perfect 1:1 mapping produces no errors"}

// Bad
{name: "rule_id check"}
{name: "test case 1"}
```

Never copy labels from a spec document (Rule 1, Rule 2, Step A) into test names. Spec labels are an implementation artefact — they change and lose meaning outside the document. Behaviour names are stable.

## Pattern 3: Arrange-Act-Assert

Keep test structure consistent:

```go
func TestHandler_Create_SetsExternalID(t *testing.T) {
    // Arrange
    handler := NewHandler(mockClient)
    resource := &Resource{Name: "test"}

    // Act
    state, err := handler.Create(context.Background(), resource)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, &State{ID: "generated-id"}, state)
}
```

## Pattern 4: Test Error Paths First

Error paths are often where bugs hide. Write error tests before happy path:

```go
// Write these FIRST
func TestProcess_NilInput(t *testing.T) {
    _, err := Process(nil)
    require.ErrorIs(t, err, ErrNilInput)
}

func TestProcess_InvalidFormat(t *testing.T) {
    _, err := Process(&Input{Format: "bad"})
    require.ErrorIs(t, err, ErrInvalidFormat)
}

// Then write the happy path
func TestProcess_ValidInput(t *testing.T) {
    result, err := Process(&Input{Format: "json", Data: "..."})
    require.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

## Pattern 5: Struct Comparisons (Repo Convention)

Per CLAUDE.md, prefer comparing entire structs:

```go
// Preferred - catches unexpected field changes
assert.Equal(t, &Resource{
    Name:       "test",
    ExternalID: "ext-123",
    Type:       TypeSource,
}, result)

// Avoid - misses fields you forgot to check
assert.Equal(t, "test", result.Name)
assert.Equal(t, "ext-123", result.ExternalID)
```

## Pattern 6: Test One Thing Per Test

Each test should verify one behavior. If a test needs multiple assertions on different behaviors, split it:

```go
// Good: focused tests
func TestValidate_RejectsEmptyName(t *testing.T) { ... }
func TestValidate_RejectsDuplicateID(t *testing.T) { ... }
func TestValidate_AcceptsMinimalSpec(t *testing.T) { ... }

// Bad: testing multiple behaviors
func TestValidate(t *testing.T) {
    // tests empty name AND duplicate ID AND minimal spec...
}
```

## Pattern 7: Test Boundaries Explicitly

For mutation testing survival, always test at boundaries:

```go
func TestIsAdult(t *testing.T) {
    tests := []struct {
        name string
        age  int
        want bool
    }{
        // Boundary values - these kill boundary mutants
        {name: "exactly 18", age: 18, want: true},
        {name: "just below", age: 17, want: false},

        // Representative values
        {name: "clearly adult", age: 30, want: true},
        {name: "clearly minor", age: 10, want: false},

        // Edge cases
        {name: "zero", age: 0, want: false},
        {name: "negative", age: -1, want: false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            assert.Equal(t, tt.want, IsAdult(tt.age))
        })
    }
}
```

## Pattern 8: Incremental Complexity (One Test at a Time — Mandatory)

Each RED-GREEN-REFACTOR cycle covers **exactly one behavior**. Do not pre-write a full test table and then implement everything. Add one row (or one test function), confirm it fails, write the minimum code, confirm it passes, run gocyclo + CRAP check, then add the next row.

```
Cycle 1: RED(empty input → error) → GREEN → REFACTOR+gocyclo
Cycle 2: RED(single valid item → result) → GREEN → REFACTOR+gocyclo
Cycle 3: RED(multiple items → all results) → GREEN → REFACTOR+gocyclo
Cycle 4: RED(duplicate items → error) → GREEN → REFACTOR+gocyclo
Cycle 5: RED(items with references → resolved) → GREEN → REFACTOR+gocyclo
```

**Correct**: Add one table row, run tests to see it fail, implement, run tests to see it pass, run gocyclo, advance.

**Wrong**: Write all table rows upfront, then implement everything in one go.

For table-driven tests this means the table is built row-by-row across cycles. The full table only exists after all cycles are complete.

**This rule holds even when switching to a new group or rule within the same test function.** Starting a new `t.Run` sub-table for a different validation rule is not a license to add all its rows at once — the one-row-per-cycle discipline applies identically within each group.

## Pattern 9: When to Use `t.Run` vs Separate Top-Level Functions

Use **one top-level function with `t.Run` subtests** when multiple scenarios exercise the same behaviour contract (same function, same outcome type, different inputs):

```go
// Good: all scenarios test the same contract — structural field validation
func TestCatalogValidate_StructuralValidation(t *testing.T) {
    tests := []struct{ name string; rule ResolvedRule; wantField string }{
        {name: "empty rule_id is rejected",         wantField: "rule_id"},
        {name: "invalid phase value is rejected",   wantField: "phase"},
        {name: "empty applies_to slice is rejected", wantField: "applies_to"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) { ... })
    }
}
```

Use **separate top-level functions** only when the behaviour contract itself is fundamentally different — a different function under test, a different class of outcome, or a different setup that cannot share a common assertion loop.

```go
// Good: genuinely different contracts — separate functions
func TestCatalogValidate_StructuralValidation(t *testing.T) { ... }  // field errors
func TestCatalogValidate_AppliesToCoverage(t *testing.T)    { ... }  // coverage errors
func TestCatalogValidate_RegisteredCompleteness(t *testing.T) { ... } // registry errors
```

**Decision rule:** if the assertion loop body would be identical (or near-identical) across functions, it belongs in one function with `t.Run`. If each function needs a meaningfully different assertion loop, keep them separate.

## When NOT to TDD

Some code genuinely doesn't benefit from test-first:
- Pure boilerplate (struct definitions, interface implementations with no logic)
- Direct delegation (method that just calls another method)
- Generated code

But if there's ANY conditional logic, TDD it.
