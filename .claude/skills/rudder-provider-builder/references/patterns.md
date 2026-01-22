# Common Patterns and Best Practices

## Naming Conventions

### ID Naming
Use fully capitalized "ID" in identifiers (Go convention for initialisms):
- ✅ `ExternalID`, `WorkspaceID`, `RemoteID`
- ❌ `ExternalId`, `WorkspaceId`, `RemoteId`

### Resource Types
Format: `<provider>_<resource>`
- Examples: `example_writer`, `data_graph_schema`, `event_stream_source`

### Spec Kinds
Match the YAML `kind` field:
- Examples: `writer`, `books`, `schema`, `source`

### Package Names
Use lowercase with hyphens for directories, underscores for Go package names:
- Directory: `data-graph/`
- Package: `package data_graph`

## Error Handling

### Wrapping Errors
Use verb/action form with context:
```go
// ✅ Good
return nil, fmt.Errorf("creating resource: %w", err)
return nil, fmt.Errorf("resolving author URN: %w", err)

// ❌ Avoid
return nil, fmt.Errorf("error: %w", err)
return nil, err // No context
```

### Sentinel Errors
Use `Err` prefix:
```go
var (
    ErrNotFound          = errors.New("resource not found")
    ErrUnsupportedKind   = errors.New("unsupported spec kind")
)
```

### Logging
Log at top layer only, not every layer:
```go
// In handler method
result, err := h.apiClient.GetResource(ctx, id)
if err != nil {
    // Don't log here, just wrap and return
    return nil, fmt.Errorf("getting resource: %w", err)
}
```

## Resource References

### Single Reference (PropertyRef)

**Spec YAML:**
```yaml
apiVersion: v1
kind: book
spec:
  id: lotr
  name: The Lord of the Rings
  author: '#/writer/common/tolkien'
```

**Spec Type:**
```go
type BookSpec struct {
    ID     string `json:"id"`
    Name   string `json:"name"`
    Author string `json:"author"` // Raw reference string
}
```

**Resource Type:**
```go
type BookResource struct {
    ID     string
    Name   string
    Author *resources.PropertyRef // URN + resolved value
}
```

**In ExtractResourcesFromSpec:**
```go
authorURN, err := writer.ParseWriterReference(spec.Author)
if err != nil {
    return nil, fmt.Errorf("parsing author reference: %w", err)
}

resource := &BookResource{
    ID:     spec.ID,
    Name:   spec.Name,
    Author: writer.CreateWriterReference(authorURN),
}
```

**In CRUD Operations:**
```go
// PropertyRef.Value is populated at runtime with remote ID from state
remoteAuthorID := data.Author.Value

remote, err := h.apiClient.CreateBook(ctx, &client.CreateBookRequest{
    Name:     data.Name,
    AuthorID: remoteAuthorID,
})
```

### Multiple References (Slice of PropertyRefs)

**Spec YAML:**
```yaml
properties:
  - '#/property/common/email'
  - '#/property/common/user_id'
```

**Resource Type:**
```go
type TrackingPlanResource struct {
    Properties []*resources.PropertyRef
}
```

**In ExtractResourcesFromSpec:**
```go
properties := make([]*resources.PropertyRef, 0, len(spec.Properties))
for _, propRef := range spec.Properties {
    propURN, err := property.ParsePropertyReference(propRef)
    if err != nil {
        return nil, fmt.Errorf("parsing property reference: %w", err)
    }
    properties = append(properties, property.CreatePropertyReference(propURN))
}

resource := &TrackingPlanResource{
    Properties: properties,
}
```

**In CRUD Operations:**
```go
propertyIDs := make([]string, 0, len(data.Properties))
for _, prop := range data.Properties {
    propertyIDs = append(propertyIDs, prop.Value) // Resolved remote IDs
}
```

### Nested References (Map of PropertyRefs)

For complex structures like events with properties:

