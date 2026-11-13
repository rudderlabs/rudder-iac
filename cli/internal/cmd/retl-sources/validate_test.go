package retlsource

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateWithoutWorkspaceAPI(t *testing.T) {
	t.Setenv("RUDDERSTACK_ACCESS_TOKEN", "")
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))

	location := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(location, "source.yaml"), []byte("version: rudder/v1\nkind: retl-source-table\nmetadata:\n  name: customers-table\nspec:\n  id: customers-table\n  display_name: Customers Table\n  account: '#account:retl-pg'\n  primary_key: id\n  source_definition: postgres\n  schema: analytics\n  table: customers\n  enabled: true\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(location, "account.yaml"), []byte("version: rudder/v1\nkind: account\nmetadata:\n  name: retl-pg\nspec:\n  id: retl-pg\n  name: RETL Postgres\n  account_definition_name: SOURCE_POSTGRES\n  config:\n    host: db.example.com\n    dbname: analytics\n    user: rudder\n    port: 5432\n    sslMode: disable\n    password: test\n"), 0o600))
	cmd := newCmdValidate()
	require.NoError(t, cmd.Flags().Set("location", location))
	var out bytes.Buffer
	cmd.SetOut(&out)
	err := cmd.RunE(cmd, []string{"customers-table"})
	require.NoError(t, err)
	assert.Contains(t, out.String(), "is valid")
}

func TestNewCmdValidateAcceptsVarFile(t *testing.T) {
	t.Parallel()

	flag := newCmdValidate().Flags().Lookup("var-file")
	require.NotNil(t, flag, "retl-sources validate must accept --var-file")
	assert.Equal(t, "stringArray", flag.Value.Type())
}

// validate reports the spec check and points at the command that does reach the
// warehouse. The whole two-line output is compared, so the wording cannot drift
// unnoticed — and an s3 source must not be pointed at preview, which exits 1 on
// it.
func TestReportValidation(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name             string
		externalID       string
		location         string
		resourceType     string
		sourceDefinition string
		varFiles         []string
		want             string
	}{
		{
			name:             "a warehouse source is pointed at preview",
			externalID:       "orders-model",
			location:         ".",
			resourceType:     "retl-source-sql-model",
			sourceDefinition: "postgres",
			want: "✅ retl-source-sql-model 'orders-model' is valid\n" +
				"   To check that its query runs against the warehouse: rudder-cli retl-sources preview orders-model\n",
		},
		{
			name:             "a warehouse source outside the working directory keeps its location in the hint",
			externalID:       "orders-model",
			location:         "./project",
			resourceType:     "retl-source-sql-model",
			sourceDefinition: "postgres",
			want: "✅ retl-source-sql-model 'orders-model' is valid\n" +
				"   To check that its query runs against the warehouse: rudder-cli retl-sources preview orders-model --location ./project\n",
		},
		{
			name:             "a warehouse source keeps var files in the preview hint",
			externalID:       "orders model",
			location:         "./project folder",
			resourceType:     "retl-source-sql-model",
			sourceDefinition: "postgres",
			varFiles:         []string{"base.vars.yaml", "prod vars.vars.yml", "team's.vars.yaml"},
			want: "✅ retl-source-sql-model 'orders model' is valid\n" +
				"   To check that its query runs against the warehouse: rudder-cli retl-sources preview 'orders model' --location './project folder' --var-file base.vars.yaml --var-file 'prod vars.vars.yml' --var-file 'team'\\''s.vars.yaml'\n",
		},
		{
			name:             "an s3 source is told there is nothing to preview",
			externalID:       "archive-bucket",
			location:         "./project",
			resourceType:     "retl-source-table",
			sourceDefinition: "s3",
			varFiles:         []string{"prod.vars.yaml"},
			want: "✅ retl-source-table 'archive-bucket' is valid\n" +
				"   It has no query to run, so there is nothing to preview.\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			reportValidation(&out, tc.externalID, tc.location, tc.varFiles, tc.resourceType, resources.ResourceData{
				sqlmodel.SourceDefinitionKey: tc.sourceDefinition,
			})

			assert.Equal(t, tc.want, out.String())
		})
	}
}
