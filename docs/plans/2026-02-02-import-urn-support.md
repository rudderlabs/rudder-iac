# Import URN Support Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Enable unambiguous resource identification in import metadata by supporting URN-based references alongside the legacy local_id format.

**Architecture:** Add a `URN` field to `ImportIds` struct with mutual exclusivity validation. BaseHandler requires URN for new providers while legacy providers continue using local_id unchanged. The URN format is `resource-type:resource-id`.

**Tech Stack:** Go, testify for assertions

---

## Task 1: Add URN Field to ImportIds Struct

**Files:**
- Modify: `cli/internal/project/specs/metadata.go:28-31`
- Test: `cli/internal/project/specs/metadata_test.go`

**Step 1: Write failing test for URN field parsing**

Add to `cli/internal/project/specs/metadata_test.go` in the `TestSpec_CommonMetadata` test cases:

```go
{
    name: "valid metadata with URN-based import",
    metadata: map[string]any{
        "name": "test-project",
        "import": map[string]any{
            "workspaces": []any{
                map[string]any{
                    "workspace_id": "ws-123",
                    "resources": []any{
                        map[string]any{
                            "urn":       "data-graph:my-graph",
                            "remote_id": "remote-1",
                        },
                        map[string]any{
                            "urn":       "model:user-model",
                            "remote_id": "remote-2",
                        },
                    },
                },
            },
        },
    },
    expected: Metadata{
        Name: "test-project",
        Import: &WorkspacesImportMetadata{
            Workspaces: []WorkspaceImportMetadata{
                {
                    WorkspaceID: "ws-123",
                    Resources: []ImportIds{
                        {URN: "data-graph:my-graph", RemoteID: "remote-1"},
                        {URN: "model:user-model", RemoteID: "remote-2"},
                    },
                },
            },
        },
    },
},
```

**Step 2: Run test to verify it fails**

Run: `go test ./cli/internal/project/specs/... -run TestSpec_CommonMetadata -v`
Expected: FAIL - URN field not recognized

**Step 3: Add URN field to ImportIds struct**

Modify `cli/internal/project/specs/metadata.go`:

```go
// ImportIds holds the local and remote IDs for a resource to be imported, as specified in import spec metadata
type ImportIds struct {
	// Deprecated: Use URN instead for new providers.
	LocalID  string `yaml:"local_id,omitempty" json:"local_id,omitempty"`
	// URN identifies the local resource (format: "resource-type:resource-id")
	URN      string `yaml:"urn,omitempty" json:"urn,omitempty"`
	RemoteID string `yaml:"remote_id" json:"remote_id" validate:"required"`
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./cli/internal/project/specs/... -run TestSpec_CommonMetadata -v`
Expected: PASS

**Step 5: Commit**

```bash
git add cli/internal/project/specs/metadata.go cli/internal/project/specs/metadata_test.go
git commit -m "feat(specs): add URN field to ImportIds struct"
```

---

## Task 2: Add ImportIds Validation Method

**Files:**
- Modify: `cli/internal/project/specs/metadata.go`
- Test: `cli/internal/project/specs/metadata_test.go`

**Step 1: Write failing tests for ImportIds.Validate()**

Add new test function to `cli/internal/project/specs/metadata_test.go`:

```go
func TestImportIds_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		importIds   ImportIds
		expectError bool
		errorText   string
	}{
		{
			name: "valid with URN only",
			importIds: ImportIds{
				URN:      "data-graph:my-graph",
				RemoteID: "remote-123",
			},
			expectError: false,
		},
		{
			name: "valid with LocalID only",
			importIds: ImportIds{
				LocalID:  "my-resource",
				RemoteID: "remote-123",
			},
			expectError: false,
		},
		{
			name: "invalid - both URN and LocalID set",
			importIds: ImportIds{
				URN:      "data-graph:my-graph",
				LocalID:  "my-resource",
				RemoteID: "remote-123",
			},
			expectError: true,
			errorText:   "urn and local_id are mutually exclusive",
		},
		{
			name: "invalid - neither URN nor LocalID set",
			importIds: ImportIds{
				RemoteID: "remote-123",
			},
			expectError: true,
			errorText:   "either urn or local_id must be set",
		},
		{
			name: "invalid - missing RemoteID with URN",
			importIds: ImportIds{
				URN: "data-graph:my-graph",
			},
			expectError: true,
			errorText:   "remote_id is required",
		},
		{
			name: "invalid - missing RemoteID with LocalID",
			importIds: ImportIds{
				LocalID: "my-resource",
			},
			expectError: true,
			errorText:   "remote_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.importIds.Validate()
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorText)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./cli/internal/project/specs/... -run TestImportIds_Validate -v`
Expected: FAIL - Validate method does not exist