**Resource Type:**
```go
type TrackingPlanResource struct {
    Events map[string]*TrackingPlanEvent
}

type TrackingPlanEvent struct {
    EventURN   *resources.PropertyRef
    Properties map[string]*resources.PropertyRef // property local ID -> PropertyRef
}
```

## Optional Fields

### Pointer vs Value Types

**Use pointers for:**
- Optional fields that may be absent
- Large structs
- Fields that need to distinguish between zero value and absent

```go
type ResourceSpec struct {
    Name        string  `json:"name"`
    Description *string `json:"description,omitempty"` // Optional
}
```

**Use values for:**
- Required fields
- Small primitives
- Fields where zero value is valid

### Handling Nil Pointers

```go
// When mapping from API response
description := ""
if remote.Description != nil {
    description = *remote.Description
}

// When creating API request
var desc *string
if data.Description != "" {
    desc = &data.Description
}
```

## State Management

### Minimal State Pattern
Only store fields needed for Update/Delete operations:

```go
// ✅ Good - minimal
type WriterState struct {
    ID string // Remote ID needed for updates
}

// ❌ Avoid - duplicates resource data
type WriterState struct {
    ID          string
    Name        string // Not needed, already in resource
    CreatedAt   string // Not needed for updates
    UpdatedAt   string // Not needed for updates
}
```

### Exception: Update-Required Fields
Include in state if needed for update logic:

```go
type ResourceState struct {
    ID      string
    Version int    // Required for optimistic locking
    ETag    string // Required for conditional updates
}
```

## Testing

### Struct Comparisons
Compare entire structs, not field-by-field:

```go
// ✅ Preferred
assert.Equal(t, &WriterResource{
    ID:   "tolkien",
    Name: "J.R.R. Tolkien",
}, result)

// ❌ Avoid
assert.Equal(t, "tolkien", result.ID)
assert.Equal(t, "J.R.R. Tolkien", result.Name)
```

### Test Organization
```go
func TestHandler_Create(t *testing.T) {
    tests := []struct {
        name    string
        input   *WriterResource
        want    *WriterState
        wantErr bool
    }{
        {
            name:  "success",
            input: &WriterResource{ID: "tolkien", Name: "J.R.R. Tolkien"},
            want:  &WriterState{ID: "remote-1"},
        },
        {
            name:    "empty name",
            input:   &WriterResource{ID: "tolkien", Name: ""},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Use testify
```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// require: stops test on failure
require.NoError(t, err)
require.NotNil(t, result)

// assert: continues test on failure
assert.Equal(t, expected, actual)
assert.Nil(t, err)
```

## Logging

### Logger Initialization
```go
import "github.com/rudderlabs/rudder-iac/cli/internal/logger"

