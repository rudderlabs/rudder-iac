package retlsource

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

// validate no longer runs the source's query, so its output must not imply that
// the warehouse was reached. The old command printed "Query executed
// successfully"; anyone reading the new line has to be told where that check
// went, or a green validate reads as a green warehouse.
func TestReportValidation(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	reportValidation(&out, "orders-model", "retl-source-sql-model")

	got := out.String()
	assert.Contains(t, got, "retl-source-sql-model 'orders-model' is valid")
	assert.Contains(t, got, "rudder-cli retl-sources preview orders-model",
		"a user who relied on validate running the query has to be told where it went")
	assert.NotContains(t, got, "Query executed",
		"validate must not claim to have run a query it no longer runs")
}
