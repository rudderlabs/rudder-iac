package retlsource

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// validate reports the spec check and points at the command that does reach the
// warehouse. The whole two-line output is compared, so the wording cannot drift
// unnoticed — and an s3 source must not be pointed at preview, which exits 1 on
// it.
func TestReportValidation(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name             string
		externalID       string
		resourceType     string
		sourceDefinition string
		want             string
	}{
		{
			name:             "a warehouse source is pointed at preview",
			externalID:       "orders-model",
			resourceType:     "retl-source-sql-model",
			sourceDefinition: "postgres",
			want: "✅ retl-source-sql-model 'orders-model' is valid\n" +
				"   To check that its query runs against the warehouse: rudder-cli retl-sources preview orders-model\n",
		},
		{
			name:             "an s3 source is not",
			externalID:       "archive-bucket",
			resourceType:     "retl-source-table",
			sourceDefinition: "s3",
			want: "✅ retl-source-table 'archive-bucket' is valid\n" +
				"   It has no query to run, so there is nothing to preview.\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			reportValidation(&out, tc.externalID, tc.resourceType, resources.ResourceData{
				sqlmodel.SourceDefinitionKey: tc.sourceDefinition,
			})

			assert.Equal(t, tc.want, out.String())
		})
	}
}
