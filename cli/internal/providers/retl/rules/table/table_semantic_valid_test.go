package table

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/accounts"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

func TestTableSemanticValidRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewTableSemanticValidRule()

	assert.Equal(t, "retl/table/semantic-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, prules.V1VersionPatterns(table.ResourceKind), rule.AppliesTo())
}

// The rule runs through the typed-rule JSON round-trip, so these cases also
// prove the raw spec's account and source_definition reach the check.
func TestTableSemanticValidRule_AccountReference(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource("prod-pg", accounts.AccountResourceType, resources.ResourceData{}, nil,
		resources.WithRawData(&accounts.AccountResource{ID: "prod-pg", AccountDefinitionName: "SOURCE_POSTGRES"})))

	warehouse := map[string]any{
		"id":                "users-table",
		"display_name":      "Users",
		"source_definition": "postgres",
		"primary_key":       "id",
		"schema":            "public",
		"table":             "users",
	}
	s3 := map[string]any{
		"id":                "events-bucket",
		"display_name":      "Events",
		"source_definition": "s3",
		"bucket_name":       "events",
	}
	with := func(spec map[string]any, key, value string) map[string]any {
		out := map[string]any{key: value}
		for k, v := range spec {
			out[k] = v
		}
		return out
	}

	tests := []struct {
		name    string
		spec    map[string]any
		wantErr string
	}{
		{
			name: "warehouse source referencing an account of its source definition",
			spec: with(warehouse, "account", "#account:prod-pg"),
		},
		{
			name:    "warehouse source referencing an account missing from the project",
			spec:    with(warehouse, "account", "#account:prod-sf"),
			wantErr: "account 'prod-sf' not found in the project",
		},
		{
			// s3 accounts carry no account definition the accounts provider can
			// manage, so every project account is a warehouse account and can
			// never back an s3 source.
			name:    "s3 source referencing a warehouse account",
			spec:    with(s3, "account", "#account:prod-pg"),
			wantErr: "account 'prod-pg' is a 'postgres' account (SOURCE_POSTGRES) and cannot back source_definition 's3'",
		},
		{
			name: "s3 source naming its account by id",
			spec: with(s3, "account_id", "acc-s3"),
		},
	}

	rule := NewTableSemanticValidRule()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			results := rule.Validate(&rules.ValidationContext{
				Kind:    table.ResourceKind,
				Version: specs.SpecVersionV1,
				Spec:    tt.spec,
				Graph:   graph,
			})

			if tt.wantErr == "" {
				assert.Empty(t, results)
				return
			}
			require.Len(t, results, 1)
			assert.Equal(t, "/spec/account", results[0].Reference)
			assert.Contains(t, results[0].Message, tt.wantErr)
		})
	}
}

// Table and SQL model sources share one namespace in the control plane, which
// compares names case-insensitively. A clash between two SQL models is
// retl/sqlmodel/semantic-valid's to report.
func TestTableSemanticValidRule_DisplayNameUniqueness(t *testing.T) {
	t.Parallel()

	source := func(resourceType, id, displayName string) *resources.Resource {
		return resources.NewResource(id, resourceType, resources.ResourceData{sqlmodel.DisplayNameKey: displayName}, nil)
	}
	spec := map[string]any{
		"id":                "users-table",
		"display_name":      "users",
		"account_id":        "acc-1",
		"source_definition": "postgres",
		"primary_key":       "id",
		"schema":            "public",
		"table":             "users",
	}

	tests := []struct {
		name   string
		others []*resources.Resource
		want   []rules.ValidationResult
	}{
		{
			name: "unique across RETL sources",
			others: []*resources.Resource{
				source(sqlmodel.ResourceType, "orders", "Orders"),
				source(table.ResourceType, "events", "Events"),
			},
		},
		{
			name:   "SQL model whose display name differs only in case",
			others: []*resources.Resource{source(sqlmodel.ResourceType, "users-model", "Users")},
			want: []rules.ValidationResult{{
				Reference: "/spec/display_name",
				Message:   "duplicate display_name 'users' (case-insensitive match with 'Users') within kinds 'retl-source-sql-model' and 'retl-source-table'",
			}},
		},
		{
			name:   "table source with the same display name",
			others: []*resources.Resource{source(table.ResourceType, "users-copy", "users")},
			want: []rules.ValidationResult{{
				Reference: "/spec/display_name",
				Message:   "duplicate display_name 'users' within kinds 'retl-source-sql-model' and 'retl-source-table'",
			}},
		},
		{
			name: "clashes with several sources reported once",
			others: []*resources.Resource{
				source(sqlmodel.ResourceType, "users-model", "USERS"),
				source(table.ResourceType, "users-copy", "Users"),
			},
			want: []rules.ValidationResult{{
				Reference: "/spec/display_name",
				Message:   "duplicate display_name 'users' (case-insensitive match with 'USERS', 'Users') within kinds 'retl-source-sql-model' and 'retl-source-table'",
			}},
		},
	}

	rule := NewTableSemanticValidRule()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			graph := resources.NewGraph()
			graph.AddResource(source(table.ResourceType, "users-table", "users"))
			for _, r := range tt.others {
				graph.AddResource(r)
			}

			results := rule.Validate(&rules.ValidationContext{
				Kind:    table.ResourceKind,
				Version: specs.SpecVersionV1,
				Spec:    spec,
				Graph:   graph,
			})

			assert.Equal(t, tt.want, results)
		})
	}
}
