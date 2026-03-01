# Naming and Repetition Decisions (Google Go Style Guide)

Use this reference when selecting identifiers for local variables, exported APIs, and package-level symbols.

## Variable Names

Choose names by balancing clarity and brevity:
- Names should be short enough to scan quickly.
- Names should still communicate purpose.
- The tighter the scope, the shorter the name can be.

### Scope-Based Guidance

- Very small scope (single short block): one-word names are usually enough.
- Medium scope (multiple branches or loops): use slightly more explicit names.
- Large scope (whole function or struct fields): be fully explicit.

### Practical Rules

- Do not repeat type information in the variable name when the type already provides context.
- Avoid context that is already obvious from package, receiver, or function name.
- Use one-letter names only for standard narrow cases (loop indices, short receiver names).
- Keep naming consistent for the same concept across a function and related files.

### Guide-Inspired Examples

```go
// Good: type already tells us these are users.
var users []User

// Avoid: "userSlice" repeats the type context.
var userSlice []User
```

```go
func parseLimit(limitRaw string) (int, error) {
    // Good: name is explicit before parsing, concise after parsing.
    limit, err := strconv.Atoi(limitRaw)
    if err != nil {
        return 0, fmt.Errorf("parsing limit: %w", err)
    }
    return limit, nil
}
```

```go
func (p *Provider) validateSpecs(specs []*Spec) error {
    // Good: short name in a tight loop.
    for _, s := range specs {
        if err := p.validateSpec(s); err != nil {
            return err
        }
    }
    return nil
}
```

## Repetition

Avoid repeating the same concept across package names, type names, and function names.

### Package and Exported Symbol Naming

Use package names to carry broad context; symbol names should add only new information.

Guide examples:
- Prefer `widget.New(...)` over `widget.NewWidget(...)`.
- Prefer `db.Load(...)` over `db.LoadFromDatabase(...)`.

### Function and Variable Naming

Avoid repeating words already present in the function name, receiver type, or surrounding domain context.

```go
func (c *Client) fetchWorkspace(ctx context.Context, id string) (*Workspace, error) {
    // Good: local name does not repeat "workspace" everywhere.
    ws, err := c.api.GetWorkspace(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("fetching workspace: %w", err)
    }
    return ws, nil
}
```

```go
// Good package/symbol pairing: provider.New(...)
// Avoid repetitive pairing: provider.NewProvider(...)
```

### Review Prompts

When reviewing code, ask:
- Can the package name carry context so symbol names become shorter?
- Does any identifier repeat words that are already obvious from type/function/package?
- If a name is shortened, is meaning still unambiguous within its scope?
