package retlsource

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

func TestReportValidation(t *testing.T) {
	t.Parallel()

	// The s3 error comes from the real table handler, so the test breaks if the
	// sentinel stops reaching the command.
	_, s3Err := table.NewHandler(nil, "retl").Preview(context.Background(), "events", resources.ResourceData{
		sqlmodel.AccountIDKey:        "acc-1",
		sqlmodel.SourceDefinitionKey: table.SourceDefinitionS3,
	}, 0)
	require.Error(t, s3Err)

	cases := []struct {
		name string
		err  error
		want string
	}{
		{"success", nil, "✅ SQL query executed successfully\n"},
		{"s3 table source", s3Err, "❌ Cannot validate this source: preview is not supported for s3 table sources\n"},
		{"failed query", errors.New("preview request failed: relation does not exist"), "❌ SQL query failed to execute: preview request failed: relation does not exist\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer

			reportValidation(&out, tc.err)

			assert.Equal(t, tc.want, out.String())
		})
	}
}
