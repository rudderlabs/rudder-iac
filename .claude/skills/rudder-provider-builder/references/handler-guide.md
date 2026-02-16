# Handler Implementation Guide

Step-by-step guide for implementing a new resource handler using the BaseHandler framework.

## Prerequisites

- API client exists in `api/client/<service>/` with methods for CRUD operations
- Understanding of the resource's data model and API endpoints
- Knowledge of spec YAML format for this resource type

## Step 1: Define Data Types

Create `model/<resource>.go` with four type definitions:

### 1.1 Spec (YAML Configuration)

Mirrors the YAML spec structure. Use JSON tags for unmarshaling.

```go
// <Resource>Spec represents the configuration for a <resource> from YAML
type <Resource>Spec struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    // Add all fields from YAML spec
}
```

For multi-resource specs (multiple resources in one file):
```go
type <Resource>Spec struct {
    <Resources> []<Resource>Item `json:"<resources>"`
}

type <Resource>Item struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    // Fields from YAML
}
```

### 1.2 Res (Resource Data)

Input data for CRUD operations. May include PropertyRefs.

```go
// <Resource>Resource represents the input data for a <resource>
type <Resource>Resource struct {
    ID   string
    Name string
    // Add all configurable fields

    // For references to other resources:
    RelatedResourceURN *resources.PropertyRef
}
```

### 1.3 State (Output State)

Computed/server-assigned fields only. Keep minimal.

```go
// <Resource>State represents the output state from the remote system
// Only contains computed fields (remote ID, timestamps if needed for updates)
type <Resource>State struct {
    ID string // Remote ID (required)
    // Add only fields needed for Update/Delete operations
}
```

### 1.4 Remote (API Response Wrapper)

Wraps the API client type to implement `RemoteResource` interface.

```go
// Remote<Resource> wraps client.<Resource> to implement RemoteResource interface
type Remote<Resource> struct {
    *client.<Resource> // Embed the API client type
}

// Metadata implements the RemoteResource interface
func (r Remote<Resource>) Metadata() handler.RemoteResourceMetadata {
    return handler.RemoteResourceMetadata{
        ID:          r.ID,          // Remote system ID
        ExternalID:  r.ExternalID,  // User-facing ID
        WorkspaceID: r.WorkspaceID, // Workspace context
        Name:        r.Name,        // Display name
    }
}
```

**Important:** Use value receiver `(r Remote<Resource>)` not pointer `(r *Remote<Resource>)` so you can use `Remote<Resource>` (not `*Remote<Resource>`) as the type parameter in BaseHandler.

## Step 2: Create Handler Implementation

Create `handlers/<resource>/handler.go`:

### 2.1 Handler Type Alias and Metadata

```go
package <resource>

import (
    "github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
    "github.com/rudderlabs/rudder-iac/cli/internal/provider/handler/export"
    "github.com/rudderlabs/rudder-iac/cli/internal/providers/<provider>/model"
)

// Type alias for readability
type <Resource>Handler = handler.BaseHandler[
    model.<Resource>Spec,
    model.<Resource>Resource,
    model.<Resource>State,
    model.Remote<Resource>,
]

// Metadata defines resource type and spec kind
var HandlerMetadata = handler.HandlerMetadata{
    ResourceType:     "<provider>_<resource>",  // e.g., "data_graph_schema"
    SpecKind:         "<resource>",             // e.g., "schema" (from YAML kind field)
    SpecMetadataName: "<resources>",           // e.g., "schemas" (for grouping)
}
```

### 2.2 HandlerImpl Struct

