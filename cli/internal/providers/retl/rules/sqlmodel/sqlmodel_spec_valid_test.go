package sqlmodel

import (
	"os"
	"path/filepath"
	"testing"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
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

	expectedPatterns := append(
		prules.LegacyVersionPatterns("retl-source-sql-model"),
		rules.MatchKindVersion(sqlmodel.ResourceKind, specs.SpecVersionV1),
	)
	assert.Equal(t, expectedPatterns, rule.AppliesTo())
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

func TestSQLModelV1SpecValidation_ValidSpecs(t *testing.T) {
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
			results := validateSQLModelV1Spec("", "", "", nil, tt.spec)
			assert.Empty(t, results, "expected no validation errors")
		})
	}
}

func TestSQLModelV1SpecValidation_AllSourceDefinitions(t *testing.T) {
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
			results := validateSQLModelV1Spec("", "", "", nil, spec)
			assert.Empty(t, results, "source definition %q should be valid", sd)
		})
	}
}

func TestSQLModelV1SpecValidation_InvalidSpecs(t *testing.T) {
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
			results := validateSQLModelV1Spec("", "", "", nil, tt.spec)
			require.Len(t, results, len(tt.wantMessages))

			var gotMessages []string
			for _, r := range results {
				gotMessages = append(gotMessages, r.Message)
			}
			assert.ElementsMatch(t, tt.wantMessages, gotMessages)
		})
	}
}