**Step 3: Implement ImportIds.Validate() method**

Add to `cli/internal/project/specs/metadata.go`:

```go
// Validate checks that ImportIds has valid field combinations
func (i *ImportIds) Validate() error {
	hasLocalID := i.LocalID != ""
	hasURN := i.URN != ""

	if hasLocalID && hasURN {
		return fmt.Errorf("urn and local_id are mutually exclusive")
	}
	if !hasLocalID && !hasURN {
		return fmt.Errorf("either urn or local_id must be set")
	}
	if i.RemoteID == "" {
		return fmt.Errorf("remote_id is required")
	}
	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./cli/internal/project/specs/... -run TestImportIds_Validate -v`
Expected: PASS

**Step 5: Commit**

```bash
git add cli/internal/project/specs/metadata.go cli/internal/project/specs/metadata_test.go
git commit -m "feat(specs): add ImportIds.Validate() for mutual exclusivity"
```

---

## Task 3: Update Metadata.Validate() to Use ImportIds.Validate()

**Files:**
- Modify: `cli/internal/project/specs/metadata.go:34-52`
- Test: `cli/internal/project/specs/metadata_test.go`

**Step 1: Write failing test for metadata validation with mutual exclusivity error**

Add test case to `TestSpec_CommonMetadata` in `cli/internal/project/specs/metadata_test.go`:

```go
{
    name: "invalid - both urn and local_id in import resource",
    metadata: map[string]any{
        "name": "test-project",
        "import": map[string]any{
            "workspaces": []any{
                map[string]any{
                    "workspace_id": "ws-123",
                    "resources": []any{
                        map[string]any{
                            "urn":       "data-graph:my-graph",
                            "local_id":  "my-resource",
                            "remote_id": "remote-1",
                        },
                    },
                },
            },
        },
    },
    expected: Metadata{
        Name: "test-project",
        Import: &WorkspacesImportMetadata{
            Workspaces: []WorkspaceImportMetadata{
                {
                    WorkspaceID: "ws-123",
                    Resources: []ImportIds{
                        {URN: "data-graph:my-graph", LocalID: "my-resource", RemoteID: "remote-1"},
                    },
                },
            },
        },
    },
    expectError: false, // CommonMetadata doesn't validate, only Metadata.Validate() does
},
```

Add new test function for `Metadata.Validate()`:

```go
func TestMetadata_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		metadata    Metadata
		expectError bool
		errorText   string
	}{
		{
			name: "valid with URN-based imports",
			metadata: Metadata{
				Name: "test",
				Import: &WorkspacesImportMetadata{
					Workspaces: []WorkspaceImportMetadata{
						{
							WorkspaceID: "ws-123",
							Resources: []ImportIds{
								{URN: "data-graph:my-graph", RemoteID: "remote-1"},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid with LocalID-based imports",
			metadata: Metadata{
				Name: "test",
				Import: &WorkspacesImportMetadata{
					Workspaces: []WorkspaceImportMetadata{
						{
							WorkspaceID: "ws-123",
							Resources: []ImportIds{
								{LocalID: "my-resource", RemoteID: "remote-1"},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid - both URN and LocalID",
			metadata: Metadata{
				Name: "test",
				Import: &WorkspacesImportMetadata{
					Workspaces: []WorkspaceImportMetadata{
						{
							WorkspaceID: "ws-123",
							Resources: []ImportIds{
								{URN: "data-graph:my-graph", LocalID: "conflict", RemoteID: "remote-1"},
							},
						},
					},
				},
			},
			expectError: true,
			errorText:   "urn and local_id are mutually exclusive",
		},
		{
			name: "invalid - missing workspace_id",
			metadata: Metadata{
				Name: "test",
				Import: &WorkspacesImportMetadata{
					Workspaces: []WorkspaceImportMetadata{
						{
							WorkspaceID: "",
							Resources: []ImportIds{
								{URN: "data-graph:my-graph", RemoteID: "remote-1"},
							},
						},
					},
				},
			},
			expectError: true,
			errorText:   "missing required field 'workspace_id'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.metadata.Validate()
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorText)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./cli/internal/project/specs/... -run TestMetadata_Validate -v`
Expected: FAIL - mutual exclusivity error not detected

**Step 3: Update Metadata.Validate() to call ImportIds.Validate()**

Modify `cli/internal/project/specs/metadata.go`:

```go
// Validate checks that all required fields are present in the Metadata
func (m *Metadata) Validate() error {
	if m.Import != nil {
		for idx, ws := range m.Import.Workspaces {
			if ws.WorkspaceID == "" {
				return fmt.Errorf("missing required field 'workspace_id' in import metadata, workspace index %d", idx)
			}
			for _, res := range ws.Resources {
				if err := res.Validate(); err != nil {
					return fmt.Errorf("invalid import resource in workspace '%s': %w", ws.WorkspaceID, err)
				}
			}
		}
	}

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./cli/internal/project/specs/... -run TestMetadata_Validate -v`
Expected: PASS

