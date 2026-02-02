# Import URN Support Design

**Date:** 2026-02-02
**Linear Issue:** PRO-5164
**Status:** Complete

## Problem Statement

The current import metadata system assumes one resource type per YAML spec file. This assumption breaks down for providers like data-graph that bundle multiple resource types (DataGraph, Model, Relationship) in a single spec.

The existing `ImportIds` struct identifies resources by `local_id` only:

```go
type ImportIds struct {
    LocalID  string `yaml:"local_id"`
    RemoteID string `yaml:"remote_id"`
}
```

When a spec contains multiple resource types, `local_id` alone is ambiguous - we cannot determine which resource type the import refers to.

## Background

### Key Concepts

- **Resource URN**: Globally unique identifier combining resource type and resource ID (e.g., `model:user-model`)
- **Spec Kind**: Describes the YAML format (e.g., `data-graph`, `properties`). One spec kind can produce multiple resource types.
- **Resource Type**: The type component of a URN (e.g., `model`, `relationship`, `data-graph`)

### Current Architecture

- **Legacy providers** (datacatalog, retl, event-stream): Custom implementations, handle import metadata in their own way
- **BaseProvider-based providers** (transformations, datagraph): Use the new framework

### Import Flow

1. `rudder-cli import` generates YAML specs with import metadata (maps local URNs to remote IDs)
2. `rudder-cli apply` reads metadata, distinguishes "import existing" vs "create new", executes sync

## Design

### 1. Schema Changes

Add `URN` field to `ImportIds` with mutual exclusivity:

```go
// In cli/internal/project/specs/metadata.go

type ImportIds struct {
    // Deprecated: Use URN instead for new providers.
    LocalID  string `yaml:"local_id,omitempty"`

    // URN identifies the local resource (format: "resource-type:resource-id")
    URN      string `yaml:"urn,omitempty"`

    RemoteID string `yaml:"remote_id"`
}

func (i *ImportIds) Validate() error {
    hasLocalID := i.LocalID != ""
    hasURN := i.URN != ""

    if hasLocalID && hasURN {
        return errors.New("urn and local_id are mutually exclusive")
    }
    if !hasLocalID && !hasURN {
        return errors.New("either urn or local_id must be set")
    }
    if i.RemoteID == "" {
        return errors.New("remote_id is required")
    }
    return nil
}
```

**YAML Examples:**

```yaml
# New format (required for multi-type specs like data-graph)
metadata:
  import:
    workspaces:
      - workspace_id: "ws-123"
        resources:
          - urn: "data-graph:my-graph"
            remote_id: "remote-dg-123"
          - urn: "model:user-model"
            remote_id: "remote-model-456"
          - urn: "relationship:user-orders"
            remote_id: "remote-rel-789"

# Legacy format (still works for single-type spec kinds)
metadata:
  import:
    workspaces:
      - workspace_id: "ws-123"
        resources:
          - local_id: "my-source"
            remote_id: "remote-xyz-456"
```

### 2. BaseProvider Changes

BaseProvider expects `URN` field to be populated. No fallback logic for `local_id`.

```go
// In cli/internal/provider/baseprovider.go or handler

func (h *BaseHandler) loadImportMetadata(m *specs.WorkspacesImportMetadata) error {
    for _, ws := range m.Workspaces {
        for _, res := range ws.Resources {
            if err := res.Validate(); err != nil {
                return fmt.Errorf("invalid import metadata: %w", err)
            }
            if res.URN == "" {
                return fmt.Errorf("urn field required for import metadata (local_id not supported)")
            }
            h.importMetadata[res.URN] = &importResourceInfo{
                RemoteID:    res.RemoteID,
                WorkspaceID: ws.WorkspaceID,
            }
        }
    }
    return nil
}
```

### 3. Provider-Specific Behavior

| Provider | Format | Notes |
|----------|--------|-------|
| datacatalog | `local_id` | Legacy, unchanged |
| retl | `local_id` | Legacy, unchanged |
| event-stream | `local_id` | Legacy, unchanged |
| transformations | `urn` | Update to use URN (coordinated, not released) |
| datagraph | `urn` | New provider, uses URN via BaseProvider |

### 4. Import Command Changes

The data-graph provider's import flow generates URN-based metadata:

```go
// When generating import metadata for a resource
importIds := specs.ImportIds{
    URN:      resources.URN(resource.Type, resource.LocalID),
    RemoteID: resource.RemoteID,
}
```

Legacy providers continue generating `local_id` format unchanged.

## Scope

### In Scope

1. Add `URN` field to `ImportIds` struct
2. Add mutual exclusivity validation
3. Update BaseProvider to require URN field
4. Update transformations provider to use URN (coordinated change)
5. Data-graph provider uses URN-based import metadata

### Out of Scope

1. Changes to legacy providers (datacatalog, retl, event-stream)
2. Migration tooling for existing spec files
3. Deprecation warnings (legacy providers work as-is)

## Migration Path

1. **Phase 1 (this change):** Add URN support, BaseProvider requires URN, legacy providers unchanged
2. **Phase 2 (future):** Update legacy providers to use URN, add deprecation warnings for `local_id`
3. **Phase 3 (future):** Remove `local_id` support entirely

## Testing Strategy

- Unit tests for `ImportIds.Validate()` mutual exclusivity
- Unit tests for BaseProvider import metadata loading with URN
- Integration tests for data-graph import flow with URN-based metadata
- Verify legacy providers continue working with `local_id` (no regressions)
