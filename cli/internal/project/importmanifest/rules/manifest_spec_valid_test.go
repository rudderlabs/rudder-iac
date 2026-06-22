package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest/manifestspec"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
)

// ctxForSpec builds a manifest ValidationContext from raw workspace blocks.
func ctxForSpec(workspaces []any) *vrules.ValidationContext {
	return &vrules.ValidationContext{
		FilePath: "import-manifest.yaml",
		Kind:     manifestspec.KindImportManifest,
		Version:  specs.SpecVersionV1,
		Metadata: map[string]any{"name": "import-manifest"},
		Spec:     map[string]any{"workspaces": workspaces},
	}
}

func TestManifestSpecSyntaxValidRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewManifestSpecSyntaxValidRule()
	assert.Equal(t, "import-manifest/spec-syntax-valid", rule.ID())
	assert.Equal(t, vrules.Error, rule.Severity())
	assert.Equal(t,
		[]vrules.MatchPattern{vrules.MatchKind(manifestspec.KindImportManifest)},
		rule.AppliesTo(),
	)
}

func TestManifestSpecSyntaxValidRule_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ctx      *vrules.ValidationContext
		expected []vrules.ValidationResult
	}{
		{
			name: "valid manifest",
			ctx: ctxForSpec([]any{
				map[string]any{
					"workspace_id": "ws-1",
					"resources":    []any{map[string]any{"urn": "source:src-1", "remote_id": "r1"}},
				},
			}),
			expected: []vrules.ValidationResult{},
		},
		{
			name:     "no workspaces",
			ctx:      ctxForSpec([]any{}),
			expected: []vrules.ValidationResult{{Reference: "/spec/workspaces", Message: "manifest has no workspaces"}},
		},
		{
			name: "missing workspace_id",
			ctx: ctxForSpec([]any{
				map[string]any{
					"resources": []any{map[string]any{"urn": "source:src-1", "remote_id": "r1"}},
				},
			}),
			expected: []vrules.ValidationResult{{
				Reference: "/spec/workspaces/0/workspace_id",
				Message:   "workspace_id is required",
			}},
		},
		{
			name: "missing remote_id",
			ctx: ctxForSpec([]any{
				map[string]any{
					"workspace_id": "ws-1",
					"resources":    []any{map[string]any{"urn": "source:src-1"}},
				},
			}),
			expected: []vrules.ValidationResult{{
				Reference: "/spec/workspaces/0/resources/0/remote_id",
				Message:   "remote_id is required",
			}},
		},
		{
			name: "local_id is rejected",
			ctx: ctxForSpec([]any{
				map[string]any{
					"workspace_id": "ws-1",
					"resources":    []any{map[string]any{"local_id": "src-1", "remote_id": "r1"}},
				},
			}),
			expected: []vrules.ValidationResult{{
				Reference: "/spec/workspaces/0/resources/0/urn",
				Message:   "urn is required in manifests (local_id not supported)",
			}},
		},
		{
			// A resource missing both required fields reports both errors, not
			// just the first.
			name: "missing urn and remote_id reports both",
			ctx: ctxForSpec([]any{
				map[string]any{
					"workspace_id": "ws-1",
					"resources":    []any{map[string]any{}},
				},
			}),
			expected: []vrules.ValidationResult{
				{
					Reference: "/spec/workspaces/0/resources/0/urn",
					Message:   "urn is required in manifests (local_id not supported)",
				},
				{
					Reference: "/spec/workspaces/0/resources/0/remote_id",
					Message:   "remote_id is required",
				},
			},
		},
		{
			// Duplicate urn detection belongs to the duplicate-urn rule; the
			// per-file rule reports only shape, so a repeated urn here is clean.
			name: "duplicate urn within a workspace is not a shape error",
			ctx: ctxForSpec([]any{
				map[string]any{
					"workspace_id": "ws-1",
					"resources": []any{
						map[string]any{"urn": "source:src-1", "remote_id": "r1"},
						map[string]any{"urn": "source:src-1", "remote_id": "r2"},
					},
				},
			}),
			expected: []vrules.ValidationResult{},
		},
		{
			name: "each workspace missing workspace_id is reported once",
			ctx: ctxForSpec([]any{
				map[string]any{"resources": []any{map[string]any{"urn": "source:src-1", "remote_id": "r1"}}},
				map[string]any{"resources": []any{map[string]any{"urn": "source:src-2", "remote_id": "r2"}}},
			}),
			expected: []vrules.ValidationResult{
				{Reference: "/spec/workspaces/0/workspace_id", Message: "workspace_id is required"},
				{Reference: "/spec/workspaces/1/workspace_id", Message: "workspace_id is required"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.ElementsMatch(t, tt.expected, NewManifestSpecSyntaxValidRule().Validate(tt.ctx))
		})
	}
}
