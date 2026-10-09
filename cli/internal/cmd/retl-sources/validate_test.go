package retlsource

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

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