func TestSQLModelV1SpecValidation_FileResolution(t *testing.T) {
	t.Parallel()

	t.Run("valid file with SQL content", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		sqlFile := filepath.Join(tmpDir, "query.sql")
		err := os.WriteFile(sqlFile, []byte("SELECT * FROM users"), 0644)
		require.NoError(t, err)

		specFilePath := filepath.Join(tmpDir, "spec.yaml")

		spec := sqlmodel.SQLModelSpec{
			ID:               "model-1",
			DisplayName:      "My Model",
			AccountID:        "acc-1",
			PrimaryKey:       "id",
			SourceDefinition: "postgres",
			File:             ptr("query.sql"),
		}

		results := validateSQLModelV1Spec(specFilePath, "", "", nil, spec)
		assert.Empty(t, results, "expected no validation errors for valid SQL file")
	})

	t.Run("file with empty content", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		sqlFile := filepath.Join(tmpDir, "empty.sql")
		err := os.WriteFile(sqlFile, []byte(""), 0644)
		require.NoError(t, err)

		specFilePath := filepath.Join(tmpDir, "spec.yaml")

		spec := sqlmodel.SQLModelSpec{
			ID:               "model-1",
			DisplayName:      "My Model",
			AccountID:        "acc-1",
			PrimaryKey:       "id",
			SourceDefinition: "postgres",
			File:             ptr("empty.sql"),
		}

		results := validateSQLModelV1Spec(specFilePath, "", "", nil, spec)
		require.Len(t, results, 1)
		assert.Equal(t, "/file", results[0].Reference)
		assert.Equal(t, "'sql' content is empty after resolving 'file'", results[0].Message)
	})

	t.Run("file with only whitespace", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		sqlFile := filepath.Join(tmpDir, "whitespace.sql")
		err := os.WriteFile(sqlFile, []byte("   \n\t  \n  "), 0644)
		require.NoError(t, err)

		specFilePath := filepath.Join(tmpDir, "spec.yaml")

		spec := sqlmodel.SQLModelSpec{
			ID:               "model-1",
			DisplayName:      "My Model",
			AccountID:        "acc-1",
			PrimaryKey:       "id",
			SourceDefinition: "postgres",
			File:             ptr("whitespace.sql"),
		}

		results := validateSQLModelV1Spec(specFilePath, "", "", nil, spec)
		require.Len(t, results, 1)
		assert.Equal(t, "/file", results[0].Reference)
		assert.Equal(t, "'sql' content is empty after resolving 'file'", results[0].Message)
	})

	t.Run("file does not exist", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		specFilePath := filepath.Join(tmpDir, "spec.yaml")

		spec := sqlmodel.SQLModelSpec{
			ID:               "model-1",
			DisplayName:      "My Model",
			AccountID:        "acc-1",
			PrimaryKey:       "id",
			SourceDefinition: "postgres",
			File:             ptr("nonexistent.sql"),
		}

		results := validateSQLModelV1Spec(specFilePath, "", "", nil, spec)
		require.Len(t, results, 1)
		assert.Equal(t, "/file", results[0].Reference)
		assert.Contains(t, results[0].Message, "failed to read sql file")
	})

	t.Run("relative file path resolution", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()

		subDir := filepath.Join(tmpDir, "queries")
		err := os.MkdirAll(subDir, 0755)
		require.NoError(t, err)

		sqlFile := filepath.Join(subDir, "query.sql")
		err = os.WriteFile(sqlFile, []byte("SELECT 1"), 0644)
		require.NoError(t, err)

		specFilePath := filepath.Join(tmpDir, "spec.yaml")

		spec := sqlmodel.SQLModelSpec{
			ID:               "model-1",
			DisplayName:      "My Model",
			AccountID:        "acc-1",
			PrimaryKey:       "id",
			SourceDefinition: "postgres",
			File:             ptr("queries/query.sql"),
		}

		results := validateSQLModelV1Spec(specFilePath, "", "", nil, spec)
		assert.Empty(t, results, "expected no validation errors for relative file path")
	})

	t.Run("parent directory relative path", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()

		sqlDir := filepath.Join(tmpDir, "sql")
		specDir := filepath.Join(tmpDir, "specs")
		err := os.MkdirAll(sqlDir, 0755)
		require.NoError(t, err)
		err = os.MkdirAll(specDir, 0755)
		require.NoError(t, err)

		sqlFile := filepath.Join(sqlDir, "query.sql")
		err = os.WriteFile(sqlFile, []byte("SELECT * FROM orders"), 0644)
		require.NoError(t, err)

		specFilePath := filepath.Join(specDir, "spec.yaml")

		spec := sqlmodel.SQLModelSpec{
			ID:               "model-1",
			DisplayName:      "My Model",
			AccountID:        "acc-1",
			PrimaryKey:       "id",
			SourceDefinition: "postgres",
			File:             ptr("../sql/query.sql"),
		}

		results := validateSQLModelV1Spec(specFilePath, "", "", nil, spec)
		assert.Empty(t, results, "expected no validation errors for parent directory path")
	})

	t.Run("absolute file path", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		sqlFile := filepath.Join(tmpDir, "absolute.sql")
		err := os.WriteFile(sqlFile, []byte("SELECT COUNT(*) FROM events"), 0644)
		require.NoError(t, err)

		specFilePath := filepath.Join(tmpDir, "spec.yaml")

		spec := sqlmodel.SQLModelSpec{
			ID:               "model-1",
			DisplayName:      "My Model",
			AccountID:        "acc-1",
			PrimaryKey:       "id",
			SourceDefinition: "postgres",
			File:             ptr(sqlFile),
		}

		results := validateSQLModelV1Spec(specFilePath, "", "", nil, spec)
		assert.Empty(t, results, "expected no validation errors for absolute file path")
	})

	t.Run("sql specified takes precedence - no file check", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		specFilePath := filepath.Join(tmpDir, "spec.yaml")

		spec := sqlmodel.SQLModelSpec{
			ID:               "model-1",
			DisplayName:      "My Model",
			AccountID:        "acc-1",
			PrimaryKey:       "id",
			SourceDefinition: "postgres",
			SQL:              ptr("SELECT 1"),
		}

		results := validateSQLModelV1Spec(specFilePath, "", "", nil, spec)
		assert.Empty(t, results, "expected no validation errors when sql is specified")
	})
}
