package retlsource

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
)

func TestReportValidation(t *testing.T) {
	t.Parallel()

	queryFailed := errors.New("preview request failed: relation does not exist")

	cases := []struct {
		name string
		err  error
		want string
		// wantExit is the error the command exits with. A source with nothing to
		// validate must exit 0, so that a CI step covering every source in a
		// project does not fail once an s3 source joins it.
		wantExit error
	}{
		{
			name: "a query that ran is a pass",
			err:  nil,
			want: "✅ SQL query executed successfully\n",
		},
		{
			name:     "a source with no query to run is a pass, not a failure",
			err:      fmt.Errorf("%w for s3 table sources", table.ErrPreviewUnsupported),
			want:     "✅ Nothing to validate: preview is not supported for s3 table sources\n",
			wantExit: nil,
		},
		{
			name:     "a query that failed is a failure",
			err:      queryFailed,
			want:     "❌ SQL query failed to execute: preview request failed: relation does not exist\n",
			wantExit: queryFailed,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer

			exit := reportValidation(&out, tc.err)

			assert.Equal(t, tc.want, out.String())
			assert.Equal(t, tc.wantExit, exit)
		})
	}
}
