---
name: rudder-provider-builder
description: Guide for creating new providers in the rudder-iac project using the latest BaseProvider and BaseHandler framework. Use when building providers for new RudderStack services (data-graph, profiles, etc.) or creating providers from scratch with custom resources. Implements type-safe handlers with generics, resource lifecycle management (CRUD), remote state sync, and import/export functionality.
---

# RudderStack Provider Builder

Create new providers for the rudder-iac project following the latest framework patterns.

## When to Use This Skill

- Creating a provider for a new RudderStack service (e.g., data-graph, profiles)
- Building a provider from scratch with custom resources
- Adding resource type handlers to an existing provider
- Understanding the BaseProvider and BaseHandler framework

## Quick Reference

- **Framework Overview**: See [framework-overview.md](references/framework-overview.md) for architecture, interfaces, and data flow
- **Handler Implementation**: See [handler-guide.md](references/handler-guide.md) for step-by-step handler creation
- **Patterns & Best Practices**: See [patterns.md](references/patterns.md) for common patterns, testing, and pitfalls
- **Example Provider**: `cli/internal/provider/testutils/example/` - complete working implementation

## Prerequisites

Before starting, ensure:

1. **API Client Exists**: The RudderStack API client should be implemented in `api/client/<service>/` with CRUD methods
2. **Spec Format Defined**: YAML spec structure for resources is documented or understood
3. **Resource Model Clear**: Understanding of resource data model, relationships, and remote API responses

## Workflow: Create a New Provider

### Step 1: Create Provider Directory Structure

Create the following structure under `cli/internal/providers/`:

```
<provider-name>/
├── provider.go           # Provider composition
├── handlers/             # Resource type handlers
│   └── <resource>/
│       └── handler.go    # Handler implementation
├── model/                # Shared data types (optional)
│   └── <resource>.go     # Spec, Res, State, Remote types
└── README.md             # Provider documentation
```

### Step 2: Implement Handlers

For each resource type:

1. **Define Data Types** in `model/<resource>.go`:
   - `Spec`: YAML configuration structure
   - `Res`: Resource data (input for CRUD)
   - `State`: Output state (computed fields only)
   - `Remote`: API response wrapper implementing `RemoteResource`

2. **Implement Handler** in `handlers/<resource>/handler.go`:
   - Create `HandlerImpl` struct with API client
   - Implement all `HandlerImpl[Spec, Res, State, Remote]` methods
   - Choose export strategy (MultiSpec or SingleSpec)
   - Add reference helper functions if resource is referenceable

**Detailed guide**: See [handler-guide.md](references/handler-guide.md)

### Step 3: Compose Provider

Create `provider.go`:

```go
package <provider>

import (
    "github.com/rudderlabs/rudder-iac/cli/internal/provider"
    // Import handler packages
)

// Provider wraps the base provider to provide a concrete type for dependency injection
type Provider struct {
    provider.Provider
}

func NewProvider(apiClient *client.Client) *Provider {
    handlers := []provider.Handler{
        resource1.NewHandler(apiClient),
        resource2.NewHandler(apiClient),
    }

    return &Provider{
        Provider: provider.NewBaseProvider(handlers),
    }
}
```

**Important**: Always create a concrete `Provider` struct that embeds `provider.Provider`. This allows the provider to be used as a specific type in the `Providers` struct in `dependencies.go`, enabling access to provider-specific methods if needed.

### Step 4: Register Provider

Register the provider in `cli/internal/app/dependencies.go`:

1. **Add to Providers struct** - Use pointer to concrete type, not interface:
```go
type Providers struct {
    // ... existing providers ...
    MyProvider *myprovider.Provider  // Concrete type, not provider.Provider
}
```

2. **Initialize in setupProviders()** - Conditionally if using experimental flag:
```go
func setupProviders(c *client.Client) (*Providers, error) {
    cfg := config.GetConfig()

    // ... existing provider setup ...

    providers := &Providers{
        // ... existing providers ...
    }

    // Initialize conditionally if experimental
    if cfg.ExperimentalFlags.MyFeature {
        myClient := myclient.New(c)
        providers.MyProvider = myprovider.NewProvider(myClient)
    }

    return providers, nil
}
```

3. **Add to composite provider** - In `NewDeps()`:
```go
providers := map[string]provider.Provider{
    "datacatalog": p.DataCatalog,
    // ... existing providers ...
}

// Add conditionally if needed
if cfg.ExperimentalFlags.MyFeature {
    providers["myprovider"] = p.MyProvider
}
```

**Important**: The `Providers` struct uses concrete pointer types (`*myprovider.Provider`) instead of the interface (`provider.Provider`) to allow access to provider-specific methods if needed.

### Step 5: Testing

1. **Unit Tests**: Test all handler methods using testify
   - Use struct comparisons, not field-by-field assertions
   - Test validation, CRUD operations, state mapping, export
2. **E2E Tests**: Add tests in `cli/tests/` if provider affects apply cycle
3. **Integration Tests**: Based on scope and complexity

Run tests:

```bash
make test                    # Unit tests
make test-e2e               # E2E tests
make test-all               # All tests
```

## Key Framework Concepts

### BaseProvider

Aggregates handlers and routes operations:

- Discovers supported kinds/types from handlers
- Routes specs to appropriate handler by kind
- Merges results from all handlers
- Implements full Provider interface

