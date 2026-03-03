package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
)

// parseSpecWithLocalIDs returns a ParseSpecFunc that always returns the given IDs as LocalIDs.
func parseSpecWithLocalIDs(ids ...string) ParseSpecFunc {
	return func(_ string, _ *specs.Spec) (*specs.ParsedSpec, error) {
		localIDs := make([]specs.LocalID, len(ids))
		for i, id := range ids {
			localIDs[i] = specs.LocalID{ID: id, JSONPointerPath: "/spec/id"}
		}
		return &specs.ParsedSpec{LocalIDs: localIDs}, nil
	}
}

// noopParseSpec returns an empty ParsedSpec — used when import ID validation
// is not the focus of a test case.
func noopParseSpec(path string, spec *specs.Spec) (*specs.ParsedSpec, error) {
	return parseSpecWithLocalIDs()(path, spec)
}

func TestMetadataSyntaxValidRule_Validate(t *testing.T) {
	t.Parallel()

	appliesToVersions := []string{
		specs.SpecVersionV0_1,
		specs.SpecVersionV0_1Variant,
	}

	tests := []struct {
		name           string
		parseSpec      ParseSpecFunc
		ctx            *rules.ValidationContext
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name:      "empty metadata returns nil",
			parseSpec: noopParseSpec,
			ctx: &rules.ValidationContext{
				Metadata: map[string]any{},
			},
			expectedErrors: 0,
			expectedRefs:   []string{},
			expectedMsgs:   []string{},
		},
		{
			name:      "valid metadata with name only",
			parseSpec: noopParseSpec,
			ctx: &rules.ValidationContext{
				Metadata: map[string]any{"name": "test-project"},
			},
			expectedErrors: 0,
			expectedRefs:   []string{},
			expectedMsgs:   []string{},
		},
		{
			name:      "missing required name field",
			parseSpec: noopParseSpec,
			ctx: &rules.ValidationContext{
				Metadata: map[string]any{"something": "else"},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/metadata/name"},
			expectedMsgs:   []string{"'name' is required"},
		},
		{
			name:      "valid metadata with complete import structure",
			parseSpec: parseSpecWithLocalIDs("local-1"),
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
			name:      "import with missing workspace_id",
			parseSpec: noopParseSpec,
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
			name:      "import with missing local_id in resource",
			parseSpec: noopParseSpec,
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
			name:      "import with missing remote_id in resource",
			parseSpec: noopParseSpec,
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
		{
			name:      "import local_id missing from spec external IDs",
			parseSpec: parseSpecWithLocalIDs("other-id"),
			ctx: &rules.ValidationContext{
				Metadata: map[string]any{
					"name": "test-project",
					"import": map[string]any{
						"workspaces": []any{
							map[string]any{
								"workspace_id": "ws-123",
								"resources": []any{
									map[string]any{
										"local_id":  "missing-id",
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
			expectedMsgs:   []string{"local_id 'missing-id' from import metadata not found in spec"},
		},
		{
			name:      "all import local_ids present in spec",
			parseSpec: parseSpecWithLocalIDs("id1", "id2", "id3"),
			ctx: &rules.ValidationContext{
				Metadata: map[string]any{
					"name": "test-project",
					"import": map[string]any{
						"workspaces": []any{
							map[string]any{
								"workspace_id": "ws-123",
								"resources": []any{
									map[string]any{
										"local_id":  "id1",
										"remote_id": "remote-1",
									},
									map[string]any{
										"local_id":  "id2",
										"remote_id": "remote-2",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewMetadataSyntaxValidRule(tt.parseSpec, appliesToVersions)
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

	appliesToVersions := []string{
		specs.SpecVersionV0_1,
		specs.SpecVersionV0_1Variant,
	}
	rule := NewMetadataSyntaxValidRule(noopParseSpec, appliesToVersions)

	assert.Equal(t, "project/metadata-syntax-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "metadata syntax must be valid", rule.Description())
	expectedPatterns := make([]rules.MatchPattern, len(appliesToVersions))
	for i, v := range appliesToVersions {
		expectedPatterns[i] = rules.MatchPattern{Kind: "*", Version: v}
	}
	assert.Equal(t, expectedPatterns, rule.AppliesTo())

	examples := rule.Examples()
	assert.NotEmpty(t, examples.Valid)
	assert.NotEmpty(t, examples.Invalid)
}
