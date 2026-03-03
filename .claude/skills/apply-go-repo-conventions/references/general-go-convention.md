# General Go Convention Decisions

Source guidance:
- https://google.github.io/styleguide/go/decisions
- https://google.github.io/styleguide/go/best-practices

Use this file for important Go decisions that are not already covered by topic-specific references.

## Value vs Pointer Parameters and Returns

Use values by default.
Use pointers when:
- Caller-visible mutation is required.
- Copying is expensive (large structs, hot paths).
- Type is non-copy-safe (for example, embedded synchronization fields).

Return values for small, immutable-like data.
Return pointers for optional/identity-oriented data or expensive-to-copy results.

## Nil vs Empty for Slices/Maps

Avoid APIs that force callers to distinguish nil vs empty unless semantics require it.
- Treat `len(x) == 0` as default emptiness check.
- Choose JSON/YAML encoding behavior intentionally and keep it consistent.

## Receiver Consistency

Use a consistent receiver style for each type.
Choose pointer receivers when methods mutate state, the type is large, or copying is unsafe.
Avoid mixing pointer and value receivers without a clear reason.

## Context Usage

- Put `context.Context` as first parameter on request-scoped operations.
- Do not store context in structs.
- Treat context as cancellation/deadline metadata, not a generic dependency container.

## Generics and Type Aliases

- Use type aliases mainly for compatibility/migration scenarios.
- Use generics when they reduce duplication without hiding behavior.
- Avoid generic abstractions that make call sites and errors harder to understand.

## Testing Signal Quality

Failure messages should quickly show function, input, and got/want mismatch.
Prefer table-driven tests for multiple scenarios.
Use `t.Fatal` only when continuing invalidates remaining assertions.

## Practical Tie-Breakers

When multiple choices are valid:
1. Prefer surrounding package conventions.
2. Prefer lower cognitive load at call sites.
3. Prefer clearer ownership and lifecycle boundaries.
4. Add short comments for non-obvious tradeoffs (why, not what).
