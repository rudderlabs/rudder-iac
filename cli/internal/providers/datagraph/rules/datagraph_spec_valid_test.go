package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
)

func extractRefs(results []rules.ValidationResult) []string {
	refs := make([]string, len(results))
	for i, r := range results {
		refs[i] = r.Reference
	}
	return refs
}

func extractMsgs(results []rules.ValidationResult) []string {
	msgs := make([]string, len(results))
	for i, r := range results {
		msgs[i] = r.Message
	}
	return msgs
}

func TestNewDataGraphSpecSyntaxValidRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewDataGraphSpecSyntaxValidRule()

	assert.Equal(t, "datagraph/data-graph/spec-syntax-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "data graph spec syntax must be valid", rule.Description())

	expectedPatterns := append(
		prules.LegacyVersionPatterns("data-graph"),
		prules.V1VersionPatterns("data-graph")...,
	)
	assert.Equal(t, expectedPatterns, rule.AppliesTo())

	examples := rule.Examples()
	assert.NotEmpty(t, examples.Valid)
	assert.NotEmpty(t, examples.Invalid)
}

func TestDataGraphSpecSyntaxValid_ValidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec dgModel.DataGraphSpec
	}{
		{
			name: "minimal data graph",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
			},
		},
		{
			name: "data graph with entity model",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{
						ID:          "user",
						DisplayName: "User",
						Type:        "entity",
						Table:       "db.schema.users",
						PrimaryID:   "user_id",
						Root:        true,
					},
				},
			},
		},
		{
			name: "data graph with event model",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{
						ID:          "purchase",
						DisplayName: "Purchase",
						Type:        "event",
						Table:       "db.schema.purchases",
						Timestamp:   "purchased_at",
					},
				},
			},
		},
		{
			name: "data graph with model and relationships",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{
						ID:          "user",
						DisplayName: "User",
						Type:        "entity",
						Table:       "db.schema.users",
						PrimaryID:   "user_id",
						Relationships: []dgModel.RelationshipSpec{
							{
								ID:            "user-account",
								DisplayName:   "User Account",
								Cardinality:   "one-to-one",
								Target:        "#data-graph-model:account",
								SourceJoinKey: "account_id",
								TargetJoinKey: "account_id",
							},
						},
					},
				},
			},
		},
		{
			name: "quoted table parts are valid",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{
						ID:          "user",
						DisplayName: "User",
						Type:        "entity",
						Table:       `"my db"."schema".users`,
						PrimaryID:   "user_id",
					},
				},
			},
		},
		{
			name: "empty models array",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models:    []dgModel.ModelSpec{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateDataGraphSpec("data-graph", specs.SpecVersionV1, nil, tt.spec)
			assert.Empty(t, results, "Valid spec should not produce errors")
		})
	}
}