```go
// HandlerImpl implements the HandlerImpl interface
type HandlerImpl struct {
    // Choose export strategy:
    // - MultiSpecExportStrategy: one resource per file
    // - SingleSpecExportStrategy: multiple resources in one file
    *export.MultiSpecExportStrategy[model.<Resource>Spec, model.Remote<Resource>]

    apiClient *client.Client // API client dependency
}

// NewHandler creates a new BaseHandler
func NewHandler(apiClient *client.Client) *<Resource>Handler {
    h := &HandlerImpl{apiClient: apiClient}
    h.MultiSpecExportStrategy = &export.MultiSpecExportStrategy[model.<Resource>Spec, model.Remote<Resource>]{
        Handler: h,
    }
    return handler.NewHandler(h)
}

func (h *HandlerImpl) Metadata() handler.HandlerMetadata {
    return HandlerMetadata
}
```

### 2.3 Spec Lifecycle Methods

```go
// NewSpec creates a new empty spec instance
func (h *HandlerImpl) NewSpec() *model.<Resource>Spec {
    return &model.<Resource>Spec{}
}

// ValidateSpec validates the spec configuration
func (h *HandlerImpl) ValidateSpec(spec *model.<Resource>Spec) error {
    // Validate required fields
    if spec.ID == "" {
        return fmt.Errorf("id is required")
    }
    if spec.Name == "" {
        return fmt.Errorf("name is required")
    }

    // Validate field constraints
    // Validate formats, ranges, etc.

    return nil
}

// ExtractResourcesFromSpec extracts resources from validated spec
func (h *HandlerImpl) ExtractResourcesFromSpec(path string, spec *model.<Resource>Spec) (map[string]*model.<Resource>Resource, error) {
    // For single-resource specs:
    resource := &model.<Resource>Resource{
        ID:   spec.ID,
        Name: spec.Name,
        // Map other fields
    }
    return map[string]*model.<Resource>Resource{
        spec.ID: resource,
    }, nil

    // For multi-resource specs:
    resources := make(map[string]*model.<Resource>Resource)
    for _, item := range spec.<Resources> {
        // Parse any references to other resources
        relatedURN, err := other.ParseOtherReference(item.Related)
        if err != nil {
            return nil, fmt.Errorf("parsing reference: %w", err)
        }

        resource := &model.<Resource>Resource{
            ID:          item.ID,
            Name:        item.Name,
            RelatedURN:  other.CreateOtherReference(relatedURN),
        }
        resources[item.ID] = resource
    }
    return resources, nil
}
```

### 2.4 Resource Validation

```go
// ValidateResource validates a single resource in graph context
func (h *HandlerImpl) ValidateResource(resource *model.<Resource>Resource, graph *resources.Graph) error {
    // Basic validation
    if resource.Name == "" {
        return fmt.Errorf("name is required")
    }

    // Cross-resource validation
    if resource.RelatedURN != nil {
        if _, exists := graph.GetResource(resource.RelatedURN.URN); !exists {
            return fmt.Errorf("related resource %s does not exist", resource.RelatedURN.URN)
        }
    }

    return nil
}
```

### 2.5 Remote Resource Loading

```go
// LoadRemoteResources fetches managed resources (with external IDs)
func (h *HandlerImpl) LoadRemoteResources(ctx context.Context) ([]*model.Remote<Resource>, error) {
    // Call API client to fetch all resources
    remoteResources, err := h.apiClient.List<Resources>(ctx)
    if err != nil {
        return nil, fmt.Errorf("listing resources: %w", err)
    }

    // Filter only managed resources (those with external IDs)
    result := make([]*model.Remote<Resource>, 0)
    for _, r := range remoteResources {
        if r.ExternalID != "" {
            result = append(result, &model.Remote<Resource>{<Resource>: r})
        }
    }
    return result, nil
}

// LoadImportableResources fetches all resources for import
func (h *HandlerImpl) LoadImportableResources(ctx context.Context) ([]*model.Remote<Resource>, error) {
    // Call API client to fetch all resources
    remoteResources, err := h.apiClient.List<Resources>(ctx)
    if err != nil {
        return nil, fmt.Errorf("listing resources: %w", err)
    }

    // Return all resources (don't filter by external ID)
    result := make([]*model.Remote<Resource>, 0, len(remoteResources))
    for _, r := range remoteResources {
        result = append(result, &model.Remote<Resource>{<Resource>: r})
    }
    return result, nil
}
```

