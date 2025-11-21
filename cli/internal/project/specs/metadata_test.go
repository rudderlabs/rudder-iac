package specs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpec_CommonMetadata(t *testing.T) {
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
				Import: WorkspacesImportMetadata{
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
				Import: WorkspacesImportMetadata{
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
				Import: WorkspacesImportMetadata{
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
			name: "workspace missing workspace_id",
			metadata: map[string]any{
				"name": "test-project",
				"import": map[string]any{
					"workspaces": []any{
						map[string]any{
							"resources": []any{
								map[string]any{
									"local_id":  "local-1",
									"remote_id": "remote-1",
								},
							},
						},
					},
				},
			},
			expected: Metadata{
				Name: "test-project",
				Import: WorkspacesImportMetadata{
					Workspaces: []WorkspaceImportMetadata{
						{
							Resources: []ImportIds{
								{LocalID: "local-1", RemoteID: "remote-1"},
							},
						},
					},
				},
			},
			expectError: true,
			errorText:   "missing required field 'workspace_id'",
		},
		{
			name: "resource with missing remote_id",
			metadata: map[string]any{
				"name": "test-project",
				"import": map[string]any{
					"workspaces": []any{
						map[string]any{
							"workspace_id": "ws-123",
							"resources": []any{
								map[string]any{
									"local_id": "local-1",
								},
							},
						},
					},
				},
			},
			expected: Metadata{
				Name: "test-project",
				Import: WorkspacesImportMetadata{
					Workspaces: []WorkspaceImportMetadata{
						{
							WorkspaceID: "ws-123",
							Resources: []ImportIds{
								{LocalID: "local-1"},
							},
						},
					},
				},
			},
			expectError: true,
			errorText:   "missing required field 'remote_id'",
		},
		{
			name: "resource with missing local_id",
			metadata: map[string]any{
				"name": "test-project",
				"import": map[string]any{
					"workspaces": []any{
						map[string]any{
							"workspace_id": "ws-123",
							"resources": []any{
								map[string]any{
									"remote_id": "remote-1",
								},
							},
						},
					},
				},
			},
			expected: Metadata{
				Name: "test-project",
				Import: WorkspacesImportMetadata{
					Workspaces: []WorkspaceImportMetadata{
						{
							WorkspaceID: "ws-123",
							Resources: []ImportIds{
								{RemoteID: "remote-1"},
							},
						},
					},
				},
			},
			expectError: true,
			errorText:   "missing required field 'local_id'",
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
				Import: WorkspacesImportMetadata{
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
				Name: "test-project",
				Import: WorkspacesImportMetadata{
					Workspaces: []WorkspaceImportMetadata{
						{},
					},
				},
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
				Name: "test-project",
				Import: WorkspacesImportMetadata{
					Workspaces: []WorkspaceImportMetadata{
						{
							WorkspaceID: "ws-123",
						},
					},
				},
			},
			expectError: true,
			errorText:   "failed to decode metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestMetadata_ToMap(t *testing.T) {
	metadata := Metadata{
		Name: "test-resource",
		Import: WorkspacesImportMetadata{
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
	}

	result, err := metadata.ToMap()
	assert.NoError(t, err)

	expected := map[string]any{
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
	}

	assert.Equal(t, expected, result)
}
