package table

import (
	"testing"

	"github.com/stretchr/testify/assert"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

func warehouseSpec() table.TableSpec {
	return table.TableSpec{
		ID:               "users-table",
		DisplayName:      "Users",
		AccountID:        "acc-1",
		SourceDefinition: "postgres",
		PrimaryKey:       "id",
		Schema:           "public",
		Table:            "users",
	}
}

func s3Spec() table.TableSpec {
	return table.TableSpec{
		ID:               "events-bucket",
		DisplayName:      "Events",
		AccountID:        "acc-s3",
		SourceDefinition: "s3",
		BucketName:       "events",
		ObjectPrefix:     "daily/",
	}
}

func TestTableSpecSyntaxValidRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewTableSpecSyntaxValidRule()

	assert.Equal(t, "retl/table/spec-syntax-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "retl table source spec syntax must be valid (experimental kind)", rule.Description())
	assert.Equal(t, prules.V1VersionPatterns(table.ResourceKind), rule.AppliesTo())
}

func TestTableSpecSyntaxValidRule_ValidSpecs(t *testing.T) {
	t.Parallel()

	cases := map[string]table.TableSpec{
		"postgres": warehouseSpec(),
		"s3":       s3Spec(),
	}
	for _, sd := range []string{"redshift", "snowflake", "bigquery", "mysql", "databricks", "trino"} {
		spec := warehouseSpec()
		spec.SourceDefinition = sd
		cases[sd] = spec
	}

	for name, spec := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, validateTableSpec("", "", nil, spec))
		})
	}
}

func TestTableSpecSyntaxValidRule_InvalidSpecs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		spec   func() table.TableSpec
		expect []rules.ValidationResult
	}{
		{
			name: "missing common fields",
			spec: func() table.TableSpec {
				s := warehouseSpec()
				s.DisplayName, s.AccountID = "", ""
				return s
			},
			expect: []rules.ValidationResult{
				{Reference: "/display_name", Message: "'display_name' is required"},
				{Reference: "/account_id", Message: "'account_id' is required"},
			},
		},
		{
			name: "unknown source definition",
			spec: func() table.TableSpec {
				s := warehouseSpec()
				s.SourceDefinition = "oracle"
				return s
			},
			expect: []rules.ValidationResult{{
				Reference: "/source_definition",
				Message:   "'source_definition' must be one of [postgres redshift snowflake bigquery mysql databricks trino s3]",
			}},
		},
		{
			name: "warehouse without primary key, schema and table",
			spec: func() table.TableSpec {
				s := warehouseSpec()
				s.PrimaryKey, s.Schema, s.Table = "", "", ""
				return s
			},
			expect: []rules.ValidationResult{
				{Reference: "/primary_key", Message: "'primary_key' is required unless 'source_definition' is s3"},
				{Reference: "/schema", Message: "'schema' is required unless 'source_definition' is s3"},
				{Reference: "/table", Message: "'table' is required unless 'source_definition' is s3"},
			},
		},
		{
			name: "warehouse with s3 fields",
			spec: func() table.TableSpec {
				s := warehouseSpec()
				s.BucketName, s.ObjectPrefix = "events", "daily/"
				return s
			},
			expect: []rules.ValidationResult{
				{Reference: "/bucket_name", Message: "'bucket_name' is not allowed unless 'source_definition' is s3"},
				{Reference: "/object_prefix", Message: "'object_prefix' is not allowed unless 'source_definition' is s3"},
			},
		},
		{
			// rudder-api rejects an s3 config without an object prefix.
			name: "s3 without object prefix",
			spec: func() table.TableSpec {
				s := s3Spec()
				s.ObjectPrefix = ""
				return s
			},
			expect: []rules.ValidationResult{
				{Reference: "/object_prefix", Message: "'object_prefix' is required when 'source_definition' is s3"},
			},
		},
		{
			name: "s3 without bucket",
			spec: func() table.TableSpec {
				s := s3Spec()
				s.BucketName = ""
				return s
			},
			expect: []rules.ValidationResult{
				{Reference: "/bucket_name", Message: "'bucket_name' is required when 'source_definition' is s3"},
			},
		},
		{
			// The s3 config the API client sends has no primary key, so an
			// accepted value would be dropped on every apply.
			name: "s3 with warehouse fields",
			spec: func() table.TableSpec {
				s := s3Spec()
				s.PrimaryKey, s.Schema, s.Table = "id", "public", "users"
				return s
			},
			expect: []rules.ValidationResult{
				{Reference: "/primary_key", Message: "'primary_key' is not allowed when 'source_definition' is s3"},
				{Reference: "/schema", Message: "'schema' is not allowed when 'source_definition' is s3"},
				{Reference: "/table", Message: "'table' is not allowed when 'source_definition' is s3"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ElementsMatch(t, tc.expect, validateTableSpec("", "", nil, tc.spec()))
		})
	}
}
