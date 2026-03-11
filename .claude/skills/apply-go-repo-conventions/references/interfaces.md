# Interface Decisions

Source guidance:
- https://google.github.io/styleguide/go/decisions

Use this reference when deciding interface shape, location, and constructor/API signatures.

## Core Rules

- Define interfaces at the consumer boundary, not in provider packages by default.
- Keep interfaces small and behavior-focused.
- Accept interfaces where polymorphism is needed; return concrete types from constructors.
- Do not use pointers to interfaces.
- Avoid creating an interface when only one concrete implementation exists and no test seam is needed.

## Design Heuristics

Choose an interface when at least one applies:
- Multiple implementations exist today or are expected soon.
- A caller needs a small behavioral contract, not full type surface.
- Tests benefit from a focused fake/stub.

Prefer concrete types when:
- The abstraction adds indirection without clear benefit.
- The implementation details are part of the intended API.
- The code path is simple and stable.

## Examples

### 1) Define Interface at Point of Use

```go
// Consumer package: syncer
type stateLoader interface {
    Load(ctx context.Context) (*State, error)
}

type Engine struct {
    loader stateLoader
}
```

```go
// Provider package returns concrete implementation.
func NewStateLoader(client *Client) *StateLoader {
    return &StateLoader{client: client}
}
```

Why: the consumer controls the minimal contract; provider stays concrete and discoverable.

### 2) Avoid Premature Provider-Side Interfaces

```go
// Avoid in provider package when there is only one implementation.
type ClientAPI interface {
    FetchWorkspace(ctx context.Context, id string) (*Workspace, error)
}
```

Prefer exposing the concrete client and letting consumers define local interfaces as needed.

### 3) Avoid Pointer to Interface

```go
// Good
func run(ctx context.Context, loader stateLoader) error { ... }

// Avoid
func run(ctx context.Context, loader *stateLoader) error { ... }
```

Interface values are already references to dynamic concrete values.

## Review Prompts

- Is the interface defined where it is consumed?
- Is each method necessary for the caller?
- Would a concrete type reduce complexity without harming testability?
- Is any pointer-to-interface usage present?