func TestDataGraphSpecSyntaxValid_InvalidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		spec           dgModel.DataGraphSpec
		expectedErrors int
		expectedRefs   []string
		expectedMsgs   []string
	}{
		{
			name: "missing id",
			spec: dgModel.DataGraphSpec{
				AccountID: "wh-123",
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/id"},
			expectedMsgs:   []string{"'id' is required"},
		},
		{
			name: "missing account_id",
			spec: dgModel.DataGraphSpec{
				ID: "my-dg",
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/account_id"},
			expectedMsgs:   []string{"'account_id' is required"},
		},
		{
			name:           "missing both id and account_id",
			spec:           dgModel.DataGraphSpec{},
			expectedErrors: 2,
			expectedRefs:   []string{"/id", "/account_id"},
			expectedMsgs:   []string{"'id' is required", "'account_id' is required"},
		},
		{
			name: "model missing required fields",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{},
				},
			},
			expectedErrors: 4,
			expectedRefs:   []string{"/models/0/id", "/models/0/display_name", "/models/0/type", "/models/0/table"},
			expectedMsgs: []string{
				"'id' is required",
				"'display_name' is required",
				"'type' is required",
				"'table' is required",
			},
		},
		{
			name: "model with invalid type",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{
						ID:          "user",
						DisplayName: "User",
						Type:        "invalid",
						Table:       "db.schema.users",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/models/0/type"},
			expectedMsgs:   []string{"'type' must be one of [entity event]"},
		},
		{
			name: "model with invalid table format - 1 part",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{
						ID:          "user",
						DisplayName: "User",
						Type:        "entity",
						Table:       "users",
						PrimaryID:   "id",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/models/0/table"},
			expectedMsgs:   []string{"'table' must be a 3-part reference in the format catalog.schema.table"},
		},
		{
			name: "model with invalid table format - 2 parts",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{
						ID:          "user",
						DisplayName: "User",
						Type:        "entity",
						Table:       "schema.users",
						PrimaryID:   "id",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/models/0/table"},
			expectedMsgs:   []string{"'table' must be a 3-part reference in the format catalog.schema.table"},
		},
		{
			name: "model with invalid table format - 4 parts",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{
						ID:          "user",
						DisplayName: "User",
						Type:        "entity",
						Table:       "a.b.c.d",
						PrimaryID:   "id",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/models/0/table"},
			expectedMsgs:   []string{"'table' must be a 3-part reference in the format catalog.schema.table"},
		},
		{
			name: "model with invalid table format - empty part",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{
						ID:          "user",
						DisplayName: "User",
						Type:        "entity",
						Table:       ".schema.table",
						PrimaryID:   "id",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/models/0/table"},
			expectedMsgs:   []string{"'table' must be a 3-part reference in the format catalog.schema.table"},
		},
		{
			name: "entity model missing primary_id",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{
						ID:          "user",
						DisplayName: "User",
						Type:        "entity",
						Table:       "db.schema.users",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/models/0/primary_id"},
			expectedMsgs:   []string{"'primary_id' is required for entity models"},
		},
		{
			name: "event model missing timestamp",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{
						ID:          "purchase",
						DisplayName: "Purchase",
						Type:        "event",
						Table:       "db.schema.purchases",
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/models/0/timestamp"},
			expectedMsgs:   []string{"'timestamp' is required for event models"},
		},
		{
			name: "relationship missing required fields",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{
						ID:          "user",
						DisplayName: "User",
						Type:        "entity",
						Table:       "db.schema.users",
						PrimaryID:   "id",
						Relationships: []dgModel.RelationshipSpec{
							{},
						},
					},
				},
			},
			expectedErrors: 6,
			expectedRefs: []string{
				"/models/0/relationships/0/id",
				"/models/0/relationships/0/display_name",
				"/models/0/relationships/0/cardinality",
				"/models/0/relationships/0/target",
				"/models/0/relationships/0/source_join_key",
				"/models/0/relationships/0/target_join_key",
			},
			expectedMsgs: []string{
				"'id' is required",
				"'display_name' is required",
				"'cardinality' is required",
				"'target' is required",
				"'source_join_key' is required",
				"'target_join_key' is required",
			},
		},
		{
			name: "relationship with invalid cardinality",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{
						ID:          "user",
						DisplayName: "User",
						Type:        "entity",
						Table:       "db.schema.users",
						PrimaryID:   "id",
						Relationships: []dgModel.RelationshipSpec{
							{
								ID:            "user-account",
								DisplayName:   "User Account",
								Cardinality:   "invalid",
								Target:        "#data-graph-model:account",
								SourceJoinKey: "account_id",
								TargetJoinKey: "account_id",
							},
						},
					},
				},
			},
			expectedErrors: 1,
			expectedRefs:   []string{"/models/0/relationships/0/cardinality"},
			expectedMsgs:   []string{"'cardinality' must be one of [one-to-one one-to-many many-to-one]"},
		},
		{
			name: "errors across multiple models",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{
						ID:          "user",
						DisplayName: "User",
						Type:        "entity",
						Table:       "db.schema.users",
						// missing primary_id
					},
					{
						ID:          "purchase",
						DisplayName: "Purchase",
						Type:        "event",
						Table:       "db.schema.purchases",
						// missing timestamp
					},
				},
			},
			expectedErrors: 2,
			expectedRefs:   []string{"/models/0/primary_id", "/models/1/timestamp"},
			expectedMsgs: []string{
				"'primary_id' is required for entity models",
				"'timestamp' is required for event models",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateDataGraphSpec("data-graph", specs.SpecVersionV1, nil, tt.spec)

			assert.Len(t, results, tt.expectedErrors, "Unexpected number of validation errors")

			if tt.expectedErrors > 0 {
				assert.ElementsMatch(t, tt.expectedRefs, extractRefs(results), "Validation error references don't match")
				assert.ElementsMatch(t, tt.expectedMsgs, extractMsgs(results), "Validation error messages don't match")
			}
		})
	}
}
