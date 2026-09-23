package retlsource

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// validate no longer runs the source's query, so its output must not imply that
// the warehouse was reached. The old command printed "Query executed
// successfully"; anyone reading the new line has to be told where that check
// went, or a green validate reads as a green warehouse.
func TestReportValidation(t *testing.T) {
	t.Parallel()

	t.Run("points a warehouse source at preview", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer
		reportValidation(&out, "orders-model", "retl-source-sql-model", resources.ResourceData{
			sqlmodel.SourceDefinitionKey: "postgres",
		})

		got := out.String()
		assert.Contains(t, got, "retl-source-sql-model 'orders-model' is valid")
		assert.Contains(t, got, "rudder-cli retl-sources preview orders-model",
			"a user who relied on validate running the query has to be told where it went")
		assert.NotContains(t, got, "Query executed",
			"validate must not claim to have run a query it no longer runs")
	})

	// An s3 table source has no query, so preview exits 1 on it. Sending the
	// reader there would be worse than saying nothing.
	t.Run("does not send an s3 source to a command that refuses it", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer
		reportValidation(&out, "archive-bucket", "retl-source-table", resources.ResourceData{
			sqlmodel.SourceDefinitionKey: "s3",
		})

		got := out.String()
		assert.Contains(t, got, "retl-source-table 'archive-bucket' is valid")
		assert.Contains(t, got, "nothing to preview")
		assert.NotContains(t, got, "retl-sources preview",
			"preview exits 1 for s3, so the success line must not point at it")
	})
}