**Step 5: Run all metadata tests**

Run: `go test ./cli/internal/project/specs/... -v`
Expected: All PASS

**Step 6: Commit**

```bash
git add cli/internal/project/specs/metadata.go cli/internal/project/specs/metadata_test.go
git commit -m "feat(specs): integrate ImportIds.Validate() into Metadata.Validate()"
```

---

## Task 4: Update Metadata.ToMap() for URN Field

**Files:**
- Modify: `cli/internal/project/specs/metadata_test.go`

**Step 1: Write test for ToMap with URN**

Add test case to `TestMetadata_ToMap` in `cli/internal/project/specs/metadata_test.go`:

```go
"Metadata with URN-based imports": {
    metadata: Metadata{
        Name: "test-resource",
        Import: &WorkspacesImportMetadata{
            Workspaces: []WorkspaceImportMetadata{
                {
                    WorkspaceID: "ws-123",
                    Resources: []ImportIds{
                        {URN: "data-graph:my-graph", RemoteID: "remote-1"},
                        {URN: "model:user-model", RemoteID: "remote-2"},
                    },
                },
            },
        },
    },
    expected: map[string]any{
        "name": "test-resource",
        "import": map[string]any{
            "workspaces": []any{
                map[string]any{
                    "workspace_id": "ws-123",
                    "resources": []any{
                        map[string]any{"urn": "data-graph:my-graph", "remote_id": "remote-1"},
                        map[string]any{"urn": "model:user-model", "remote_id": "remote-2"},
                    },
                },
            },
        },
    },
},
```

**Step 2: Run test to verify it passes**

Run: `go test ./cli/internal/project/specs/... -run TestMetadata_ToMap -v`
Expected: PASS (JSON marshaling handles omitempty correctly)

**Step 3: Commit**

```bash
git add cli/internal/project/specs/metadata_test.go
git commit -m "test(specs): add ToMap test for URN-based imports"
```

---

## Task 5: Update BaseHandler to Require URN

**Files:**
- Modify: `cli/internal/provider/handler/basehandler.go:105-117`
- Test: Create new test file or add to existing handler tests

**Step 1: Write failing test for URN requirement**

Create or add to test file. Since BaseHandler is generic, we'll test through the example provider in `cli/internal/provider/testutils/example/`.

First, check if there's an existing test we can extend, or create a focused unit test:

```go
// Add to appropriate test file, e.g., cli/internal/provider/handler/basehandler_test.go

func TestBaseHandler_LoadImportMetadata_RequiresURN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		metadata    *specs.WorkspacesImportMetadata
		expectError bool
		errorText   string
	}{
		{
			name: "valid URN-based metadata",
			metadata: &specs.WorkspacesImportMetadata{
				Workspaces: []specs.WorkspaceImportMetadata{
					{
						WorkspaceID: "ws-123",
						Resources: []specs.ImportIds{
							{URN: "test-type:resource-1", RemoteID: "remote-1"},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid - local_id not supported",
			metadata: &specs.WorkspacesImportMetadata{
				Workspaces: []specs.WorkspaceImportMetadata{
					{
						WorkspaceID: "ws-123",
						Resources: []specs.ImportIds{
							{LocalID: "resource-1", RemoteID: "remote-1"},
						},
					},
				},
			},
			expectError: true,
			errorText:   "urn field is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Test implementation depends on how BaseHandler is tested
			// This may need adjustment based on existing test patterns
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./cli/internal/provider/handler/... -run TestBaseHandler_LoadImportMetadata -v`
Expected: FAIL - local_id still accepted

**Step 3: Update loadImportMetadata to require URN**

Modify `cli/internal/provider/handler/basehandler.go`:

```go
func (h *BaseHandler[Spec, Res, State, Remote]) loadImportMetadata(m *specs.WorkspacesImportMetadata) error {
	workspaces := m.Workspaces
	for _, workspaceMetadata := range workspaces {
		workspaceId := workspaceMetadata.WorkspaceID
		resources := workspaceMetadata.Resources
		for _, resourceMetadata := range resources {
			if err := resourceMetadata.Validate(); err != nil {
				return fmt.Errorf("invalid import metadata for workspace '%s': %w", workspaceId, err)
			}
			if resourceMetadata.URN == "" {
				return fmt.Errorf("urn field is required for import metadata in workspace '%s' (local_id is not supported)", workspaceId)
			}
			h.importMetadata[resourceMetadata.URN] = &importResourceInfo{
				WorkspaceId: workspaceId,
				RemoteId:    resourceMetadata.RemoteID,
			}
		}
	}
	return nil
}
```