### 2.6 State Mapping

```go
// MapRemoteToState converts remote resource to Res and State
func (h *HandlerImpl) MapRemoteToState(remote *model.Remote<Resource>, urnResolver handler.URNResolver) (*model.<Resource>Resource, *model.<Resource>State, error) {
    // Skip resources without external IDs
    if remote.ExternalID == "" {
        return nil, nil, nil // Convention: return nil to skip
    }

    // Resolve any references to other resources
    var relatedURN *resources.PropertyRef
    if remote.RelatedID != "" {
        urn, err := urnResolver.GetURNByID(otherResourceType, remote.RelatedID)
        if err != nil {
            return nil, nil, fmt.Errorf("resolving related URN: %w", err)
        }
        relatedURN = other.CreateOtherReference(urn)
    }

    // Map to Resource
    resource := &model.<Resource>Resource{
        ID:         remote.ExternalID,
        Name:       remote.Name,
        RelatedURN: relatedURN,
        // Map other fields
    }

    // Map to State (minimal, computed fields only)
    state := &model.<Resource>State{
        ID: remote.ID, // Remote system ID
    }

    return resource, state, nil
}
```

### 2.7 CRUD Operations

```go
// Create provisions a new resource
func (h *HandlerImpl) Create(ctx context.Context, data *model.<Resource>Resource) (*model.<Resource>State, error) {
    // Resolve PropertyRefs to remote IDs
    var relatedRemoteID string
    if data.RelatedURN != nil {
        relatedRemoteID = data.RelatedURN.Value // Resolved at runtime
    }

    // Call API client
    remote, err := h.apiClient.Create<Resource>(ctx, &client.Create<Resource>Request{
        Name:      data.Name,
        ExternalID: data.ID,
        RelatedID: relatedRemoteID,
        // Map other fields
    })
    if err != nil {
        return nil, fmt.Errorf("creating resource: %w", err)
    }

    // Return state
    return &model.<Resource>State{
        ID: remote.ID,
    }, nil
}

// Update modifies an existing resource
func (h *HandlerImpl) Update(ctx context.Context, newData *model.<Resource>Resource, oldData *model.<Resource>Resource, oldState *model.<Resource>State) (*model.<Resource>State, error) {
    // Resolve PropertyRefs
    var relatedRemoteID string
    if newData.RelatedURN != nil {
        relatedRemoteID = newData.RelatedURN.Value
    }

    // Call API client
    remote, err := h.apiClient.Update<Resource>(ctx, oldState.ID, &client.Update<Resource>Request{
        Name:      newData.Name,
        RelatedID: relatedRemoteID,
        // Map other fields
    })
    if err != nil {
        return nil, fmt.Errorf("updating resource: %w", err)
    }

    // Return updated state
    return &model.<Resource>State{
        ID: remote.ID,
    }, nil
}

// Import associates existing remote resource with config
func (h *HandlerImpl) Import(ctx context.Context, data *model.<Resource>Resource, remoteId string) (*model.<Resource>State, error) {
    // Set external ID on remote resource
    _, err := h.apiClient.Update<Resource>(ctx, remoteId, &client.Update<Resource>Request{
        ExternalID: data.ID,
        // May also update other fields to match config
    })
    if err != nil {
        return nil, fmt.Errorf("setting external ID: %w", err)
    }

    // Return state
    return &model.<Resource>State{
        ID: remoteId,
    }, nil
}

// Delete removes a resource
func (h *HandlerImpl) Delete(ctx context.Context, id string, oldData *model.<Resource>Resource, oldState *model.<Resource>State) error {
    // Call API client
    err := h.apiClient.Delete<Resource>(ctx, oldState.ID)
    if err != nil {
        return fmt.Errorf("deleting resource: %w", err)
    }
    return nil
}
```

