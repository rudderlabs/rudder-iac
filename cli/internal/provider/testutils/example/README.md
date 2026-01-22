# Example Provider

This is a reference implementation of the generic provider framework demonstrating how to build a type-safe, generic resource provider.

## Architecture

The example provider demonstrates the new provider framework with the following features:

- **Type-safe generic handlers**: Uses Go generics to provide compile-time type safety while working with `any` data at runtime
- **In-memory backend**: Simulates a remote system using in-memory maps with remote IDs as keys
- **Two resource types**: Writer and Book with relationships between them
- **Separate subfolders**: Each resource type is organized in its own package

## Structure

```
example/
├── backend/           # In-memory storage backend
│   └── backend.go
├── handlers/
│   ├── writer/       # Writer resource implementation
│   │   ├── types.go
│   │   └── handler.go
│   └── book/         # Book resource implementation
│       ├── types.go
│       └── handler.go
├── testutils/
│   └── testdata/
│       └── examples/ # Example spec files
│           ├── writer_tolkien.yaml
│           └── books_fantasy.yaml
├── provider.go       # Provider composition
└── README.md
```

## Resources

### Writer

A simple resource representing a writer.

**Fields:**
- `name` (string): The writer's name

**Example:**
```yaml
apiVersion: v1
kind: writer
spec:
  id: tolkien
  name: J.R.R. Tolkien
```

### Book

A resource representing a book with a reference to its author. Books are defined as an array within the spec.

**Fields:**
- `books` (array): List of book items, each containing:
  - `id` (string): The book's identifier
  - `name` (string): The book's title
  - `author` (string): URN reference to a Writer resource

**Example:**
```yaml
apiVersion: v1
kind: books
spec:
  books:
    - id: lotr
      name: The Lord of the Rings
      author: '#/writer/common/tolkien'
    - id: hobbit
      name: The Hobbit
      author: '#/writer/common/tolkien'
```

## Implementation Details

### Backend

The backend package provides an in-memory storage system that simulates a remote API:

- Thread-safe operations using `sync.RWMutex`
- Remote IDs are generated automatically (e.g., "remote-1", "remote-2")
- External IDs track which resources are managed by the IaC system
- Separate maps for each resource type

### Handlers

Each resource type has its own handler that implements `HandlerImpl` interface:

1. **Types**: Defines Spec (config), Resource (input), State (output), and Remote (backend) types
2. **Handler**: Implements all CRUD operations and resource lifecycle methods

The handlers use the `BaseHandler` generic type which provides:
- Automatic spec loading and parsing
- Type-safe CRUD operations with runtime type assertions
- Resource graph management
- Import/export functionality

### Provider

The main provider composes all handlers using `BaseProvider`:

```go
func NewProvider(backend *backend.Backend) provider.Provider {
    handlers := []provider.Handler{
        writer.NewHandler(backend),
        book.NewHandler(backend),
    }
    return provider.NewBaseProvider("example", handlers)
}
```

## Key Patterns

### Generic Type Parameters

Each handler uses four type parameters:

- `Spec`: Configuration from YAML files (e.g., `WriterSpec`)
- `Res`: Input/resource data (e.g., `WriterResource`)
- `State`: Output/state data from backend (e.g., `WriterState`)
- `Remote`: Backend response type (e.g., `*RemoteWriter`)

### Resource References

Resources reference each other using URNs with PropertyRef:
- Book → Writer (single reference via `PropertyRef`)

Reference strings in YAML follow the format: `#/{kind}/{group}/{id}` (e.g., `#/writer/common/tolkien`)
URNs use the format: `resourceType:localId` (e.g., `example-writer:tolkien`)

Each handler provides a `Parse{ResourceType}Reference()` function to convert reference strings to URNs.

The framework automatically validates these references during the validation phase.

### Remote Resource Interface

All remote types implement the `RemoteResource` interface:

```go
type RemoteWriter struct {
    *backend.RemoteWriter
}

func (r RemoteWriter) GetResourceMetadata() provider.RemoteResourceMetadata {
    return provider.RemoteResourceMetadata{
        ID:         r.ID,
        ExternalID: r.ExternalID,
        Name:       r.Name,
    }
}
```

Note: The method uses a value receiver and returns a value (not pointer) to enable using `RemoteWriter` (not `*RemoteWriter`) as the type parameter. This allows `BaseHandler` to work generically with any remote resource type.

## Usage

To use this provider in your project:

1. Register the provider in your composite provider
2. Place spec files in your project directory
3. Run IaC commands (plan, apply, etc.)

The provider will:
- Load specs and build a resource graph
- Validate cross-resource references
- Execute CRUD operations against the in-memory backend
- Track state for drift detection
