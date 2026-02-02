package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
)

func TestMetadataSyntaxValidRule_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		ctx            *rules.ValidationContext
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "empty metadata returns nil",
			ctx: &rules.ValidationContext{
				Metadata: map[string]any{},
			},
			expectedErrors: 0,
			expectedRefs:   []string{},
			expectedMsgs:   []string{},
		},
		{
			name: "valid metadata with name only",
			ctx: &rules.ValidationContext{
				Metadata: map[string]any{"name": "test-project"},
			},
			expectedErrors: 0,
			expectedRefs:   []string{},
			expectedMsgs:   []string{},
		},
		{
			name: "missing required name field",
			ctx: &rules.ValidationContext{
				Metadata: map[string]any{"something": "else"},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/metadata/name"},
			expectedMsgs:   []string{"'name' is required"},
		},
		{
			name: "valid metadata with complete import structure",
			ctx: &rules.ValidationContext{
				Metadata: map[string]any{
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
								},
							},
						},
					},
				},
			},
			expectedErrors: 0,
			expectedRefs:   []string{},
			expectedMsgs:   []string{},
		},
		{
			name: "import with missing workspace_id",
			ctx: &rules.ValidationContext{
				Metadata: map[string]any{
					"name": "test-project",
					"import": map[string]any{
						"workspaces": []any{
							map[string]any{
								"resources": []any{},
							},
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/metadata/import/workspaces/0/workspace_id"},
			expectedMsgs:   []string{"'workspace_id' is required"},
		},
		{
			name: "import with missing local_id in resource",
			ctx: &rules.ValidationContext{
				Metadata: map[string]any{
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
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/metadata/import/workspaces/0/resources/0/local_id"},
			expectedMsgs:   []string{"'local_id' is required"},
		},
		{
			name: "import with missing remote_id in resource",
			ctx: &rules.ValidationContext{
				Metadata: map[string]any{
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
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/metadata/import/workspaces/0/resources/0/remote_id"},
			expectedMsgs:   []string{"'remote_id' is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewMetadataSyntaxValidRule()
			results := rule.Validate(tt.ctx)

			assert.Len(t, results, tt.expectedErrors, "unexpected number of validation errors")

			if tt.expectedErrors > 0 {
				actualRefs := make([]string, len(results))
				actualMsgs := make([]string, len(results))
				for i, r := range results {
					actualRefs[i] = r.Reference
					actualMsgs[i] = r.Message
				}

				assert.ElementsMatch(t, tt.expectedRefs, actualRefs, "references don't match")
				assert.ElementsMatch(t, tt.expectedMsgs, actualMsgs, "messages don't match")
			}
		})
	}
}

func TestMetadataSyntaxValidRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewMetadataSyntaxValidRule()

	assert.Equal(t, "project/metadata-syntax-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "metadata syntax must be valid", rule.Description())
	assert.Equal(t, []string{"*"}, rule.AppliesTo())

	examples := rule.Examples()
	assert.NotEmpty(t, examples.Valid)
	assert.NotEmpty(t, examples.Invalid)
}