### 2.8 Export for Import

**For MultiSpecExportStrategy:**
```go
// MapRemoteToSpec converts single remote resource to spec
func (h *HandlerImpl) MapRemoteToSpec(externalID string, remote *model.Remote<Resource>) (*export.SpecExportData[model.<Resource>Spec], error) {
    return &export.SpecExportData[model.<Resource>Spec]{
        Data: &model.<Resource>Spec{
            ID:   externalID,
            Name: remote.Name,
            // Map other fields
        },
        RelativePath: fmt.Sprintf("<resources>/%s.yaml", externalID),
    }, nil
}
```

**For SingleSpecExportStrategy:**
```go
// MapRemoteToSpec converts multiple remote resources to single spec
func (h *HandlerImpl) MapRemoteToSpec(data map[string]*model.Remote<Resource>, resolver resolver.ReferenceResolver) (*export.SpecExportData[model.<Resource>Spec], error) {
    items := make([]model.<Resource>Item, 0, len(data))

    for externalID, remote := range data {
        // Resolve references to spec format
        relatedRef, err := resolver.ResolveToReference(otherResourceType, remote.RelatedID)
        if err != nil {
            return nil, fmt.Errorf("resolving reference: %w", err)
        }

        items = append(items, model.<Resource>Item{
            ID:      externalID,
            Name:    remote.Name,
            Related: relatedRef, // '#/other/group/id'
        })
    }

    return &export.SpecExportData[model.<Resource>Spec]{
        Data:         &model.<Resource>Spec{<Resources>: items},
        RelativePath: "<resources>/<resources>.yaml",
    }, nil
}
```

### 2.9 Reference Helper Functions

```go
// Parse<Resource>Reference converts spec reference to URN
func Parse<Resource>Reference(ref string) (string, error) {
    specRef, err := specs.ParseSpecReference(ref, map[string]string{
        HandlerMetadata.SpecKind: HandlerMetadata.ResourceType,
    })
    if err != nil {
        return "", err
    }
    return specRef.URN, nil
}

// Create<Resource>Reference creates a PropertyRef with state resolver
var Create<Resource>Reference = func(urn string) *resources.PropertyRef {
    return handler.CreatePropertyRef(urn, func(stateOutput *model.<Resource>State) (string, error) {
        return stateOutput.ID, nil // Extract remote ID from state
    })
}
```

## Step 3: Register Handler in Provider

Update `provider.go`:

```go
func NewProvider(apiClient *client.Client) provider.Provider {
    handlers := []provider.Handler{
        resource1.NewHandler(apiClient),
        resource2.NewHandler(apiClient), // Add your new handler
    }
    return provider.NewBaseProvider(handlers)
}
```

## Step 4: Write Unit Tests

Create `handlers/<resource>/handler_test.go`:

```go
func TestHandler_Create(t *testing.T) {
    // Setup mock API client
    // Create handler
    // Create test data
    // Call Create
    // Assert expected state returned
}

// Test all HandlerImpl methods
```

## Validation Checklist

- [ ] All four types defined (Spec, Res, State, Remote)
- [ ] Remote implements RemoteResource interface with value receiver
- [ ] HandlerMetadata configured correctly
- [ ] NewSpec, ValidateSpec, ExtractResourcesFromSpec implemented
- [ ] ValidateResource validates required fields and references
- [ ] LoadRemoteResources filters by external ID
- [ ] LoadImportableResources returns all resources
- [ ] MapRemoteToState handles missing references gracefully
- [ ] CRUD operations map PropertyRef.Value to remote IDs
- [ ] Export methods resolve references correctly
- [ ] Reference helper functions implemented if resource is referenceable
- [ ] Handler registered in provider
- [ ] Unit tests cover all methods
- [ ] Unit tests use struct comparisons (not field-by-field)
