# CRAP Score Guide for Go

## What is CRAP?

**C**hange **R**isk **A**nti-**P**atterns (CRAP) is a metric that combines cyclomatic complexity and code coverage to estimate how risky a function is to change.

## Formula

```
CRAP(m) = complexity(m)^2 * (1 - coverage(m))^3 + complexity(m)
```

- `complexity(m)`: Cyclomatic complexity of method m
- `coverage(m)`: Test coverage ratio (0.0 to 1.0) of method m

## Why CRAP < 8?

A CRAP score of 8 is the sweet spot where:
- Simple functions (complexity 1-3) pass even with moderate coverage
- Medium functions (complexity 5-7) need high coverage (~80%+)
- Complex functions (complexity 10+) **cannot** achieve CRAP < 8 without near-100% coverage

This naturally forces you to either:
1. Keep functions simple (preferred)
2. Test complex functions exhaustively

## Measuring in Go

### Step 1: Cyclomatic Complexity

Install and run `gocyclo`:

```bash
go install github.com/fzipp/gocyclo/cmd/gocyclo@latest

# Show all functions with complexity > 5
gocyclo -over 5 ./cli/internal/providers/

# Show top 10 most complex functions
gocyclo -top 10 ./cli/internal/
```

### Step 2: Per-Function Coverage

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./cli/internal/providers/datacatalog/...

# View per-function coverage
go tool cover -func=coverage.out

# HTML report for visual inspection
go tool cover -html=coverage.out -o coverage.html
```

### Step 3: Calculate CRAP

For each function, combine complexity and coverage:

```
Example: func ProcessSpec() has complexity=6, coverage=85%
CRAP = 6^2 * (1 - 0.85)^3 + 6
     = 36 * 0.003375 + 6
     = 0.1215 + 6
     = 6.12  (PASS - under 8)
```

```
Example: func ValidateAll() has complexity=12, coverage=70%
CRAP = 12^2 * (1 - 0.70)^3 + 12
     = 144 * 0.027 + 12
     = 3.888 + 12
     = 15.89  (FAIL - way over 8, must refactor)
```

## Refactoring Strategies When CRAP > 8

### 1. Extract Method

Split a complex function into smaller, focused functions:

```go
// BEFORE: complexity=10, hard to fully test
func (h *Handler) ProcessSpec(spec *Spec) error {
    // validation logic (3 branches)
    // transformation logic (4 branches)
    // persistence logic (3 branches)
}

// AFTER: 3 functions, each complexity ~3
func (h *Handler) ProcessSpec(spec *Spec) error {
    if err := h.validateSpec(spec); err != nil {
        return fmt.Errorf("validating: %w", err)
    }
    transformed, err := h.transformSpec(spec)
    if err != nil {
        return fmt.Errorf("transforming: %w", err)
    }
    return h.persistSpec(transformed)
}
```

### 2. Replace Conditional with Polymorphism

Use interfaces/strategy pattern instead of switch/case chains.

### 3. Table-Driven Logic

Replace nested conditionals with lookup tables:

```go
// BEFORE: complexity grows with each case
func severity(level string) int {
    if level == "critical" { return 4 }
    if level == "high" { return 3 }
    // ... more branches
}

// AFTER: complexity=1
var severityMap = map[string]int{
    "critical": 4, "high": 3, "medium": 2, "low": 1,
}

func severity(level string) int {
    return severityMap[level]
}
```

## Thresholds

| CRAP Score | Risk Level | Action |
|-----------|------------|--------|
| 1-4       | Low        | No action needed |
| 5-8       | Acceptable | Monitor, add tests if complexity rises |
| 8-15      | High       | Refactor or increase test coverage |
| 15-30     | Very High  | Must refactor into smaller functions |
| 30+       | Critical   | Rewrite the function immediately |