**Step 4: Update LoadSpec to handle loadImportMetadata error**

The current `LoadSpec` ignores the return value. Update it:

```go
func (h *BaseHandler[Spec, Res, State, Remote]) LoadSpec(path string, s *specs.Spec) error {
	// ... existing code ...

	if commonMetadata.Import != nil {
		if err := h.loadImportMetadata(commonMetadata.Import); err != nil {
			return fmt.Errorf("loading import metadata: %w", err)
		}
	}

	return nil
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./cli/internal/provider/handler/... -v`
Expected: PASS

**Step 6: Commit**

```bash
git add cli/internal/provider/handler/basehandler.go
git commit -m "feat(handler): require URN field in import metadata for BaseHandler"
```

---

## Task 6: Update BaseHandler.Resources() to Use URN Key

**Files:**
- Modify: `cli/internal/provider/handler/basehandler.go:193-212`

**Step 1: Understand the current behavior**

Currently `Resources()` looks up import metadata by `resourceId`:
```go
if importMetadata, ok := h.importMetadata[resourceId]; ok {
```

After Task 5, `importMetadata` is keyed by URN. We need to construct the URN to look up.

**Step 2: Write test for Resources() with URN-based import metadata**

This can be tested through the example provider or a dedicated test.

**Step 3: Update Resources() to construct URN for lookup**

Modify `cli/internal/provider/handler/basehandler.go`:

```go
func (h *BaseHandler[Spec, Res, State, Remote]) Resources() ([]*resources.Resource, error) {
	result := make([]*resources.Resource, 0, len(h.resources))
	for resourceId, s := range h.resources {
		opts := []resources.ResourceOpts{
			resources.WithRawData(s),
		}
		// Construct URN to look up import metadata (now keyed by URN, not resourceId)
		urn := resources.URN(resourceId, h.metadata.ResourceType)
		if importMetadata, ok := h.importMetadata[urn]; ok {
			opts = append(opts, resources.WithResourceImportMetadata(importMetadata.RemoteId, importMetadata.WorkspaceId))
		}
		r := resources.NewResource(
			resourceId,
			h.metadata.ResourceType,
			resources.ResourceData{}, // deprecated, will be removed
			[]string{},
			opts...,
		)
		result = append(result, r)
	}
	return result, nil
}
```

**Step 4: Run handler tests**

Run: `go test ./cli/internal/provider/handler/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add cli/internal/provider/handler/basehandler.go
git commit -m "feat(handler): use URN key for import metadata lookup in Resources()"
```

---

## Task 7: Verify Transformations Provider Compatibility

**Files:**
- Test: `cli/internal/providers/transformations/provider_test.go`

The transformations provider uses BaseProvider but may not have import functionality yet. We need to verify it still works and doesn't break.

**Step 1: Run existing transformations tests**

Run: `go test ./cli/internal/providers/transformations/... -v`
Expected: PASS (no import-related tests should fail)

**Step 2: If any tests fail, investigate and fix**

The transformations provider doesn't appear to have import tests, so this should pass.

**Step 3: Commit if any fixes needed**

No commit needed if tests pass.

---

## Task 8: Verify Data Graph Provider Works with URN

**Files:**
- Test: `cli/internal/providers/datagraph/provider_test.go`

**Step 1: Run existing datagraph tests**

Run: `go test ./cli/internal/providers/datagraph/... -v`
Expected: PASS

**Step 2: Add test for import metadata with URN**

If not already covered, add a test case for loading a data-graph spec with URN-based import metadata.

**Step 3: Commit if any changes needed**

```bash
git add cli/internal/providers/datagraph/
git commit -m "test(datagraph): verify URN-based import metadata works"
```

---

## Task 9: Run Full Test Suite

**Step 1: Run all unit tests**

Run: `make test`
Expected: All PASS

**Step 2: Run E2E tests if available**

Run: `make test-e2e`
Expected: All PASS (legacy providers should work unchanged)

**Step 3: Fix any failures**

If legacy provider tests fail, ensure they're not affected by the BaseHandler changes (they use their own implementations).

**Step 4: Final commit if needed**

```bash
git commit -m "chore: ensure all tests pass after URN import support"
```

---

## Task 10: Update Design Document Status

**Files:**
- Modify: `docs/plans/2026-02-02-import-urn-support-design.md`

**Step 1: Update status to Complete**

Change `**Status:** Draft` to `**Status:** Complete`

**Step 2: Commit**

```bash
git add docs/plans/2026-02-02-import-urn-support-design.md
git commit -m "docs: mark import URN support design as complete"
```
