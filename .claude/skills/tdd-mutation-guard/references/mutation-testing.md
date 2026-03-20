# Mutation Testing Guide for Go

## What is Mutation Testing?

Mutation testing evaluates **test quality** by introducing small changes (mutations) to your source code and checking if your tests catch them.

- **Mutant**: A copy of your code with one small change (e.g., `>` becomes `<`)
- **Killed**: Tests fail on the mutant (good - your tests caught it)
- **Survived**: Tests still pass on the mutant (bad - your tests missed it)
- **Mutation Score**: `killed / total mutants * 100`

## Why It Matters

Line coverage tells you **what code runs** during tests. Mutation testing tells you **what code is actually verified**. You can have 100% line coverage with weak assertions that miss real bugs.

```go
// This function has a subtle bug waiting to happen
func IsEligible(age int, score float64) bool {
    return age >= 18 && score > 75.0
}

// This test gives 100% coverage but is WEAK
func TestIsEligible(t *testing.T) {
    assert.True(t, IsEligible(25, 80.0))
}
// Mutant: age >= 18 -> age > 18 ... SURVIVES (no test with age=18)
// Mutant: score > 75.0 -> score >= 75.0 ... SURVIVES (no test with score=75.0)
// Mutant: && -> || ... SURVIVES (no false case tested)
```

## Go Mutation Testing Tools

### Gremlins (Recommended)

Modern, fast mutation testing for Go.

```bash
# Install
go install github.com/go-gremlins/gremlins/cmd/gremlins@latest

# Run on a specific package
gremlins unleash ./cli/internal/providers/datacatalog/...

# Run on changed files only (faster for TDD cycles)
gremlins unleash --diff

# With specific tags
gremlins unleash --tags "unit" ./cli/internal/providers/datacatalog/...
```

### go-mutesting (Alternative)

```bash
go install github.com/zimmski/go-mutesting/cmd/go-mutesting@latest

# Run on a package
go-mutesting ./cli/internal/providers/datacatalog/...
```

## Mutation Operators

Common mutations applied to Go code:

| Operator | Original | Mutant | What It Tests |
|----------|----------|--------|---------------|
| Conditionals boundary | `>=` | `>` | Boundary conditions |
| Negate conditionals | `==` | `!=` | Boolean logic |
| Math operators | `+` | `-` | Arithmetic correctness |
| Return values | `return true` | `return false` | Return path verification |
| Remove statement | `list = append(...)` | ` ` | Side effect verification |
| Conditional replacement | `&&` | `\|\|` | Logical operator correctness |

## Hunting Survivors: Strategies

### 1. Boundary Value Testing

When a boundary mutant survives (`>=` to `>`), add a test at the exact boundary:

```go
// Survivor: age >= 18 mutated to age > 18
// Fix: add boundary test
func TestIsEligible_ExactBoundary(t *testing.T) {
    assert.True(t, IsEligible(18, 80.0))   // exactly 18
    assert.False(t, IsEligible(17, 80.0))  // just below
}
```

### 2. Negation Testing

When a negation mutant survives, add explicit false-case tests:

```go
// Survivor: score > 75.0 mutated to score <= 75.0
// Fix: test both sides of the condition
func TestIsEligible_ScoreBoundary(t *testing.T) {
    assert.True(t, IsEligible(25, 75.1))   // just above
    assert.False(t, IsEligible(25, 75.0))  // exactly at boundary
    assert.False(t, IsEligible(25, 74.9))  // just below
}
```

### 3. Return Value Verification

When a return value mutant survives, ensure you check all return paths:

```go
// Survivor: return nil mutated to return error
// Fix: explicitly assert no error
func TestProcess_Success(t *testing.T) {
    err := Process(validInput)
    require.NoError(t, err)  // Explicitly check, don't ignore
}
```

### 4. Side Effect Verification

When a statement removal survives, verify the side effect matters:

```go
// Survivor: removing `cache.Set(key, val)` doesn't fail any test
// Fix: verify the side effect
func TestProcess_CachesResult(t *testing.T) {
    cache := NewTestCache()
    Process(cache, input)
    assert.Equal(t, expectedVal, cache.Get(key))  // Verify cache was populated
}
```

## Workflow Integration

### Per Red-Green-Refactor Cycle

```
1. RED:      Write failing test
2. GREEN:    Make it pass (minimal code)
3. REFACTOR: Clean up, check CRAP < 8
4. MUTATE:   Run gremlins on changed package
5. KILL:     Write tests for each survivor
6. REPEAT:   Until mutation score >= 90%
```

### Acceptable Survivors

Some mutations are safe to ignore (equivalent mutants):

- Mutating logging strings (doesn't affect behavior)
- Mutating comments or documentation
- Performance-only mutations (e.g., changing buffer size)
- Dead code that should be removed anyway

### Target Mutation Score

| Context | Target |
|---------|--------|
| Core business logic | >= 95% |
| Handlers/CRUD | >= 90% |
| Utility functions | >= 85% |
| Integration glue code | >= 75% |

## Example: Full Cycle

```go
// 1. RED - Write test first
func TestDiscount_SeniorCitizen(t *testing.T) {
    got := CalculateDiscount(65, 100.0)
    assert.Equal(t, 20.0, got)  // 20% discount for age >= 65
}

// 2. GREEN - Minimal implementation
func CalculateDiscount(age int, price float64) float64 {
    if age >= 65 {
        return price * 0.20
    }
    return 0
}

// 3. REFACTOR - Already simple, CRAP = ~2

// 4. MUTATE - Run gremlins
// Survivors found:
//   - age >= 65 -> age > 65 (boundary)
//   - price * 0.20 -> price * 0 (math)
//   - return 0 -> return 1 (return value)

// 5. KILL - Add tests for each survivor
func TestDiscount_BoundaryAge(t *testing.T) {
    assert.Equal(t, 20.0, CalculateDiscount(65, 100.0))  // kills >= -> >
    assert.Equal(t, 0.0, CalculateDiscount(64, 100.0))   // kills >= -> >
}

func TestDiscount_NoDiscount(t *testing.T) {
    assert.Equal(t, 0.0, CalculateDiscount(30, 100.0))   // kills return 0 -> return 1
}

func TestDiscount_CalculationCorrect(t *testing.T) {
    assert.Equal(t, 10.0, CalculateDiscount(70, 50.0))   // kills * 0.20 -> * 0
}
```
