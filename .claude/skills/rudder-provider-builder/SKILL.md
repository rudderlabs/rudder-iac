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

**CRITICAL**: Resources must **NEVER** reference other resources using direct ID strings. Always use `*resources.PropertyRef` for cross-resource references.

#### Why PropertyRef is Required

Remote IDs are not known until resources are created. Using direct ID strings would cause:
- Create operations to fail (referenced resource doesn't exist yet)
- Invalid references during the apply cycle
- Incorrect dependency tracking

#### How to Use PropertyRef

1. **In Resource struct** - Use `*resources.PropertyRef`, not `string`:
```go
type ModelResource struct {
    ID           string
    DisplayName  string
    DataGraphRef *resources.PropertyRef  // ✅ CORRECT
    // DataGraphID  string                // ❌ WRONG - Don't use direct IDs
}
```

2. **In Spec Parsing** - Create PropertyRef from external ID using URN:
```go
func (h *HandlerImpl) ExtractResourcesFromSpec(path string, spec *ModelSpec) (map[string]*ModelResource, error) {
    // Create URN for the parent resource
    dataGraphURN := resources.URN(spec.DataGraphID, datagraph.HandlerMetadata.ResourceType)

    // Create PropertyRef to the parent resource using URN
    dataGraphRef := datagraph.CreateDataGraphReference(dataGraphURN)

    resource := &ModelResource{
        DataGraphRef: dataGraphRef,  // Store PropertyRef, not ID
    }
    return map[string]*ModelResource{spec.ID: resource}, nil
}
```

**CRITICAL**: Always use `resources.URN()` to construct URNs. Never use `fmt.Sprintf` or string concatenation. Always use `HandlerMetadata.ResourceType` instead of hardcoding resource type strings.

3. **Create Reference Helper** - Provide in parent handler:
```go
// In the parent resource's handler (e.g., datagraph/handler.go)
// IMPORTANT: The urn parameter is a full URN (e.g., "data-graph:my-dg"), not just an ID
func CreateDataGraphReference(urn string) *resources.PropertyRef {
    return handler.CreatePropertyRef[DataGraphState](
        urn,  // Use URN directly, don't construct it here
        func(state *DataGraphState) (string, error) {
            return state.ID, nil  // Extract remote ID from state
        },
    )
}
```

4. **In CRUD Operations** - Access resolved value:
```go
func (h *HandlerImpl) Create(ctx context.Context, data *ModelResource) (*ModelState, error) {
    // PropertyRef is resolved by the syncer before Create is called
    dataGraphRemoteID := data.DataGraphRef.Value  // Resolved remote ID

    req := &CreateModelRequest{
        Name: data.DisplayName,
        // ... other fields
    }
    remote, err := h.client.CreateModel(ctx, dataGraphRemoteID, req)
    // ...
}
```

5. **In MapRemoteToState** - Convert remote ID to PropertyRef:
```go
func (h *HandlerImpl) MapRemoteToState(remote *RemoteModel, urnResolver handler.URNResolver) (*ModelResource, *ModelState, error) {
    // Resolve the parent's URN from its remote ID
    // IMPORTANT: Always use HandlerMetadata.ResourceType, never hardcode resource type strings
    parentURN, err := urnResolver.GetURNByID(datagraph.HandlerMetadata.ResourceType, remote.DataGraphID)
    if err != nil {
        return nil, nil, fmt.Errorf("resolving data graph URN: %w", err)
    }

    // Create PropertyRef to the parent using the resolved URN
    // IMPORTANT: Never parse or split URNs - use them as-is
    parentRef := datagraph.CreateDataGraphReference(parentURN)

    resource := &ModelResource{
        DataGraphRef: parentRef,  // Store PropertyRef
        // ...
    }
    return resource, state, nil
}
```

**CRITICAL RULES**:
- ❌ **NEVER** parse URNs by splitting on `:` or using string manipulation
- ❌ **NEVER** hardcode resource type strings (e.g., `"data-graph"`)
- ✅ **ALWAYS** use `HandlerMetadata.ResourceType` for resource types
- ✅ **ALWAYS** use `resources.URN(id, resourceType)` to construct URNs
- ✅ **ALWAYS** pass URNs directly to Create*Reference functions

#### Complete Reference Example

**Spec (YAML)**:
```yaml
spec:
  id: "my-dg"
  name: "My Data Graph"
  models:
    - id: "user"
      display_name: "User"
      # DataGraphID is set during extraction, not in YAML
```

**Parent Handler (datagraph/handler.go)** - Provide reference helper:
```go
// CreateDataGraphReference creates a PropertyRef for data graph references
// The urn parameter is a full URN (e.g., "data-graph:my-dg")
func CreateDataGraphReference(urn string) *resources.PropertyRef {
    return handler.CreatePropertyRef(
        urn,
        func(state *DataGraphState) (string, error) {
            return state.ID, nil
        },
    )
}
```

**Child Handler (model/handler.go)** - Use references:
```go
// ExtractResourcesFromSpec - Create PropertyRef from parent external ID
func (h *HandlerImpl) ExtractResourcesFromSpec(path string, spec *ModelSpec) (map[string]*ModelResource, error) {
    // Construct URN using resources.URN and HandlerMetadata
    dataGraphURN := resources.URN(spec.DataGraphID, datagraph.HandlerMetadata.ResourceType)
    dataGraphRef := datagraph.CreateDataGraphReference(dataGraphURN)

    resource := &ModelResource{DataGraphRef: dataGraphRef}
    return map[string]*ModelResource{spec.ID: resource}, nil
}

// ValidateResource - Check reference exists using URN
func (h *HandlerImpl) ValidateResource(resource *ModelResource, graph *resources.Graph) error {
    if resource.DataGraphRef == nil {
        return fmt.Errorf("data_graph reference is required")
    }
    if _, exists := graph.GetResource(resource.DataGraphRef.URN); !exists {
        return fmt.Errorf("referenced data graph does not exist")
    }
    return nil
}

// Create - Use resolved value (syncer resolves refs before calling Create)
func (h *HandlerImpl) Create(ctx context.Context, data *ModelResource) (*ModelState, error) {
    dataGraphRemoteID := data.DataGraphRef.Value  // Resolved by syncer
    remote, err := h.client.CreateModel(ctx, dataGraphRemoteID, req)
    // ...
}

// MapRemoteToState - Convert remote ID to PropertyRef using URN resolver
func (h *HandlerImpl) MapRemoteToState(remote *RemoteModel, urnResolver handler.URNResolver) (*ModelResource, *ModelState, error) {
    // Resolve parent's URN from its remote ID
    parentURN, err := urnResolver.GetURNByID(datagraph.HandlerMetadata.ResourceType, remote.DataGraphID)
    if err != nil {
        return nil, nil, fmt.Errorf("resolving data graph URN: %w", err)
    }

    // Use the resolved URN directly - don't parse it!
    parentRef := datagraph.CreateDataGraphReference(parentURN)

    resource := &ModelResource{DataGraphRef: parentRef}
    return resource, state, nil
}
```

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
- Resource types: kebab-case `<provider>-<resource>` (e.g., `data-graph-model`)
- Spec kinds: match YAML `kind` field (e.g., `data-graph`)

### URN Construction and Resource References

**CRITICAL RULES** - Follow these strictly:

1. **Always use `resources.URN()`** to construct URNs:
   ```go
   // ✅ CORRECT
   urn := resources.URN(externalID, HandlerMetadata.ResourceType)

   // ❌ WRONG
   urn := fmt.Sprintf("%s:%s", "data-graph", externalID)
   urn := "data-graph:" + externalID
   ```

2. **Always use `HandlerMetadata.ResourceType`**, never hardcode:
   ```go
   // ✅ CORRECT
   urn := resources.URN(id, datagraph.HandlerMetadata.ResourceType)
   dataGraphURN, err := urnResolver.GetURNByID(datagraph.HandlerMetadata.ResourceType, remoteID)

   // ❌ WRONG
   urn := resources.URN(id, "data-graph")
   dataGraphURN, err := urnResolver.GetURNByID("data-graph", remoteID)
   ```

3. **Never parse or split URNs**:
   ```go
   // ❌ WRONG - Don't parse URNs
   externalID := urn[len("data-graph:"):]
   parts := strings.Split(urn, ":")

   // ✅ CORRECT - Use URNs as opaque identifiers
   ref := datagraph.CreateDataGraphReference(urn)  // Pass URN directly
   ```

4. **Create*Reference functions take URNs**, not external IDs:
   ```go
   // ✅ CORRECT
   urn := resources.URN(externalID, datagraph.HandlerMetadata.ResourceType)
   ref := datagraph.CreateDataGraphReference(urn)

   // ❌ WRONG
   ref := datagraph.CreateDataGraphReference(externalID)
   ```

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