### BaseHandler (Generic)

Type-safe wrapper around HandlerImpl:

- Type parameters: `[Spec, Res, State, Remote]`
- Handles type conversions (`any` ↔ concrete types)
- Manages resource collection and graph building
- Delegates business logic to HandlerImpl

### HandlerImpl Interface

Resource-specific implementation providing:

- Spec lifecycle (parse, validate, extract resources)
- Resource validation in graph context
- Remote operations (load managed/importable resources)
- State mapping (Remote → Res + State)
- CRUD operations (Create, Update, Import, Delete)
- Export formatting (resources → YAML specs)

**Complete reference**: See [framework-overview.md](references/framework-overview.md)

## Common Patterns

### Resource References

Resources reference each other using URNs and PropertyRefs:

1. **In Spec**: `'#/writer/common/tolkien'` (reference string)
2. **Parse to URN**: `writer.ParseWriterReference(ref)` → `"example_writer:tolkien"`
3. **Create PropertyRef**: `writer.CreateWriterReference(urn)`
4. **Use in CRUD**: `authorRemoteID := data.Author.Value` (resolved at runtime)

### Export Strategies

- **MultiSpecExportStrategy**: One resource per file (independent resources)
- **SingleSpecExportStrategy**: Multiple resources in one file (grouped resources)

### Validation

Two-phase validation:

1. **ValidateSpec**: YAML structure, required fields (no graph access)
2. **ValidateResource**: Business logic, cross-resource references (with graph)

**More patterns**: See [patterns.md](references/patterns.md)

## Implementation Checklist

### Provider Setup

- [ ] Provider directory created under `cli/internal/providers/`
- [ ] API client dependency available
- [ ] README.md documents provider purpose and resources

### Per Handler

- [ ] Data types defined (Spec, Res, State, Remote)
- [ ] Remote implements `RemoteResource` with value receiver
- [ ] HandlerMetadata configured (ResourceType, SpecKind, SpecMetadataName)
- [ ] All HandlerImpl methods implemented:
  - [ ] Metadata, NewSpec, ValidateSpec
  - [ ] ExtractResourcesFromSpec, ValidateResource
  - [ ] LoadRemoteResources, LoadImportableResources
  - [ ] MapRemoteToState
  - [ ] Create, Update, Import, Delete
  - [ ] FormatForExport (MapRemoteToSpec)
- [ ] Reference helpers if resource is referenceable:
  - [ ] Parse{Resource}Reference
  - [ ] Create{Resource}Reference
- [ ] Export strategy chosen and implemented
- [ ] Unit tests cover all methods

### Integration

- [ ] Handler registered in provider.go
- [ ] Provider registered in application
- [ ] E2E tests added/updated if apply cycle affected
- [ ] All tests passing (`make test-all`)

## Code Standards

### Naming

- Use fully capitalized "ID" (not "Id"): `ExternalID`, `WorkspaceID`
- Resource types: kebab-case `<provider>-<resource>` (e.g., `data-graph-schema`)
- Spec kinds: match YAML `kind` field (e.g., `schema`)

### Error Handling

- Wrap errors with context: `fmt.Errorf("creating resource: %w", err)`
- Use sentinel errors with `Err` prefix: `ErrNotFound`
- Log errors at top layer only

### Testing

- Prefer struct comparisons over field-by-field
- Use testify (assert/require)
- Test validation, CRUD, state mapping, export

### Logging

- Use `logger.New("packagename")`
- Log actionable operations only, NOT hot paths
- Include structured attributes

**Complete standards**: See CLAUDE.md in project root

## Troubleshooting

### Common Issues

**"cannot use RemoteWriter as type parameter"**

- Ensure `Metadata()` uses value receiver: `func (r RemoteWriter) Metadata()`

**"nil pointer dereference in PropertyRef"**

- Always check: `if data.Author != nil { ... }`

**"reference not found in graph"**

- Validate references exist in `ValidateResource`

**"external ID not set on import"**

- Set external ID in `Import()` method

**"tests comparing pointers fail"**

- Compare struct values, not pointers: `assert.Equal(t, expected, *actual)`

## Additional Resources

- **Example Provider**: `cli/internal/provider/testutils/example/` (Writer and Book resources)
- **Provider Interface**: `cli/internal/provider/provider.go`
- **BaseProvider**: `cli/internal/provider/baseprovider.go`
- **Handler Interfaces**: `cli/internal/provider/handler/handler.go`
- **BaseHandler**: `cli/internal/provider/handler/basehandler.go`
- **Project Guidelines**: `CLAUDE.md` in project root

## Tips for Success

1. **Start with the example provider** - Read and understand the Writer and Book handlers
2. **Define types first** - Clear type definitions make implementation easier
3. **Implement incrementally** - One handler method at a time, test as you go
4. **Use reference resolution** - Don't hardcode relationships, use PropertyRefs
5. **Keep state minimal** - Only store fields needed for Update/Delete
6. **Test thoroughly** - Unit tests catch issues early
7. **Follow conventions** - Consistency with existing code matters

## Next Steps

After creating your provider:

1. **Documentation**: Update provider README with resource types and examples
2. **CLI Integration**: Add commands if needed (`cli/internal/cmd/`)
3. **Spec Examples**: Create example YAML specs for testing
4. **User Documentation**: Document spec format and usage patterns
5. **Migration**: If migrating from old provider, create migration plan
