# Provider Framework Overview

## Architecture

The rudder-iac provider framework uses a generic, type-safe architecture that separates concerns:

```
Provider (BaseProvider)
  └─> Handlers (BaseHandler[Spec, Res, State, Remote])
       └─> HandlerImpl (resource-specific logic)
```

### Key Components

#### 1. Provider Interface

Located in `cli/internal/provider/provider.go`, defines all capabilities:

- **TypeProvider**: Declares supported kinds and types
- **SpecLoader**: Loads YAML specs into resource graphs
- **Validator**: Validates resource graphs
- **RemoteResourceLoader**: Fetches resources from remote API
- **StateLoader**: Converts remote resources to state
- **LifecycleManager**: CRUD operations (Create, Update, Delete, Import)
- **Exporter**: Formats resources for export to YAML
- **SpecMigrator**: Migrates specs between versions

#### 2. BaseProvider

Located in `cli/internal/provider/baseprovider.go`. Implements the Provider interface by:
- Aggregating multiple handlers
- Routing operations to appropriate handler based on resource type/kind
- Merging results from all handlers

**Usage:**
```go
func NewProvider(apiClient *client.Client) provider.Provider {
    handlers := []provider.Handler{
        resource1.NewHandler(apiClient),
        resource2.NewHandler(apiClient),
    }
    return provider.NewBaseProvider(handlers)
}
```

#### 3. Handler Interface

Located in `cli/internal/provider/baseprovider.go`, each handler manages one resource type:

```go
type Handler interface {
    ResourceType() string
    SpecKind() string
    LoadSpec(path string, s *specs.Spec) error
    Validate(graph *resources.Graph) error
    ParseSpec(path string, s *specs.Spec) (*specs.ParsedSpec, error)
    Resources() ([]*resources.Resource, error)
    Create(ctx context.Context, data any) (any, error)
    Update(ctx context.Context, newData any, oldData any, oldState any) (any, error)
    Delete(ctx context.Context, ID string, oldData any, oldState any) error
    Import(ctx context.Context, data any, remoteId string) (any, error)
    LoadResourcesFromRemote(ctx context.Context) (*resources.RemoteResources, error)
    MapRemoteToState(collection *resources.RemoteResources) (*state.State, error)
    LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.RemoteResources, error)
    FormatForExport(...) ([]writer.FormattableEntity, error)
}
```

#### 4. BaseHandler (Generic)

Located in `cli/internal/provider/handler/basehandler.go`. Type-safe wrapper that:
- Handles type conversions between `any` and concrete types
- Manages resource collection and graph building
- Delegates to HandlerImpl for resource-specific logic

**Type Parameters:**
- `Spec`: YAML configuration structure
- `Res`: Resource data (input/configuration)
- `State`: State data (output from remote API)
- `Remote`: Remote API response type (must implement `RemoteResource`)

#### 5. HandlerImpl Interface

Located in `cli/internal/provider/handler/handler.go`. Resource-specific implementation:

```go
type HandlerImpl[Spec any, Res any, State any, Remote RemoteResource] interface {
    Metadata() HandlerMetadata
    NewSpec() *Spec
    ValidateSpec(spec *Spec) error
    ExtractResourcesFromSpec(path string, spec *Spec) (map[string]*Res, error)
    ValidateResource(resource *Res, graph *resources.Graph) error
    LoadRemoteResources(ctx context.Context) ([]*Remote, error)
    LoadImportableResources(ctx context.Context) ([]*Remote, error)
    MapRemoteToState(remote *Remote, urnResolver URNResolver) (*Res, *State, error)
    Create(ctx context.Context, data *Res) (*State, error)
    Update(ctx context.Context, newData *Res, oldData *Res, oldState *State) (*State, error)
    Import(ctx context.Context, data *Res, remoteId string) (*State, error)
    Delete(ctx context.Context, id string, oldData *Res, oldState *State) error
    FormatForExport(...) ([]writer.FormattableEntity, error)
}
```

## Data Flow

### Spec Loading → Resource Graph
1. User creates YAML spec files
2. `LoadSpec()` → `NewSpec()` creates empty spec
3. YAML unmarshaled into Spec
4. `ValidateSpec()` validates configuration
5. `ExtractResourcesFromSpec()` creates Res objects
6. `ValidateResource()` validates each resource in graph context
7. Resources added to graph with dependencies

### Remote State Sync
1. `LoadRemoteResources()` fetches managed resources from API
2. `MapRemoteToState()` converts Remote → (Res, State)
3. State used for drift detection and planning

