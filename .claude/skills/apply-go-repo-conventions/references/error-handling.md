# Error Handling Decisions

Source guidance:
- https://google.github.io/styleguide/go/decisions

Use this reference when designing error flow, wrapping, and branch structure.

## Core Rules

- Return errors instead of panicking for expected runtime failures.
- Wrap errors with action-oriented context using `%w`.
- Keep normal flow left-aligned; handle error branches early.
- Avoid unnecessary `else` after `return` or `continue`.
- Introduce sentinel or typed errors only when callers must branch on category.

## Wrapping Pattern

```go
spec, err := loader.Load(path)
if err != nil {
    return nil, fmt.Errorf("loading spec %q: %w", path, err)
}
```

Avoid low-signal wraps:

```go
// Avoid: message provides no operation context.
return nil, fmt.Errorf("error: %w", err)
```

## Propagation and Logging Boundary

- Wrap at lower layers with operation context.
- Log near command/user boundary once.
- Avoid duplicate logs for the same error at every layer.

## Branching on Error Type

Use sentinel/type checks only for behavior differences:

```go
if errors.Is(err, ErrNotFound) {
    return nil
}
return fmt.Errorf("syncing workspace: %w", err)
```

Do not introduce sentinels when callers only surface the message.

## Control Flow Example

```go
for _, spec := range specs {
    if err := validate(spec); err != nil {
        return fmt.Errorf("validating %s: %w", spec.ID, err)
    }
    if err := apply(spec); err != nil {
        return fmt.Errorf("applying %s: %w", spec.ID, err)
    }
}
```

Error handling stays local; success path remains easy to scan.

## Review Prompts

- Does every propagated error include actionable context?
- Is `%w` used when preserving the original cause matters?
- Are error branches early and readable?
- Are sentinel/type errors justified by caller behavior?