var log = logger.New("packagename")
```

### When to Log
- ✅ Actionable operations (create, update, delete started/completed)
- ✅ Unexpected conditions that aren't errors
- ✅ Important state transitions
- ❌ Hot paths (spec parsing, graph building for each resource)
- ❌ Every API call
- ❌ Validation checks

### Structured Logging
```go
log.Info("creating resource", "id", resourceID, "type", resourceType)
log.Debug("resource details", "resource", resource) // Debug only
log.Error("operation failed", "error", err, "id", resourceID)
```

## Validation Patterns

### Two-Phase Validation

**Phase 1: Spec Validation (ValidateSpec)**
- Validates YAML structure
- Checks required fields
- Validates formats and constraints
- No access to resource graph

```go
func (h *HandlerImpl) ValidateSpec(spec *BookSpec) error {
    if len(spec.Books) == 0 {
        return fmt.Errorf("at least one book is required")
    }
    for i, book := range spec.Books {
        if book.ID == "" {
            return fmt.Errorf("book[%d]: id is required", i)
        }
        if book.Author == "" {
            return fmt.Errorf("book[%d]: author is required", i)
        }
    }
    return nil
}
```

**Phase 2: Resource Validation (ValidateResource)**
- Validates business logic
- Checks cross-resource references
- Validates dependencies exist in graph

```go
func (h *HandlerImpl) ValidateResource(resource *BookResource, graph *resources.Graph) error {
    if resource.Author == nil {
        return fmt.Errorf("author is required")
    }

    // Validate reference exists
    if _, exists := graph.GetResource(resource.Author.URN); !exists {
        return fmt.Errorf("author %s does not exist", resource.Author.URN)
    }

    return nil
}
```

## Export Patterns

### Choosing Export Strategy

**Use MultiSpecExportStrategy when:**
- Each resource is independent
- Resources are large or complex
- Users typically manage resources individually
- Example: Writers (one writer per file)

**Use SingleSpecExportStrategy when:**
- Resources are grouped logically
- Resources are small and numerous
- Resources share context or configuration
- Example: Books (all books in one file)

### Reference Resolution in Export

```go
// In MapRemoteToSpec for SingleSpecExportStrategy
func (h *HandlerImpl) MapRemoteToSpec(
    data map[string]*RemoteBook,
    resolver resolver.ReferenceResolver,
) (*export.SpecExportData[BookSpec], error) {
    items := make([]BookItem, 0, len(data))

    for externalID, remote := range data {
        // Resolve remote ID to spec reference format
        authorRef, err := resolver.ResolveToReference(
            writerResourceType,
            remote.AuthorID,
        )
        if err != nil {
            return nil, fmt.Errorf("resolving author: %w", err)
        }

        items = append(items, BookItem{
            ID:     externalID,
            Name:   remote.Name,
            Author: authorRef, // '#/writer/common/tolkien'
        })
    }

    return &export.SpecExportData[BookSpec]{
        Data:         &BookSpec{Books: items},
        RelativePath: "books/books.yaml",
    }, nil
}
```

## Concurrency Considerations

### API Client Concurrency
BaseProvider calls handlers concurrently. Ensure:
- API clients are thread-safe
- Shared state is protected with mutexes
- No race conditions in handler fields

### Handler State
Handlers should be stateless or use proper synchronization:

```go
// ✅ Stateless - preferred
type HandlerImpl struct {
    apiClient *client.Client // Shared, read-only reference
}

// ⚠️ Stateful - needs synchronization
type HandlerImpl struct {
    mu    sync.RWMutex
    cache map[string]*Resource
}
```

## Common Pitfalls

### 1. Pointer vs Value Receivers
```go
// ❌ Wrong - prevents using RemoteWriter as type parameter
func (r *RemoteWriter) Metadata() handler.RemoteResourceMetadata

// ✅ Correct - enables RemoteWriter (not *RemoteWriter) as parameter
func (r RemoteWriter) Metadata() handler.RemoteResourceMetadata
```

### 2. Nil PropertyRef Checks
```go
// Always check PropertyRef is not nil before accessing
if data.Author != nil {
    authorID := data.Author.Value
}
```

### 3. Missing External ID Filter
```go
// ✅ In LoadRemoteResources - filter by external ID
if remote.ExternalID != "" {
    result = append(result, remote)
}

// ✅ In LoadImportableResources - don't filter
result = append(result, remote)
```

### 4. Forgetting to Wrap API Client Type
```go
// ❌ Wrong - API type doesn't implement RemoteResource
return []*client.Writer{...}, nil

// ✅ Correct - wrap in type that implements RemoteResource
return []*model.RemoteWriter{{Writer: clientWriter}}, nil
```

### 5. Not Handling Reference Resolution Errors
```go
// ❌ Wrong - ignoring errors
authorURN, _ := writer.ParseWriterReference(spec.Author)

// ✅ Correct - handle errors
authorURN, err := writer.ParseWriterReference(spec.Author)
if err != nil {
    return nil, fmt.Errorf("parsing author reference: %w", err)
}
```
