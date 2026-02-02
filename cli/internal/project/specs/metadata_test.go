package specs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpec_CommonMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		metadata    map[string]any
		expected    Metadata
		expectError bool
		errorText   string
	}{
		{
			name: "valid metadata with all fields",
			metadata: map[string]any{
				"name": "test-project",
				"import": map[string]any{
					"workspaces": []any{
						map[string]any{
							"workspace_id": "ws-123",
							"resources": []any{
								map[string]any{
									"local_id":  "local-1",
									"remote_id": "remote-1",
								},
								map[string]any{
									"local_id":  "local-2",
									"remote_id": "remote-2",
								},
							},
						},
						map[string]any{
							"workspace_id": "ws-456",
							"resources": []any{
								map[string]any{
									"local_id":  "local-3",
									"remote_id": "remote-3",
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
								{LocalID: "local-1", RemoteID: "remote-1"},
								{LocalID: "local-2", RemoteID: "remote-2"},
							},
						},
						{
							WorkspaceID: "ws-456",
							Resources: []ImportIds{
								{LocalID: "local-3", RemoteID: "remote-3"},
							},
						},
					},
				},
			},
		},
		{
			name: "valid metadata with only name",
			metadata: map[string]any{
				"name": "simple-project",
			},
			expected: Metadata{Name: "simple-project"},
		},
		{
			name: "valid metadata with empty import",
			metadata: map[string]any{
				"name": "test-project",
				"import": map[string]any{
					"workspaces": []any{},
				},
			},
			expected: Metadata{
				Name: "test-project",
				Import: &WorkspacesImportMetadata{
					Workspaces: []WorkspaceImportMetadata{},
				},
			},
		},
		{
			name:     "empty metadata map",
			metadata: map[string]any{},
			expected: Metadata{},
		},
		{
			name:     "nil metadata map",
			metadata: nil,
			expected: Metadata{},
		},
		{
			name: "workspace with empty resources",
			metadata: map[string]any{
				"name": "test-project",
				"import": map[string]any{
					"workspaces": []any{
						map[string]any{
							"workspace_id": "ws-123",
							"resources":    []any{},
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
							Resources:   []ImportIds{},
						},
					},
				},
			},
		},
		{
			name: "metadata with extra unknown fields",
			metadata: map[string]any{
				"name":          "test-project",
				"unknown_field": "should-be-ignored",
				"import": map[string]any{
					"workspaces": []any{
						map[string]any{
							"workspace_id": "ws-123",
							"extra_field":  "ignored",
							"resources": []any{
								map[string]any{
									"local_id":      "local-1",
									"remote_id":     "remote-1",
									"another_extra": "ignored",
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
								{LocalID: "local-1", RemoteID: "remote-1"},
							},
						},
					},
				},
			},
		},
		{
			name: "invalid type for name field",
			metadata: map[string]any{
				"name": 12345,
			},
			expectError: true,
			errorText:   "failed to decode metadata",
		},
		{
			name: "invalid type for import field",
			metadata: map[string]any{
				"name":   "test-project",
				"import": "invalid-string",
			},
			expected: Metadata{
				Name: "test-project",
			},
			expectError: true,
			errorText:   "failed to decode metadata",
		},
		{
			name: "invalid type for workspaces field",
			metadata: map[string]any{
				"name": "test-project",
				"import": map[string]any{
					"workspaces": "invalid-string",
				},
			},
			expected: Metadata{
				Name: "test-project",
			},
			expectError: true,
			errorText:   "failed to decode metadata",
		},
		{
			name: "invalid workspace structure",
			metadata: map[string]any{
				"name": "test-project",
				"import": map[string]any{
					"workspaces": []any{
						"invalid-workspace",
					},
				},
			},
			expected: Metadata{
				Name:   "test-project",
				Import: nil,
			},
			expectError: true,
			errorText:   "failed to decode metadata",
		},
		{
			name: "invalid resources structure",
			metadata: map[string]any{
				"name": "test-project",
				"import": map[string]any{
					"workspaces": []any{
						map[string]any{
							"workspace_id": "ws-123",
							"resources":    "invalid-resources",
						},
					},
				},
			},
			expected: Metadata{
				Name:   "test-project",
				Import: nil,
			},
			expectError: true,
			errorText:   "failed to decode metadata",
		},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			spec := &Spec{
				Metadata: tt.metadata,
			}

			metadata, err := spec.CommonMetadata()

			// metadata are expected even if there are validation errors
			assert.Equal(t, tt.expected, metadata, "unexpected metadata result")
			assert.Equal(t, tt.expectError, err != nil, "unexpected error state")

			if tt.expectError {
				assert.Contains(t, err.Error(), tt.errorText)
			}
		})
	}
}

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

func TestMetadata_ToMap(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		metadata Metadata
		expected map[string]any
	}{
		"empty Metadata": {
			metadata: Metadata{},
			expected: map[string]any{},
		},
		"Metadata with only Name": {
			metadata: Metadata{
				Name: "example-name",
			},
			expected: map[string]any{
				"name": "example-name",
			},
		},
		"Metadata with Name, Import and Workspaces": {
			metadata: Metadata{
				Name: "test-resource",
				Import: &WorkspacesImportMetadata{
					Workspaces: []WorkspaceImportMetadata{
						{
							WorkspaceID: "ws-123",
							Resources: []ImportIds{
								{LocalID: "local-1", RemoteID: "remote-1"},
								{LocalID: "local-2", RemoteID: "remote-2"},
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
								map[string]any{"local_id": "local-1", "remote_id": "remote-1"},
								map[string]any{"local_id": "local-2", "remote_id": "remote-2"},
							},
						},
					},
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := tt.metadata.ToMap()
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