### Apply Cycle (CRUD)
1. Syncer computes diff between desired (graph) and actual (state)
2. For new resources: `Create(Res) → State`
3. For changed resources: `Update(newRes, oldRes, oldState) → State`
4. For deleted resources: `Delete(id, oldRes, oldState)`
5. State updated with results

### Import Cycle
1. `LoadImportableResources()` fetches all resources from API
2. User selects resources to import
3. `Import(Res, remoteId) → State` associates resource with config
4. `FormatForExport()` generates YAML specs

## Type System

### Spec (Configuration)
- Mirrors YAML structure
- May contain multiple resource definitions
- Validated before resource extraction
- Example: `WriterSpec { ID string; Name string }`

### Res (Resource/Input)
- Single resource configuration
- Used for CRUD operations
- May contain PropertyRefs to other resources
- Example: `WriterResource { ID string; Name string }`

### State (Output)
- Computed/server-assigned fields only
- Remote IDs, timestamps, etc.
- Minimal data needed for updates/deletes
- Example: `WriterState { ID string }` (remote ID only)

### Remote (API Response)
- Full response from remote API
- Must implement `RemoteResource` interface
- Wraps API client types
- Example: `RemoteWriter { *backend.RemoteWriter }`

```go
type RemoteResource interface {
    Metadata() RemoteResourceMetadata
}

type RemoteResourceMetadata struct {
    ID          string // Remote system ID
    ExternalID  string // User-facing ID (from config)
    WorkspaceID string
    Name        string
}
```

## Resource References

Resources reference each other using **URNs** (Uniform Resource Names) and **PropertyRefs**.

### URN Format
`{resourceType}:{localId}`

Example: `example_writer:tolkien`

### PropertyRef
```go
type PropertyRef struct {
    URN   string
    Value string // Resolved at runtime from state
}
```

**Usage in Spec:**
```yaml
author: '#/writer/common/tolkien'  # Spec reference format
```

**Conversion to URN:**
```go
// In ExtractResourcesFromSpec
authorURN, err := writer.ParseWriterReference(spec.Author)
// authorURN = "example_writer:tolkien"

resource := &BookResource{
    Author: writer.CreateWriterReference(authorURN),
}
```

**Resolution at runtime:**
```go
// PropertyRef.Value populated with remote ID from writer's state
// Used in Create/Update operations to pass remote ID to API
remoteAuthorID := bookData.Author.Value
```

## Export Strategies

Two patterns for exporting resources to YAML:

### MultiSpecExportStrategy
One resource per file.

```go
type HandlerImpl struct {
    *export.MultiSpecExportStrategy[WriterSpec, RemoteWriter]
    // ...
}

func (h *HandlerImpl) MapRemoteToSpec(externalID string, remote *RemoteWriter) (*export.SpecExportData[WriterSpec], error) {
    return &export.SpecExportData[WriterSpec]{
        Data: &WriterSpec{
            ID:   externalID,
            Name: remote.Name,
        },
        RelativePath: fmt.Sprintf("writers/%s.yaml", externalID),
    }, nil
}
```

### SingleSpecExportStrategy
Multiple resources in one file.

```go
type HandlerImpl struct {
    *export.SingleSpecExportStrategy[BookSpec, RemoteBook]
    // ...
}

func (h *HandlerImpl) MapRemoteToSpec(data map[string]*RemoteBook, resolver resolver.ReferenceResolver) (*export.SpecExportData[BookSpec], error) {
    books := make([]BookItem, 0, len(data))
    for externalID, remote := range data {
        // Resolve references to other resources
        authorRef, err := resolver.ResolveToReference(writerResourceType, remote.AuthorID)

        books = append(books, BookItem{
            ID:     externalID,
            Name:   remote.Name,
            Author: authorRef, // '#/writer/common/tolkien'
        })
    }

    return &export.SpecExportData[BookSpec]{
        Data:         &BookSpec{Books: books},
        RelativePath: "books/books.yaml",
    }, nil
}
```

## Directory Structure

```
providers/
└── <provider-name>/
    ├── provider.go           # Provider composition
    ├── handlers/
    │   ├── <resource-type1>/
    │   │   └── handler.go    # Handler implementation
    │   └── <resource-type2>/
    │       └── handler.go
    ├── model/                # Shared types (optional)
    │   ├── <resource1>.go    # Spec, Res, State, Remote types
    │   └── <resource2>.go
    └── README.md
```

## Example Reference

See `cli/internal/provider/testutils/example/` for complete working implementation with:
- Writer resource (simple, no references)
- Book resource (references Writer)
- In-memory backend (simulates API)
- Full CRUD operations
- Import/export functionality
