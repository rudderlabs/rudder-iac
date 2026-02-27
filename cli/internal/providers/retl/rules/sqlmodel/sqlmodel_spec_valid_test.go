package sqlmodel

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr(s string) *string { return &s }

func TestSQLModelSpecSyntaxValidRule_Metadata(t *testing.T) {
	rule := NewSQLModelSpecSyntaxValidRule()

	assert.Equal(t, "retl/sqlmodel/spec-syntax-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "retl sql model spec syntax must be valid", rule.Description())
	assert.Equal(t, []rules.MatchPattern{rules.MatchKind("retl-source-sql-model")}, rule.AppliesTo())
}

func TestSQLModelSpecSyntaxValidRule_ValidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec sqlmodel.SQLModelSpec
	}{
		{
			name: "minimal with sql",
			spec: sqlmodel.SQLModelSpec{
				ID:               "model-1",
				DisplayName:      "My Model",
				AccountID:        "acc-1",
				PrimaryKey:       "id",
				SourceDefinition: "postgres",
				SQL:              ptr("SELECT * FROM users"),
			},
		},
		{
			name: "minimal with file",
			spec: sqlmodel.SQLModelSpec{
				ID:               "model-2",
				DisplayName:      "My Model",
				AccountID:        "acc-1",
				PrimaryKey:       "id",
				SourceDefinition: "snowflake",
				File:             ptr("./query.sql"),
			},
		},
		{
			name: "with description",
			spec: sqlmodel.SQLModelSpec{
				ID:               "model-3",
				DisplayName:      "My Model",
				Description:      "A test model",
				AccountID:        "acc-1",
				PrimaryKey:       "id",
				SourceDefinition: "bigquery",
				SQL:              ptr("SELECT 1"),
			},
		},
		{
			name: "with enabled=false",
			spec: sqlmodel.SQLModelSpec{
				ID:               "model-4",
				DisplayName:      "My Model",
				AccountID:        "acc-1",
				PrimaryKey:       "id",
				SourceDefinition: "redshift",
				SQL:              ptr("SELECT 1"),
				Enabled:          func() *bool { b := false; return &b }(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			results := validateSQLModelSpec("", "", nil, tt.spec)
			assert.Empty(t, results, "expected no validation errors")
		})
	}
}

func TestSQLModelSpecSyntaxValidRule_AllSourceDefinitions(t *testing.T) {
	t.Parallel()

	sourceDefinitions := []sqlmodel.SourceDefinition{
		"postgres", "redshift", "snowflake", "bigquery", "mysql", "databricks", "trino",
	}

	for _, sd := range sourceDefinitions {
		t.Run(string(sd), func(t *testing.T) {
			t.Parallel()
			spec := sqlmodel.SQLModelSpec{
				ID:               "model-1",
				DisplayName:      "My Model",
				AccountID:        "acc-1",
				PrimaryKey:       "id",
				SourceDefinition: sd,
				SQL:              ptr("SELECT 1"),
			}
			results := validateSQLModelSpec("", "", nil, spec)
			assert.Empty(t, results, "source definition %q should be valid", sd)
		})
	}
}

func TestSQLModelSpecSyntaxValidRule_InvalidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		spec         sqlmodel.SQLModelSpec
		wantMessages []string
	}{
		{
			name: "missing id",
			spec: sqlmodel.SQLModelSpec{
				DisplayName:      "My Model",
				AccountID:        "acc-1",
				PrimaryKey:       "id",
				SourceDefinition: "postgres",
				SQL:              ptr("SELECT 1"),
			},
			wantMessages: []string{"'id' is required"},
		},
		{
			name: "missing display_name",
			spec: sqlmodel.SQLModelSpec{
				ID:               "model-1",
				AccountID:        "acc-1",
				PrimaryKey:       "id",
				SourceDefinition: "postgres",
				SQL:              ptr("SELECT 1"),
			},
			wantMessages: []string{"'display_name' is required"},
		},
		{
			name: "missing account_id",
			spec: sqlmodel.SQLModelSpec{
				ID:               "model-1",
				DisplayName:      "My Model",
				PrimaryKey:       "id",
				SourceDefinition: "postgres",
				SQL:              ptr("SELECT 1"),
			},
			wantMessages: []string{"'account_id' is required"},
		},
		{
			name: "missing primary_key",
			spec: sqlmodel.SQLModelSpec{
				ID:               "model-1",
				DisplayName:      "My Model",
				AccountID:        "acc-1",
				SourceDefinition: "postgres",
				SQL:              ptr("SELECT 1"),
			},
			wantMessages: []string{"'primary_key' is required"},
		},
		{
			name: "missing source_definition",
			spec: sqlmodel.SQLModelSpec{
				ID:          "model-1",
				DisplayName: "My Model",
				AccountID:   "acc-1",
				PrimaryKey:  "id",
				SQL:         ptr("SELECT 1"),
			},
			wantMessages: []string{"'source_definition' is required"},
		},
		{
			name: "invalid source_definition",
			spec: sqlmodel.SQLModelSpec{
				ID:               "model-1",
				DisplayName:      "My Model",
				AccountID:        "acc-1",
				PrimaryKey:       "id",
				SourceDefinition: "oracle",
				SQL:              ptr("SELECT 1"),
			},
			wantMessages: []string{
				"'source_definition' must be one of [postgres redshift snowflake bigquery mysql databricks trino]",
			},
		},
		{
			name: "neither sql nor file",
			spec: sqlmodel.SQLModelSpec{
				ID:               "model-1",
				DisplayName:      "My Model",
				AccountID:        "acc-1",
				PrimaryKey:       "id",
				SourceDefinition: "postgres",
			},
			wantMessages: []string{"'sql' is required when 'file' is not specified"},
		},
		{
			name: "both sql and file",
			spec: sqlmodel.SQLModelSpec{
				ID:               "model-1",
				DisplayName:      "My Model",
				AccountID:        "acc-1",
				PrimaryKey:       "id",
				SourceDefinition: "postgres",
				SQL:              ptr("SELECT 1"),
				File:             ptr("./query.sql"),
			},
			wantMessages: []string{"'sql' and 'file' cannot be specified together"},
		},
		{
			name: "all required fields missing",
			spec: sqlmodel.SQLModelSpec{},
			wantMessages: []string{
				"'id' is required",
				"'display_name' is required",
				"'sql' is required when 'file' is not specified",
				"'account_id' is required",
				"'primary_key' is required",
				"'source_definition' is required",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			results := validateSQLModelSpec("", "", nil, tt.spec)
			require.Len(t, results, len(tt.wantMessages))

			var gotMessages []string
			for _, r := range results {
				gotMessages = append(gotMessages, r.Message)
			}
			assert.ElementsMatch(t, tt.wantMessages, gotMessages)
		})
	}
}
