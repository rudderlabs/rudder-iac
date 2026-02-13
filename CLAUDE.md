# rudder-iac

## Project Summary

Go-based CLI tool (`rudder-cli`) for RudderStack Infrastructure-as-Code management. Enables declarative management of RudderStack resources (sources, destinations, connections, tracking plans) via YAML specs with a Terraform-like apply cycle.

## Directory Structure

```
rudder-iac/
├── api/client/          # API clients for RudderStack services
├── cli/
│   ├── cmd/rudder-cli/  # CLI entrypoint
│   ├── internal/
│   │   ├── cmd/         # Cobra commands (auth, project, workspace, etc.)
│   │   ├── provider/    # Provider interface + BaseProvider
│   │   ├── providers/   # Provider implementations
│   │   │   ├── datacatalog/
│   │   │   ├── event-stream/
│   │   │   ├── retl/
│   │   │   └── workspace/
│   │   ├── resources/   # Resource graph, state management
│   │   ├── syncer/      # Apply cycle orchestration
│   │   ├── project/     # Spec loading, formatting
│   │   ├── logger/      # slog wrapper -> ~/.rudder/cli.log
│   │   └── typer/       # Code generation (Kotlin, etc.)
│   └── tests/           # E2E tests (binary interaction)
└── docs/
```

## Architecture Overview

### Provider Pattern

Providers manage resource types through a unified interface (`cli/internal/provider/provider.go`):

| Interface            | Methods                                         |
| -------------------- | ----------------------------------------------- |
| TypeProvider         | `SupportedKinds()`, `SupportedTypes()`          |
| SpecLoader           | `LoadSpec()`, `ParseSpec()`, `ResourceGraph()`  |
| Validator            | `Validate(graph)`                               |
| RemoteResourceLoader | `LoadResourcesFromRemote()`, `LoadImportable()` |
| StateLoader          | `MapRemoteToState()`                            |
| LifecycleManager     | `Create()`, `Update()`, `Delete()`, `Import()`  |
| Exporter             | `FormatForExport()`                             |

Use `BaseProvider` (`cli/internal/provider/baseprovider.go`) for common functionality. Providers delegate to `Handler` implementations per resource type.

### Apply Cycle

Similar to Terraform: loads local specs -> fetches remote state -> computes diff -> applies changes. Core commands: `apply`, `validate`, `import`.

## Code Standards

### Testing

- **Unit tests**: Mandatory. Use `testify` (assert/require) exclusively
- **E2E tests**: Required when changes affect the apply cycle or add new providers. Located in `cli/tests/`, interacts with binary via `os.Exec`
- **Integration tests**: Based on scope for new capabilities
- **Struct comparisons**: Prefer comparing entire structs over field-by-field assertions:

  ```go
  // Preferred
  assert.Equal(t, &DataGraph{ID: "dg-123", Name: "Test"}, result)

  // Avoid
  assert.Equal(t, "dg-123", result.ID)
  assert.Equal(t, "Test", result.Name)
  ```

### Logging

- Use `logger.New("pkg-name")` wrapper (writes to `~/.rudder/cli.log`)
- Log actionable operations only, NOT hot paths
- Include structured attributes for debugging context

### Error Handling

- Wrap errors with context: `fmt.Errorf("making request: %w", err)` (verb/action form)
- Sentinel errors use `Err` prefix: `ErrNotFound`, `ErrUnsupportedKind`
- Log errors at top layer only, not every layer

### Code Style

- SOLID principles
- Concise, well-defined variable names fitting existing ecosystem
- Consistent patterns with existing codebase
- Prefer early `continue`/`return` over `else` to reduce indentation depth (guard clause pattern)
- When declaring 2+ related variables together, prefer a `var` block over individual `:=` assignments for better visual grouping:

  ```go
  // Preferred
  var (
      actualVal  = fieldVal
      actualKind = fieldVal.Kind()
  )

  // Avoid
  actualVal := fieldVal
  actualKind := fieldVal.Kind()
  ```

- **ID Naming Convention**: Use fully capitalized "ID" in identifiers (Go convention for initialisms), e.g., `ExternalID` not `ExternalId`, `WorkspaceID` not `WorkspaceId`

### Comments

- **Focus on "why"**, not "what" - explain the reasoning, not the mechanics
- **Be selective** - not every code section needs comments; only add them where they truly help the reader understand non-obvious decisions or trade-offs
- **Avoid over-commenting** - self-explanatory code doesn't need narration; let the code speak for itself
- Examples of good comments:
  - "Stop early to avoid building a graph from invalid specs" (explains reasoning)
  - "Graph is built once here - single source of truth" (explains architectural decision)
- Examples of unnecessary comments:
  - "Create validation engine" (obvious from `NewValidationEngine()`)
  - "Loop through specs" (obvious from `for path, spec := range p.specs`)

## Standard Workflows

### 1. New CLI Command + Provider Enhancement

Before modifying code:

1. Review Linear ticket and LLD for requirements
2. Check if API client support exists in `api/client/`
3. Identify affected provider(s) in `cli/internal/providers/`
4. Determine if apply cycle is affected (triggers E2E test requirement)

Implementation sequence:

1. Add/modify API client methods if needed
2. Update provider handler or add new handler
3. Wire up CLI command using Cobra in `cli/internal/cmd/`
4. Write unit tests (mandatory)
5. Augment E2E tests if apply cycle affected
6. Run `make test` and `make test-e2e`

### 2. New Provider

Before implementation:

1. Review Linear ticket and LLD
2. Study existing provider as reference (e.g., `cli/internal/providers/retl/`)
3. Understand required resource types and spec kinds

Implementation sequence:

1. Create provider directory under `cli/internal/providers/<name>/`
2. Implement `Provider` interface (embed `BaseProvider` for common functionality)
3. Create handler(s) for each resource type
4. Register provider in the application
5. Add CLI commands if needed
6. Write unit tests for all handlers
7. Add E2E tests for apply cycle integration
8. Consider integration tests based on scope
9. Run full test suite: `make test-all`

### 3. Bug Fixes

Before fixing:

1. Reproduce the issue
2. Identify root cause location (API client, provider, CLI command, syncer)
3. Check if fix affects apply cycle

Implementation:

1. Write failing test that captures the bug
2. Implement fix with minimal scope
3. Verify test passes
4. Run `make test`; run `make test-e2e` if apply cycle affected

### 4. Refactoring / Performance

Before changes:

1. Document current behavior with tests if not covered
2. Identify blast radius of changes
3. Ensure no functional changes mixed with refactoring

Implementation:

1. Make incremental changes
2. Run tests after each significant change
3. Verify no behavioral changes (same test results)
4. Run full test suite before PR

## Build & Test Commands

```bash
make lint               # Lint (must pass before committing)
make build              # Build CLI binary
make test               # Unit tests
make test-e2e           # E2E tests (apply cycle)
make test-it            # Integration tests
make test-all           # All tests
```

Always run `make lint` after making changes and before committing.
